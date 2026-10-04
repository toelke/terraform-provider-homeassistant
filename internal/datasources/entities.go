package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

var _ datasource.DataSourceWithConfigure = (*entitiesDataSource)(nil)

// entitiesReader is the part of the client the entities data source uses.
type entitiesReader interface {
	States(ctx context.Context) ([]client.EntityState, error)
	RenderTemplate(ctx context.Context, template string) (string, error)
}

// NewEntities returns the `homeassistant_entities` data source.
func NewEntities() datasource.DataSource {
	return &entitiesDataSource{}
}

type entitiesDataSource struct {
	client entitiesReader
}

type entitiesModel struct {
	Domain      types.String  `tfsdk:"domain"`
	Area        types.String  `tfsdk:"area"`
	Label       types.String  `tfsdk:"label"`
	DeviceClass types.String  `tfsdk:"device_class"`
	NamePattern types.String  `tfsdk:"name_pattern"`
	EntityIDs   types.List    `tfsdk:"entity_ids"`
	Entities    types.Dynamic `tfsdk:"entities"`
}

func (d *entitiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entities"
}

func (d *entitiesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the entities that match every filter that is set (`GET /api/states`). " +
			"Without filters, it lists every entity. There is no filter on the current state, so " +
			"the result only changes when entities or their registry entries change.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Description: "Only entities of this domain, e.g. `light`.",
				Optional:    true,
			},
			"area": schema.StringAttribute{
				Description: "Only entities in this area, given by area ID or name. Entities of a device " +
					"in the area count too. An unknown area matches no entities.",
				Optional: true,
			},
			"label": schema.StringAttribute{
				Description: "Only entities with this label, given by label ID or name. An unknown label " +
					"matches no entities.",
				Optional: true,
			},
			"device_class": schema.StringAttribute{
				Description: "Only entities whose `device_class` attribute is this, e.g. `temperature`.",
				Optional:    true,
			},
			"name_pattern": schema.StringAttribute{
				Description: "Only entities whose `friendly_name` matches this glob pattern: `*` matches " +
					"any characters, `?` matches one character, and everything else matches itself, " +
					"case-sensitively. Entities without a `friendly_name` never match.",
				Optional: true,
			},
			"entity_ids": schema.ListAttribute{
				Description: "IDs of the matching entities, sorted.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"entities": schema.DynamicAttribute{
				Description: "The matching entities, as an object keyed by entity ID. Each value has " +
					"`state`, `friendly_name` (null if the entity has none), `attributes` (keeping " +
					"their JSON types), and `last_changed`.",
				Computed: true,
			},
		},
	}
}

func (d *entitiesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := restClient(req, resp); c != nil {
		d.client = c
	}
}

func (d *entitiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg entitiesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	states, err := d.client.States(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Reading entity states", client.ErrorDetail(err))
		return
	}

	f := entityFilter{
		domain:      cfg.Domain.ValueString(),
		deviceClass: cfg.DeviceClass.ValueString(),
	}
	if !cfg.NamePattern.IsNull() {
		f.name = globRegexp(cfg.NamePattern.ValueString())
	}
	for _, m := range []struct {
		attribute string
		value     types.String
		function  string
		target    *map[string]bool
	}{
		{"area", cfg.Area, "area_entities", &f.area},
		{"label", cfg.Label, "label_entities", &f.label},
	} {
		if m.value.IsNull() {
			continue
		}
		ids, err := d.entityIDsOf(ctx, m.function, m.value.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root(m.attribute),
				fmt.Sprintf("Resolving the entities of %s %q", m.attribute, m.value.ValueString()),
				client.ErrorDetail(err))
			return
		}
		*m.target = ids
	}

	matched := f.apply(states)
	ids := make([]attr.Value, len(matched))
	for i, s := range matched {
		ids[i] = types.StringValue(s.EntityID)
	}
	cfg.EntityIDs = types.ListValueMust(types.StringType, ids)
	cfg.Entities, err = entitiesValue(matched)
	if err != nil {
		resp.Diagnostics.AddError("Decoding entity attributes", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}

// entityIDsOf renders `<function>(<value>)`, e.g. area_entities, through the template API and
// returns the entity IDs as a set.
func (d *entitiesDataSource) entityIDsOf(ctx context.Context, function, value string) (map[string]bool, error) {
	result, err := d.client.RenderTemplate(ctx, fmt.Sprintf("{{ %s(%s) | tojson }}", function, jinjaString(value)))
	if err != nil {
		return nil, err
	}
	var ids []string
	if err := json.Unmarshal([]byte(result), &ids); err != nil {
		return nil, fmt.Errorf("decoding the result of %s: %w", function, err)
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// entityFilter holds the filters of the entities data source. Zero values don't filter.
type entityFilter struct {
	domain      string
	deviceClass string
	name        *regexp.Regexp
	// area and label are the entity IDs in the area and with the label.
	area, label map[string]bool
}

// apply returns the states that pass every filter, sorted by entity ID.
func (f entityFilter) apply(states []client.EntityState) []client.EntityState {
	var out []client.EntityState
	for _, s := range states {
		if f.matches(s) {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EntityID < out[j].EntityID })
	return out
}

func (f entityFilter) matches(s client.EntityState) bool {
	if f.domain != "" && !strings.HasPrefix(s.EntityID, f.domain+".") {
		return false
	}
	if f.area != nil && !f.area[s.EntityID] {
		return false
	}
	if f.label != nil && !f.label[s.EntityID] {
		return false
	}
	if f.deviceClass == "" && f.name == nil {
		return true
	}
	var attrs struct {
		FriendlyName any `json:"friendly_name"`
		DeviceClass  any `json:"device_class"`
	}
	_ = json.Unmarshal(s.Attributes, &attrs) // missing or malformed attributes match nothing
	if f.deviceClass != "" && attrs.DeviceClass != f.deviceClass {
		return false
	}
	if f.name != nil {
		name, ok := attrs.FriendlyName.(string)
		if !ok || !f.name.MatchString(name) {
			return false
		}
	}
	return true
}

// globRegexp compiles a glob pattern: `*` matches any characters, `?` one character, and
// everything else itself.
func globRegexp(pattern string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString(`(?s)^`)
	for _, r := range pattern {
		switch r {
		case '*':
			b.WriteString(`.*`)
		case '?':
			b.WriteString(`.`)
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString(`$`)
	return regexp.MustCompile(b.String())
}

// jinjaString quotes s as a Jinja string literal. Jinja decodes the backslash escapes of a
// literal like Python's unicode-escape codec, so every character other than printable ASCII, and
// the quote and backslash, is written as `\uXXXX` or `\UXXXXXXXX`. Nothing in the result can
// end the literal or the surrounding `{{ }}`.
func jinjaString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\' || r < 0x20 || r > 0x7e:
			if r > 0xffff {
				fmt.Fprintf(&b, `\U%08x`, r)
			} else {
				fmt.Fprintf(&b, `\u%04x`, r)
			}
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// entitiesValue returns the `entities` attribute: an object keyed by entity ID.
func entitiesValue(states []client.EntityState) (types.Dynamic, error) {
	typs := make(map[string]attr.Type, len(states))
	values := make(map[string]attr.Value, len(states))
	for _, s := range states {
		raw := s.Attributes
		if len(raw) == 0 {
			raw = json.RawMessage("{}")
		}
		attrs, err := dyntype.FromJSON(raw)
		if err != nil {
			return types.Dynamic{}, fmt.Errorf("entity %s: %w", s.EntityID, err)
		}
		var flat map[string]any
		if err := json.Unmarshal(raw, &flat); err != nil {
			return types.Dynamic{}, fmt.Errorf("entity %s: %w", s.EntityID, err)
		}
		a := attrs.UnderlyingValue()
		attrTypes := map[string]attr.Type{
			"state":         types.StringType,
			"friendly_name": types.StringType,
			"attributes":    a.Type(context.Background()),
			"last_changed":  types.StringType,
		}
		v, diags := types.ObjectValue(attrTypes, map[string]attr.Value{
			"state":         types.StringValue(s.State),
			"friendly_name": stringAttribute(flat, "friendly_name"),
			"attributes":    a,
			"last_changed":  types.StringValue(s.LastChanged),
		})
		if diags.HasError() {
			return types.Dynamic{}, fmt.Errorf("entity %s: %v", s.EntityID, diags)
		}
		typs[s.EntityID] = types.ObjectType{AttrTypes: attrTypes}
		values[s.EntityID] = v
	}
	obj, diags := types.ObjectValue(typs, values)
	if diags.HasError() {
		return types.Dynamic{}, fmt.Errorf("building entities: %v", diags)
	}
	return types.DynamicValue(obj), nil
}
