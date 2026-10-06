package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// defaultConfigItemCreateTimeout is how long Create waits for HA's asynchronous reload to create
// the entity, unless `timeouts.create` says otherwise.
const defaultConfigItemCreateTimeout = 60 * time.Second

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
	ID       types.String   `tfsdk:"id"`
	Config   dyntype.Value  `tfsdk:"config"`
	EntityID types.String   `tfsdk:"entity_id"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *configItemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.spec.domain
}

func (r *configItemResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
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
	timeout, diags := plan.Timeouts.Create(ctx, defaultConfigItemCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.create(ctx, &plan, timeout, resp.Private, &resp.Diagnostics) {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	}
}

// create creates the item in plan and sets its entity ID, waiting at most timeout for it. It
// reports problems in diags and returns whether the item was saved, and so belongs in state. An
// item saved without an entity goes into state with a null `entity_id` and an error, so it is
// tainted and replaced, not orphaned.
func (r *configItemResource) create(ctx context.Context, plan *configItemModel, timeout time.Duration, private privateSetter, diags *diag.Diagnostics) bool {
	id := plan.ID.ValueString()
	// Saving replaces an existing item with this ID, so fail instead (ADR-0024).
	_, err := r.items.Get(ctx, id)
	switch {
	case errors.Is(err, client.ErrNotFound):
	case err != nil:
		diags.AddError("Checking for an existing "+r.spec.domain, client.ErrorDetail(err))
		return false
	default:
		diags.AddAttributeError(path.Root("id"), withArticle(r.spec.domain)+" with this ID already exists",
			fmt.Sprintf("Home Assistant already has %s with the ID %q. Either import it "+
				"(`tofu import homeassistant_%s.<name> %s`), or choose another `id`.",
				withArticle(r.spec.domain), id, r.spec.domain, id))
		return false
	}

	if !r.save(ctx, *plan, private, diags) {
		return false
	}

	entityID, err := r.awaitEntity(ctx, id, timeout)
	if err != nil {
		plan.EntityID = types.StringNull()
		diags.AddError("Finding the "+r.spec.domain+"'s entity", client.ErrorDetail(err))
		return true
	}
	plan.EntityID = types.StringValue(entityID)
	return true
}

// awaitEntity polls until HA's reload has created the item's entity, for at most timeout.
func (r *configItemResource) awaitEntity(ctx context.Context, id string, timeout time.Duration) (string, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var entityID string
	var lastErr error
	err := waitFor(waitCtx, func(ctx context.Context) (bool, error) {
		var err error
		entityID, err = r.items.FindEntity(ctx, id, "")
		if errors.Is(err, client.ErrEntityNotFound) {
			lastErr = err
			return false, nil
		}
		return err == nil, err
	})
	if err != nil && errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil && lastErr != nil {
		return "", fmt.Errorf("%w within %s.\n\nThe %s was saved, but Home Assistant has not "+
			"reloaded it yet. If it needs longer, raise `timeouts.create`", lastErr, timeout, r.spec.domain)
	}
	return entityID, err
}

func (r *configItemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state configItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.read(ctx, &state, req.Private, &resp.Diagnostics) {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// read refreshes state from HA. It reports problems in diags and returns false if the item is
// gone.
func (r *configItemResource) read(ctx context.Context, state *configItemModel, private privateGetter, diags *diag.Diagnostics) bool {
	id := state.ID.ValueString()
	raw, err := r.items.Get(ctx, id)
	if errors.Is(err, client.ErrNotFound) {
		return false
	}
	if err != nil {
		diags.AddError("Reading "+r.spec.domain, client.ErrorDetail(err))
		return true
	}
	baseline, d := storedBaseline(ctx, private)
	diags.Append(d...)
	if diags.HasError() {
		return true
	}
	config, err := refreshedConfig(state.Config, raw, baseline)
	if err != nil {
		diags.AddError("Reading "+r.spec.domain, err.Error())
		return true
	}
	state.Config = config

	if entityID, ok := r.items.FixedEntityID(id); ok {
		state.EntityID = types.StringValue(entityID)
		return true
	}
	entityID, err := r.items.FindEntity(ctx, id, state.EntityID.ValueString())
	switch {
	case errors.Is(err, client.ErrEntityNotFound):
		// A reload may still be running; keep what state has.
	case err != nil:
		diags.AddError("Finding the "+r.spec.domain+"'s entity", client.ErrorDetail(err))
	default:
		state.EntityID = types.StringValue(entityID)
	}
	return true
}

func (r *configItemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan configItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The entity ID stays the same, so there is no reload to wait for.
	if !r.save(ctx, plan, resp.Private, &resp.Diagnostics) {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// save sends the planned config to HA, then reads it back and stores it as the baseline
// (ADR-0023). It reports problems in diags and returns whether the config was saved. Failing to
// read it back is only a warning: without a baseline, Read falls back to semantic equality.
func (r *configItemResource) save(ctx context.Context, plan configItemModel, private privateSetter, diags *diag.Diagnostics) bool {
	config, err := plan.Config.JSON()
	if err != nil {
		diags.AddError("Invalid "+r.spec.domain+" config", err.Error())
		return false
	}
	id := plan.ID.ValueString()
	if err := r.items.Save(ctx, id, config); err != nil {
		addSaveError(diags, "Saving "+r.spec.domain, err)
		return false
	}
	stored, err := r.items.Get(ctx, id)
	if err != nil {
		diags.AddWarning("Reading the "+r.spec.domain+" back",
			client.ErrorDetail(err)+"\n\nThe "+r.spec.domain+" was saved. Until the next apply, "+
				"keys that Home Assistant renamed on save may show as a difference.")
		return true
	}
	diags.Append(storeBaseline(ctx, private, stored)...)
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
