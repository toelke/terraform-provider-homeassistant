package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = (*areaResource)(nil)
	_ resource.ResourceWithImportState = (*areaResource)(nil)
)

// NewArea returns the `homeassistant_area` resource.
func NewArea() resource.Resource {
	return &areaResource{}
}

type areaResource struct {
	areas client.Registry[client.Area]
}

type areaModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	FloorID             types.String `tfsdk:"floor_id"`
	Icon                types.String `tfsdk:"icon"`
	Aliases             types.Set    `tfsdk:"aliases"`
	Labels              types.Set    `tfsdk:"labels"`
	Picture             types.String `tfsdk:"picture"`
	TemperatureEntityID types.String `tfsdk:"temperature_entity_id"`
	HumidityEntityID    types.String `tfsdk:"humidity_entity_id"`
}

func (r *areaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_area"
}

// emptyStringSet is the default of set attributes for which HA reports none as `[]`.
func emptyStringSet() defaults.Set {
	return setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{}))
}

func (r *areaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An area: a room or zone that groups devices and entities, optionally on a floor.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Area ID, a slug such as `living_room`. Home Assistant derives it from " +
					"`name` when the area is created, and keeps it when the area is renamed. Set it to " +
					"choose the ID yourself. Changing it replaces the area. Creation fails if the ID is " +
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
				Description: "Display name. Changing it renames the area in place.",
				Required:    true,
			},
			"floor_id": schema.StringAttribute{
				Description: "ID of the floor the area is on, e.g. `homeassistant_floor.ground.id`. " +
					"Changing it moves the area in place.",
				Optional: true,
			},
			"icon": schema.StringAttribute{
				Description: "Icon, e.g. `mdi:sofa`.",
				Optional:    true,
			},
			"aliases": schema.SetAttribute{
				Description: "Other names for the area, for voice assistants. Defaults to none.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyStringSet(),
			},
			"labels": schema.SetAttribute{
				Description: "IDs of the labels on the area, e.g. `homeassistant_label.managed.id`. " +
					"Defaults to none.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyStringSet(),
			},
			"picture": schema.StringAttribute{
				Description: "URL of the area's picture, e.g. `/local/kitchen.jpg`.",
				Optional:    true,
			},
			"temperature_entity_id": schema.StringAttribute{
				Description: "Entity whose state is shown as the area's temperature. Home Assistant " +
					"requires an existing `sensor` with device class `temperature`.",
				Optional: true,
			},
			"humidity_entity_id": schema.StringAttribute{
				Description: "Entity whose state is shown as the area's humidity. Home Assistant " +
					"requires an existing `sensor` with device class `humidity`.",
				Optional: true,
			},
		},
	}
}

func (r *areaResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.HAClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.HAClient, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.areas = client.Areas(c.WS)
}

// fields are the area's settable fields other than name. Unset ones are sent as null or
// empty, which clears them.
func (m areaModel) fields(ctx context.Context) (map[string]any, diag.Diagnostics) {
	aliases, labels := []string{}, []string{}
	diags := m.Aliases.ElementsAs(ctx, &aliases, false)
	diags.Append(m.Labels.ElementsAs(ctx, &labels, false)...)
	return map[string]any{
		"floor_id":              m.FloorID.ValueStringPointer(),
		"icon":                  m.Icon.ValueStringPointer(),
		"aliases":               aliases,
		"labels":                labels,
		"picture":               m.Picture.ValueStringPointer(),
		"temperature_entity_id": m.TemperatureEntityID.ValueStringPointer(),
		"humidity_entity_id":    m.HumidityEntityID.ValueStringPointer(),
	}, diags
}

func areaState(ctx context.Context, a client.Area) (areaModel, diag.Diagnostics) {
	aliases, diags := types.SetValueFrom(ctx, types.StringType, append([]string{}, a.Aliases...))
	labels, d := types.SetValueFrom(ctx, types.StringType, append([]string{}, a.Labels...))
	diags.Append(d...)
	return areaModel{
		ID:                  types.StringValue(a.AreaID),
		Name:                types.StringValue(a.Name),
		FloorID:             types.StringPointerValue(a.FloorID),
		Icon:                types.StringPointerValue(a.Icon),
		Aliases:             aliases,
		Labels:              labels,
		Picture:             types.StringPointerValue(a.Picture),
		TemperatureEntityID: types.StringPointerValue(a.TemperatureEntityID),
		HumidityEntityID:    types.StringPointerValue(a.HumidityEntityID),
	}, diags
}

func (r *areaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan areaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fields, diags := plan.fields(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Unlike update, create rejects null, so unset fields are left out.
	for k, v := range fields {
		if p, ok := v.(*string); ok && p == nil {
			delete(fields, k)
		}
	}

	g := generatedID{
		kind:     "area",
		typeName: "homeassistant_area",
		create: func(ctx context.Context, name string) (string, error) {
			fields["name"] = name
			a, err := r.areas.Create(ctx, fields)
			return a.AreaID, err
		},
		rename: func(ctx context.Context, id, name string) error {
			_, err := r.areas.Update(ctx, id, map[string]any{"name": name})
			return err
		},
		delete: r.areas.Delete,
	}
	id, err := g.Create(ctx, plan.Name.ValueString(), plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Creating area", client.ErrorDetail(err))
		return
	}

	plan.ID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *areaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state areaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	a, ok, err := r.areas.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading area", client.ErrorDetail(err))
		return
	}
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	state, diags := areaState(ctx, a)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *areaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan areaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fields, diags := plan.fields(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	fields["name"] = plan.Name.ValueString()
	if _, err := r.areas.Update(ctx, plan.ID.ValueString(), fields); err != nil {
		resp.Diagnostics.AddError("Updating area", client.ErrorDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *areaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state areaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.areas.Delete(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Deleting area", client.ErrorDetail(err))
	}
}

func (r *areaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
