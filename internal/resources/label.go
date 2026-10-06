package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = (*labelResource)(nil)
	_ resource.ResourceWithImportState = (*labelResource)(nil)
)

// NewLabel returns the `homeassistant_label` resource.
func NewLabel() resource.Resource {
	return &labelResource{}
}

type labelResource struct {
	labels client.Registry[client.Label]
}

type labelModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Color       types.String `tfsdk:"color"`
	Icon        types.String `tfsdk:"icon"`
	Description types.String `tfsdk:"description"`
}

func (r *labelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_label"
}

func (r *labelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A label: a tag that can be applied to areas, devices, entities, automations, and more.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Label ID, a slug such as `managed_by_tofu`. Home Assistant derives it from " +
					"`name` when the label is created, and keeps it when the label is renamed. Set it to " +
					"choose the ID yourself. Changing it replaces the label. Creation fails if the ID is " +
					"taken. Import with this ID.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{slugValidator{}},
			},
			"name": schema.StringAttribute{
				Description: "Display name. Changing it renames the label in place.",
				Required:    true,
			},
			"color": schema.StringAttribute{
				Description: "Color, either `#RRGGBB` or a theme color such as `red` or `indigo`.",
				Optional:    true,
			},
			"icon": schema.StringAttribute{
				Description: "Icon, e.g. `mdi:tag`.",
				Optional:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description.",
				Optional:    true,
			},
		},
	}
}

func (r *labelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c := haClient(req, resp)
	if c == nil {
		return
	}
	r.labels = client.Labels(c.WS)
}

// fields are the label's settable fields other than name. Unset ones are sent as null, which
// clears them.
func (m labelModel) fields() map[string]any {
	return map[string]any{
		"color":       m.Color.ValueStringPointer(),
		"icon":        m.Icon.ValueStringPointer(),
		"description": m.Description.ValueStringPointer(),
	}
}

func (r *labelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan labelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	g := generatedID{
		kind:     "label",
		typeName: "homeassistant_label",
		create: func(ctx context.Context, name string) (string, error) {
			fields := withoutNulls(plan.fields())
			fields["name"] = name
			l, err := r.labels.Create(ctx, fields)
			return l.LabelID, err
		},
		rename: func(ctx context.Context, id, name string) error {
			_, err := r.labels.Update(ctx, id, map[string]any{"name": name})
			return err
		},
		delete: r.labels.Delete,
	}
	id, err := g.Create(ctx, plan.Name.ValueString(), plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Creating label", client.ErrorDetail(err))
		return
	}

	plan.ID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *labelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state labelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	l, ok, err := r.labels.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading label", client.ErrorDetail(err))
		return
	}
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, labelModel{
		ID:          types.StringValue(l.LabelID),
		Name:        types.StringValue(l.Name),
		Color:       types.StringPointerValue(l.Color),
		Icon:        types.StringPointerValue(l.Icon),
		Description: types.StringPointerValue(l.Description),
	})...)
}

func (r *labelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan labelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fields := plan.fields()
	fields["name"] = plan.Name.ValueString()
	if _, err := r.labels.Update(ctx, plan.ID.ValueString(), fields); err != nil {
		resp.Diagnostics.AddError("Updating label", client.ErrorDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *labelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state labelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.labels.Delete(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Deleting label", client.ErrorDetail(err))
	}
}

func (r *labelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
