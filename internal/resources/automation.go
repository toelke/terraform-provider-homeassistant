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

var (
	_ resource.ResourceWithConfigure   = (*automationResource)(nil)
	_ resource.ResourceWithImportState = (*automationResource)(nil)
)

// NewAutomation returns the `homeassistant_automation` resource.
func NewAutomation() resource.Resource {
	return &automationResource{}
}

type automationResource struct {
	automations client.ConfigItems
}

type automationModel struct {
	ID       types.String  `tfsdk:"id"`
	Config   dyntype.Value `tfsdk:"config"`
	EntityID types.String  `tfsdk:"entity_id"`
}

func (r *automationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automation"
}

func (r *automationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An automation, stored in `automations.yaml` like one made in the automation " +
			"editor. Whether it is enabled is runtime state and not managed; to disable it as " +
			"code, set `initial_state = false` inside `config`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Config ID of the automation, e.g. `motion_bedroom`: lowercase letters, " +
					"digits, and underscores. Changing it replaces the automation. Import with this ID.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{configItemIDValidator{}},
			},
			"config": schema.DynamicAttribute{
				Description: "The automation (`alias`, `description`, `mode`, `triggers`, " +
					"`conditions`, `actions`, …), as an HCL object or " +
					"`yamldecode(file(\"automation.yaml\"))`. YAML copied from the automation " +
					"editor works unchanged once its `id` line is removed. Home Assistant validates " +
					"it on apply.",
				CustomType: dyntype.Type{},
				Required:   true,
				Validators: []validator.Dynamic{configItemConfigValidator{}},
			},
			"entity_id": schema.StringAttribute{
				Description: "Entity ID of the automation, e.g. `automation.bedroom_lights_on_motion`. " +
					"Home Assistant derives it from the alias when the automation is first created, " +
					"and keeps it afterwards.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *automationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.HAClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.HAClient, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.automations = client.NewAutomations(c.REST)
}

func (r *automationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan automationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.save(ctx, plan, resp.Diagnostics.AddError) {
		return
	}

	entityID, err := r.automations.FindEntity(ctx, plan.ID.ValueString(), entityWait)
	if err != nil {
		// Saved but not found: keep it in state, so it is tainted and replaced, not orphaned.
		plan.EntityID = types.StringNull()
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Finding the automation's entity", client.ErrorDetail(err))
		return
	}
	plan.EntityID = types.StringValue(entityID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *automationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state automationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	raw, err := r.automations.Get(ctx, id)
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading automation", client.ErrorDetail(err))
		return
	}
	config, err := dyntype.FromJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Reading automation", err.Error())
		return
	}

	entityID, err := r.automations.FindEntity(ctx, id, 0)
	switch {
	case errors.Is(err, client.ErrEntityNotFound):
		// A reload may still be running; keep what state has.
	case err != nil:
		resp.Diagnostics.AddError("Finding the automation's entity", client.ErrorDetail(err))
		return
	default:
		state.EntityID = types.StringValue(entityID)
	}

	state.Config = config
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *automationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan automationModel
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
func (r *automationResource) save(ctx context.Context, plan automationModel, addError func(string, string)) bool {
	config, err := plan.Config.JSON()
	if err != nil {
		addError("Invalid automation config", err.Error())
		return false
	}
	if err := r.automations.Save(ctx, plan.ID.ValueString(), config); err != nil {
		addError("Saving automation", client.ErrorDetail(err))
		return false
	}
	return true
}

func (r *automationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state automationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.automations.Delete(ctx, state.ID.ValueString())
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Deleting automation", client.ErrorDetail(err))
	}
}

func (r *automationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
