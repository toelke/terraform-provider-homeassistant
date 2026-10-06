package datasources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

var _ datasource.DataSourceWithConfigure = (*entityDataSource)(nil)

// stateReader is the part of the client the entity data source uses.
type stateReader interface {
	State(ctx context.Context, entityID string) (*client.EntityState, error)
}

// NewEntity returns the `homeassistant_entity` data source.
func NewEntity() datasource.DataSource {
	return &entityDataSource{}
}

type entityDataSource struct {
	client stateReader
}

type entityModel struct {
	EntityID          types.String  `tfsdk:"entity_id"`
	State             types.String  `tfsdk:"state"`
	FriendlyName      types.String  `tfsdk:"friendly_name"`
	DeviceClass       types.String  `tfsdk:"device_class"`
	UnitOfMeasurement types.String  `tfsdk:"unit_of_measurement"`
	Attributes        types.Dynamic `tfsdk:"attributes"`
	LastChanged       types.String  `tfsdk:"last_changed"`
	LastUpdated       types.String  `tfsdk:"last_updated"`
}

func (d *entityDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entity"
}

func (d *entityDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the current state and attributes of one entity (`GET /api/states/<entity_id>`). " +
			"The entity must exist.",
		Attributes: map[string]schema.Attribute{
			"entity_id": schema.StringAttribute{
				Description: "Entity ID, e.g. `sun.sun`.",
				Required:    true,
				Validators:  []validator.String{entityIDValidator},
			},
			"state": schema.StringAttribute{
				Description: "Current state, e.g. `on` or `21.5`.",
				Computed:    true,
			},
			"friendly_name": schema.StringAttribute{
				Description: "The `friendly_name` attribute, or null if the entity has none.",
				Computed:    true,
			},
			"device_class": schema.StringAttribute{
				Description: "The `device_class` attribute, or null if the entity has none.",
				Computed:    true,
			},
			"unit_of_measurement": schema.StringAttribute{
				Description: "The `unit_of_measurement` attribute, or null if the entity has none.",
				Computed:    true,
			},
			"attributes": schema.DynamicAttribute{
				Description: "All attributes, as an object that keeps their JSON types: numbers stay " +
					"numbers and booleans stay booleans.",
				Computed: true,
			},
			"last_changed": schema.StringAttribute{
				Description: "When the state last changed, in ISO 8601.",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "When the state or attributes last changed, in ISO 8601.",
				Computed:    true,
			},
		},
	}
}

func (d *entityDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := restClient(req, resp); c != nil {
		d.client = c
	}
}

func (d *entityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg entityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	entityID := cfg.EntityID.ValueString()

	s, err := d.client.State(ctx, entityID)
	if errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddAttributeError(entityIDPath, "Entity not found",
			fmt.Sprintf("Home Assistant has no entity %q. Check the entity ID: it is case-sensitive, "+
				"and Settings → Devices & services → Entities lists the existing ones.", entityID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Reading entity %s", entityID), client.ErrorDetail(err))
		return
	}

	model, err := entityModelFrom(entityID, s)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Decoding attributes of entity %s", entityID), err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// decodeAttributes decodes the attributes of s twice: as a dynamic value, and flat, for reading
// single attributes. Missing attributes decode as an empty object.
func decodeAttributes(s *client.EntityState) (attrs dyntype.Value, flat map[string]any, err error) {
	raw := s.Attributes
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	if attrs, err = dyntype.FromJSON(raw); err != nil {
		return attrs, nil, err
	}
	err = json.Unmarshal(raw, &flat)
	return attrs, flat, err
}

func entityModelFrom(entityID string, s *client.EntityState) (entityModel, error) {
	attrs, flat, err := decodeAttributes(s)
	if err != nil {
		return entityModel{}, err
	}
	return entityModel{
		EntityID:          types.StringValue(entityID),
		State:             types.StringValue(s.State),
		FriendlyName:      stringAttribute(flat, "friendly_name"),
		DeviceClass:       stringAttribute(flat, "device_class"),
		UnitOfMeasurement: stringAttribute(flat, "unit_of_measurement"),
		Attributes:        attrs.DynamicValue,
		LastChanged:       types.StringValue(s.LastChanged),
		LastUpdated:       types.StringValue(s.LastUpdated),
	}, nil
}

// stringAttribute returns the attribute key if it is a string, and null otherwise.
func stringAttribute(attrs map[string]any, key string) types.String {
	if s, ok := attrs[key].(string); ok {
		return types.StringValue(s)
	}
	return types.StringNull()
}

// entityIDValidator checks the shape of an entity ID, `<domain>.<object_id>`. It also keeps the
// ID from reaching other paths of the REST API.
var entityIDValidator = stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9_]+\.[a-z0-9_]+$`),
	`must be an entity ID like "light.bedroom_ceiling": a domain, a dot, and an object ID, both `+
		`made of lowercase letters, digits, and underscores`)
