package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = (*floorResource)(nil)
	_ resource.ResourceWithImportState = (*floorResource)(nil)
)

// NewFloor returns the `homeassistant_floor` resource.
func NewFloor() resource.Resource {
	return &floorResource{}
}

type floorResource struct {
	floors client.Registry[client.Floor]
}

type floorModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Level   types.Int64  `tfsdk:"level"`
	Icon    types.String `tfsdk:"icon"`
	Aliases types.Set    `tfsdk:"aliases"`
}

func (r *floorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_floor"
}

func (r *floorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A floor: a level of a building that groups areas.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Floor ID, a slug such as `ground_floor`. Home Assistant derives it from " +
					"`name` when the floor is created, and keeps it when the floor is renamed. Set it to " +
					"choose the ID yourself. Changing it replaces the floor. Creation fails if the ID is " +
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
				Description: "Display name. Changing it renames the floor in place.",
				Required:    true,
			},
			"level": schema.Int64Attribute{
				Description: "Level, used to sort floors: `0` for the ground floor, negative for basements.",
				Optional:    true,
			},
			"icon": schema.StringAttribute{
				Description: "Icon, e.g. `mdi:home-floor-1`.",
				Optional:    true,
			},
			"aliases": schema.SetAttribute{
				Description: "Other names for the floor, for voice assistants. Defaults to none.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
		},
	}
}

func (r *floorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.HAClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.HAClient, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.floors = client.Floors(c.WS)
}

// fields are the floor's settable fields other than name. Unset ones are sent as null or
// empty, which clears them.
func (m floorModel) fields(ctx context.Context) (map[string]any, diag.Diagnostics) {
	aliases := []string{}
	diags := m.Aliases.ElementsAs(ctx, &aliases, false)
	return map[string]any{
		"level":   m.Level.ValueInt64Pointer(),
		"icon":    m.Icon.ValueStringPointer(),
		"aliases": aliases,
	}, diags
}

func floorState(ctx context.Context, f client.Floor) (floorModel, diag.Diagnostics) {
	aliases, diags := types.SetValueFrom(ctx, types.StringType, append([]string{}, f.Aliases...))
	return floorModel{
		ID:      types.StringValue(f.FloorID),
		Name:    types.StringValue(f.Name),
		Level:   types.Int64PointerValue(f.Level),
		Icon:    types.StringPointerValue(f.Icon),
		Aliases: aliases,
	}, diags
}

func (r *floorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan floorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fields, diags := plan.fields(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	g := generatedID{
		kind:     "floor",
		typeName: "homeassistant_floor",
		create: func(ctx context.Context, name string) (string, error) {
			fields["name"] = name
			f, err := r.floors.Create(ctx, fields)
			return f.FloorID, err
		},
		rename: func(ctx context.Context, id, name string) error {
			_, err := r.floors.Update(ctx, id, map[string]any{"name": name})
			return err
		},
		delete: r.floors.Delete,
	}
	id, err := g.Create(ctx, plan.Name.ValueString(), plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Creating floor", client.ErrorDetail(err))
		return
	}

	plan.ID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *floorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state floorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	f, ok, err := r.floors.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading floor", client.ErrorDetail(err))
		return
	}
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	state, diags := floorState(ctx, f)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *floorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan floorModel
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
	if _, err := r.floors.Update(ctx, plan.ID.ValueString(), fields); err != nil {
		resp.Diagnostics.AddError("Updating floor", client.ErrorDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *floorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state floorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.floors.Delete(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Deleting floor", client.ErrorDetail(err))
	}
}

func (r *floorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
