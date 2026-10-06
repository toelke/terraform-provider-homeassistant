package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewESPHome returns the `homeassistant_esphome` resource (spec/integrations.md).
func NewESPHome() resource.Resource {
	return &typedIntegration[esphomeModel, *esphomeModel]{spec: typedIntegrationSpec{
		domain: "esphome",
		description: "A config entry of the ESPHome integration: one ESPHome device, added by its address. " +
			"Changing `host`, `port` or `password` reconfigures the entry in place, and the options " +
			"are set through its options flow. Home Assistant cannot return the data or options an " +
			"entry has, so they are never compared with Home Assistant; only an entry that is gone " +
			"is detected as drift. After an import, the first apply only records the attributes.",
		attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "Host name or IP address of the device, e.g. `kitchen-sensor.local`.",
				Required:    true,
			},
			"port": schema.Int64Attribute{
				Description: "Port of the device's native API. Defaults to `6053`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(6053),
				Validators:  []validator.Int64{int64validator.Between(1, 65535)},
			},
			"noise_psk": schema.StringAttribute{
				Description: "The device's API encryption key (`api: encryption: key:` in its YAML). " +
					"Home Assistant asks for it only if the device uses encryption and it has no key " +
					"from the ESPHome dashboard. Changing it replaces the entry.",
				Optional:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{replaceUnlessImported("host")},
			},
			"password": schema.StringAttribute{
				Description: "The device's API password, which ESPHome deprecates in favour of `noise_psk`. " +
					"Home Assistant asks for it only if the device uses one.",
				Optional:  true,
				Sensitive: true,
			},
			"allow_service_calls": schema.BoolAttribute{
				Description: "Whether the device may perform Home Assistant actions. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"subscribe_logs": schema.BoolAttribute{
				Description: "Whether Home Assistant subscribes to the device's logs. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}}
}

type esphomeModel struct {
	entryModel
	Host              types.String `tfsdk:"host"`
	Port              types.Int64  `tfsdk:"port"`
	NoisePSK          types.String `tfsdk:"noise_psk"`
	Password          types.String `tfsdk:"password"`
	AllowServiceCalls types.Bool   `tfsdk:"allow_service_calls"`
	SubscribeLogs     types.Bool   `tfsdk:"subscribe_logs"`
}

func (m *esphomeModel) entry() *entryModel { return &m.entryModel }

func (m *esphomeModel) imported() bool { return m.Host.IsNull() }

// flowInput answers the forms `user` (host, port), `encryption_key` (noise_psk) and
// `authenticate` (password). ESPHome takes no files.
//
//nolint:unparam // implements typedModel
func (m *esphomeModel) flowInput(context.Context) (map[string]any, map[string]string, diag.Diagnostics) {
	values := map[string]any{
		"host": m.Host.ValueString(),
		"port": m.Port.ValueInt64(),
	}
	setString(values, "noise_psk", m.NoisePSK)
	setString(values, "password", m.Password)
	return values, nil, nil
}

func (m *esphomeModel) optionInput() map[string]any {
	return map[string]any{
		"allow_service_calls": m.AllowServiceCalls.ValueBool(),
		"subscribe_logs":      m.SubscribeLogs.ValueBool(),
	}
}

// setString sets values[key] to v unless v is null.
func setString(values map[string]any, key string, v types.String) {
	if !v.IsNull() {
		values[key] = v.ValueString()
	}
}
