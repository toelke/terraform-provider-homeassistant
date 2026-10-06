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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// helperSpec describes one input helper domain (ADR-0013). Every helper shares the generic
// implementation in helperResource.
type helperSpec struct {
	// domain is the helper's domain, e.g. "input_boolean".
	domain string
	// description is the resource description.
	description string
	// fields are the helper's attributes besides `id`, `name`, `icon`, and `entity_id`. Each is
	// stored in HA under the attribute's name.
	fields map[string]schema.Attribute
	// validate, if set, checks the config at plan time. It gets the config's attributes; values
	// that are not known yet are unknown, and it skips them.
	validate func(attrs map[string]attr.Value) diag.Diagnostics
}

var (
	_ resource.ResourceWithConfigure      = (*helperResource)(nil)
	_ resource.ResourceWithImportState    = (*helperResource)(nil)
	_ resource.ResourceWithValidateConfig = (*helperResource)(nil)
)

// newHelper returns the resource for a helper domain.
func newHelper(spec helperSpec) resource.Resource {
	return &helperResource{spec: spec}
}

// helperResource manages helpers of one domain. Its model is the whole resource object, so the
// same code serves every helper's schema: attributes other than `id` and `entity_id` map 1:1 to
// the helper's fields in HA.
type helperResource struct {
	spec    helperSpec
	helpers client.Registry[client.Helper]
}

func (r *helperResource) typeName() string {
	return "homeassistant_" + r.spec.domain
}

func (r *helperResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.spec.domain
}

func (r *helperResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "Helper ID, a slug such as `guest_mode`. Home Assistant derives it from `name` " +
				"when the helper is created, and keeps it when the helper is renamed. Set it to choose the ID " +
				"yourself. Changing it replaces the helper. Creation fails if the ID is taken. Import with " +
				"this ID.",
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
			Validators: []validator.String{slugValidator{}},
		},
		"name": schema.StringAttribute{
			Description: "Display name. Changing it renames the helper in place.",
			Required:    true,
		},
		"icon": schema.StringAttribute{
			Description: "Icon, e.g. `mdi:toggle-switch`.",
			Optional:    true,
		},
		"entity_id": schema.StringAttribute{
			Description: fmt.Sprintf("Entity ID of the helper, `%s.<id>`.", r.spec.domain),
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
	for name, a := range r.spec.fields {
		attrs[name] = a
	}
	resp.Schema = schema.Schema{Description: r.spec.description, Attributes: attrs}
}

func (r *helperResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if r.spec.validate == nil {
		return
	}
	var cfg types.Object
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.spec.validate(cfg.Attributes())...)
}

func (r *helperResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c := haClient(req, resp)
	if c == nil {
		return
	}
	r.helpers = client.Helpers(c.WS, r.spec.domain)
}

// helperFields returns the helper's fields from the resource object m, including `name`. HA
// replaces the whole helper on update, so unset attributes are left out, which clears them.
func helperFields(m types.Object) (map[string]any, error) {
	fields := map[string]any{}
	for name, v := range m.Attributes() {
		if name == "id" || name == "entity_id" || v.IsNull() || v.IsUnknown() {
			continue
		}
		switch v := v.(type) {
		case types.String:
			fields[name] = v.ValueString()
		case types.Bool:
			fields[name] = v.ValueBool()
		case types.Float64:
			fields[name] = v.ValueFloat64()
		case types.Int64:
			fields[name] = v.ValueInt64()
		case types.List:
			items := make([]any, 0, len(v.Elements()))
			for _, e := range v.Elements() {
				s, ok := e.(types.String)
				if !ok || s.IsNull() || s.IsUnknown() {
					return nil, fmt.Errorf("attribute %q has an element %v that is not a known string. This is a bug in the provider", name, e)
				}
				items = append(items, s.ValueString())
			}
			fields[name] = items
		default:
			return nil, fmt.Errorf("attribute %q has the unsupported type %T. This is a bug in the provider", name, v)
		}
	}
	return fields, nil
}

// helperObject converts a helper as HA stores it into the resource object of type t. Fields HA
// does not report are null.
func (r *helperResource) helperObject(t types.ObjectType, h client.Helper) (types.Object, diag.Diagnostics) {
	vals := map[string]attr.Value{}
	for name, typ := range t.AttrTypes {
		raw, set := h[name]
		switch name {
		case "id":
			raw, set = h.HelperID(), true
		case "entity_id":
			raw, set = r.spec.domain+"."+h.HelperID(), true
		}
		switch {
		case typ.Equal(types.StringType):
			s, ok := raw.(string)
			if !set || !ok {
				vals[name] = types.StringNull()
				continue
			}
			vals[name] = types.StringValue(s)
		case typ.Equal(types.BoolType):
			b, ok := raw.(bool)
			if !set || !ok {
				vals[name] = types.BoolNull()
				continue
			}
			vals[name] = types.BoolValue(b)
		case typ.Equal(types.Float64Type):
			f, ok := raw.(float64)
			if !set || !ok {
				vals[name] = types.Float64Null()
				continue
			}
			vals[name] = types.Float64Value(f)
		case typ.Equal(types.Int64Type):
			// JSON numbers decode to float64.
			f, ok := raw.(float64)
			if !set || !ok {
				vals[name] = types.Int64Null()
				continue
			}
			vals[name] = types.Int64Value(int64(f))
		case typ.Equal(types.ListType{ElemType: types.StringType}):
			items, ok := raw.([]any)
			if !set || !ok {
				vals[name] = types.ListNull(types.StringType)
				continue
			}
			elems := make([]attr.Value, 0, len(items))
			for _, item := range items {
				s, _ := item.(string)
				elems = append(elems, types.StringValue(s))
			}
			vals[name] = types.ListValueMust(types.StringType, elems)
		default:
			var diags diag.Diagnostics
			diags.AddError("Reading helper", fmt.Sprintf(
				"Attribute %q has the unsupported type %s. This is a bug in the provider.", name, typ))
			return types.ObjectNull(t.AttrTypes), diags
		}
	}
	return types.ObjectValue(t.AttrTypes, vals)
}

func (r *helperResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan types.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fields, err := helperFields(plan)
	if err != nil {
		resp.Diagnostics.AddError("Creating helper", err.Error())
		return
	}
	var planID types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("id"), &planID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// withName returns the fields with name replaced.
	withName := func(name string) map[string]any {
		f := make(map[string]any, len(fields))
		for k, v := range fields {
			f[k] = v
		}
		f["name"] = name
		return f
	}
	g := generatedID{
		kind:     "helper",
		typeName: r.typeName(),
		create: func(ctx context.Context, name string) (string, error) {
			h, err := r.helpers.Create(ctx, withName(name))
			return h.HelperID(), err
		},
		rename: func(ctx context.Context, id, name string) error {
			_, err := r.helpers.Update(ctx, id, withName(name))
			return err
		},
		delete: r.helpers.Delete,
	}
	id, err := g.Create(ctx, fields["name"].(string), planID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Creating helper", client.ErrorDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entity_id"), r.spec.domain+"."+id)...)
}

func (r *helperResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	h, ok, err := r.helpers.Get(ctx, id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading helper", client.ErrorDetail(err))
		return
	}
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	obj, diags := r.helperObject(state.Type(ctx).(types.ObjectType), h)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, obj)...)
}

func (r *helperResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan types.Object
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var id types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fields, err := helperFields(plan)
	if err != nil {
		resp.Diagnostics.AddError("Updating helper", err.Error())
		return
	}

	if _, err := r.helpers.Update(ctx, id.ValueString(), fields); err != nil {
		resp.Diagnostics.AddError("Updating helper", client.ErrorDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *helperResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.helpers.Delete(ctx, id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Deleting helper", client.ErrorDetail(err))
	}
}

func (r *helperResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// configValue returns the config attribute `name` as T. set is false if it is null; known is
// false if it is unknown. Plan-time checks skip unknown values.
func configValue[T attr.Value](attrs map[string]attr.Value, name string) (v T, set, known bool) {
	a, ok := attrs[name]
	if !ok {
		return v, false, true
	}
	v, _ = a.(T)
	return v, !a.IsNull(), !a.IsUnknown()
}
