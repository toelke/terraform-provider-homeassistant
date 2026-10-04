package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

// entityWait bounds how long Create waits for HA's asynchronous reload to create the entity.
const entityWait = time.Minute

// configItemSpec describes one REST config editor domain: automation, script, or scene. Each
// shares the generic implementation in configItemResource.
type configItemSpec struct {
	// domain is the domain, e.g. "automation". It is also the noun in diagnostics.
	domain string
	// newItems returns the domain's config editor.
	newItems func(*client.RESTClient) client.ConfigItems
	// description, idDescription, configDescription, and entityIDDescription describe the
	// resource and its attributes.
	description         string
	idDescription       string
	configDescription   string
	entityIDDescription string
}

var (
	_ resource.ResourceWithConfigure   = (*configItemResource)(nil)
	_ resource.ResourceWithImportState = (*configItemResource)(nil)
)

// configItemResource manages the items of one REST config editor.
type configItemResource struct {
	spec  configItemSpec
	items client.ConfigItems
}

type configItemModel struct {
	ID       types.String  `tfsdk:"id"`
	Config   dyntype.Value `tfsdk:"config"`
	EntityID types.String  `tfsdk:"entity_id"`
}

func (r *configItemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.spec.domain
}

func (r *configItemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: r.spec.description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: r.spec.idDescription,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{configItemIDValidator{}},
			},
			"config": schema.DynamicAttribute{
				Description: r.spec.configDescription,
				CustomType:  dyntype.Type{},
				Required:    true,
				Validators:  []validator.Dynamic{configItemConfigValidator{}},
			},
			"entity_id": schema.StringAttribute{
				Description: r.spec.entityIDDescription,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *configItemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.HAClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.HAClient, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.items = r.spec.newItems(c.REST)
}

func (r *configItemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan configItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.save(ctx, plan, resp.Diagnostics.AddError) {
		return
	}

	entityID, err := r.items.FindEntity(ctx, plan.ID.ValueString(), entityWait)
	if err != nil {
		// Saved but not found: keep it in state, so it is tainted and replaced, not orphaned.
		plan.EntityID = types.StringNull()
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Finding the "+r.spec.domain+"'s entity", client.ErrorDetail(err))
		return
	}
	plan.EntityID = types.StringValue(entityID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *configItemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state configItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	raw, err := r.items.Get(ctx, id)
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading "+r.spec.domain, client.ErrorDetail(err))
		return
	}
	config, err := dyntype.FromJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Reading "+r.spec.domain, err.Error())
		return
	}

	entityID, err := r.items.FindEntity(ctx, id, 0)
	switch {
	case errors.Is(err, client.ErrEntityNotFound):
		// A reload may still be running; keep what state has.
	case err != nil:
		resp.Diagnostics.AddError("Finding the "+r.spec.domain+"'s entity", client.ErrorDetail(err))
		return
	default:
		state.EntityID = types.StringValue(entityID)
	}

	state.Config = config
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *configItemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan configItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The entity ID stays the same, so there is no reload to wait for.
	if !r.save(ctx, plan, resp.Diagnostics.AddError) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// save sends the planned config to HA. It reports failures through addError and returns whether
// it succeeded.
func (r *configItemResource) save(ctx context.Context, plan configItemModel, addError func(string, string)) bool {
	config, err := plan.Config.JSON()
	if err != nil {
		addError("Invalid "+r.spec.domain+" config", err.Error())
		return false
	}
	if err := r.items.Save(ctx, plan.ID.ValueString(), config); err != nil {
		addError("Saving "+r.spec.domain, client.ErrorDetail(err))
		return false
	}
	return true
}

func (r *configItemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state configItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.items.Delete(ctx, state.ID.ValueString())
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Deleting "+r.spec.domain, client.ErrorDetail(err))
	}
}

func (r *configItemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
