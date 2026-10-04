package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// HA's defaults for the MQTT broker settings and options (spec/integrations.md).
const (
	mqttDefaultPort       = 1883
	mqttDefaultKeepalive  = 60
	mqttDefaultProtocol   = "3.1.1"
	mqttDefaultTransport  = "tcp"
	mqttDefaultPrefix     = "homeassistant"
	mqttStatusTopic       = mqttDefaultPrefix + "/status"
	mqttTransportWS       = "websockets"
	mqttAdvancedOptions   = "advanced_options"
	mqttBirthPayload      = "online"
	mqttWillPayload       = "offline"
	mqttDefaultMessageQoS = 0
)

// mqttAdvancedDefaults are HA's defaults for the broker fields that HA before 2026.9 shows only
// after `advanced_options`. A value equal to its default needs no advanced form.
var mqttAdvancedDefaults = map[string]any{
	"keepalive":       int64(mqttDefaultKeepalive),
	"protocol":        mqttDefaultProtocol,
	"transport":       mqttDefaultTransport,
	"set_ca_cert":     "off",
	"set_client_cert": false,
	"tls_insecure":    false,
}

// NewMQTT returns the `homeassistant_mqtt` resource (spec/integrations.md).
func NewMQTT() resource.Resource {
	return &typedIntegration[mqttModel, *mqttModel]{spec: typedIntegrationSpec{
		domain: "mqtt",
		description: "The config entry of the MQTT integration: the connection to an MQTT broker. Home " +
			"Assistant allows only one. Changing the broker settings reconfigures the entry in place, " +
			"and discovery and the birth and will messages are set through its options flow. Home " +
			"Assistant cannot return the data or options an entry has, so they are never compared " +
			"with Home Assistant; only an entry that is gone is detected as drift. After an import, " +
			"the first apply only records the attributes.",
		attributes: map[string]schema.Attribute{
			"broker": schema.StringAttribute{
				Description: "Host name or IP address of the broker.",
				Required:    true,
			},
			"port": schema.Int64Attribute{
				Description: "Port of the broker. Defaults to `1883`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(mqttDefaultPort),
				Validators:  []validator.Int64{int64Between{1, 65535}},
			},
			"username": schema.StringAttribute{
				Description: "User name to log in to the broker.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password to log in to the broker. Removing it replaces the entry, because " +
					"a reconfigure keeps the old password when none is given.",
				Optional:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIf(passwordRemoved, mqttPasswordDesc, mqttPasswordDesc)},
			},
			"client_id": schema.StringAttribute{
				Description: "Client ID to connect with. Unset, the MQTT client picks a random one.",
				Optional:    true,
			},
			"keepalive": schema.Int64Attribute{
				Description: "Seconds between keepalive messages, at least 15. Defaults to `60`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(mqttDefaultKeepalive),
				Validators:  []validator.Int64{int64Between{min: 15}},
			},
			"protocol": schema.StringAttribute{
				Description: "MQTT protocol version: `3.1`, `3.1.1` or `5`. Defaults to `3.1.1`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(mqttDefaultProtocol),
				Validators:  []validator.String{oneOfValidator{"3.1", "3.1.1", "5"}},
			},
			"transport": schema.StringAttribute{
				Description: "Transport: `tcp` or `websockets`. Defaults to `tcp`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(mqttDefaultTransport),
				Validators:  []validator.String{oneOfValidator{mqttDefaultTransport, mqttTransportWS}},
			},
			"ws_path": schema.StringAttribute{
				Description: "WebSocket path, only with `transport = \"websockets\"`. Unset, Home Assistant uses `/`.",
				Optional:    true,
			},
			"ws_headers": schema.MapAttribute{
				Description: "Extra HTTP headers of the WebSocket connection, only with `transport = \"websockets\"`.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"tls": schema.SingleNestedAttribute{
				Description: "Connect with TLS. Unset, the connection is not encrypted.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"ca_certificate": schema.StringAttribute{
						Description: "PEM certificate of the CA that signed the broker's certificate, e.g. " +
							"`file(\"ca.pem\")`. Unset, the system's CAs are trusted.",
						Optional: true,
					},
					"client_certificate": schema.StringAttribute{
						Description: "PEM client certificate to log in with. Requires `client_key`.",
						Optional:    true,
					},
					"client_key": schema.StringAttribute{
						Description: "PEM private key of `client_certificate`, not encrypted.",
						Optional:    true,
						Sensitive:   true,
					},
					"insecure": schema.BoolAttribute{
						Description: "Do not check that the broker's certificate matches `broker`. Defaults to `false`.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
				},
			},
			"discovery": schema.BoolAttribute{
				Description: "Whether MQTT discovery is on. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"discovery_prefix": schema.StringAttribute{
				Description: "Topic prefix of MQTT discovery. Defaults to `homeassistant`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(mqttDefaultPrefix),
			},
			"birth_message": mqttMessageAttribute("Message Home Assistant publishes when it connects.", mqttBirthPayload),
			"will_message":  mqttMessageAttribute("Message the broker publishes when Home Assistant disconnects.", mqttWillPayload),
		},
		validate: validateMQTT,
		answer:   mqttAnswer,
	}}
}

const mqttPasswordDesc = "Removing the password replaces the config entry."

// passwordRemoved requires replacement when a password is removed: a reconfigure flow without a
// password keeps the old one.
func passwordRemoved(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = req.PlanValue.IsNull() && !req.StateValue.IsNull()
}

var mqttMessageTypes = map[string]attr.Type{
	"enabled": types.BoolType,
	"topic":   types.StringType,
	"payload": types.StringType,
	"qos":     types.Int64Type,
	"retain":  types.BoolType,
}

// mqttMessageAttribute is a birth or will message, defaulting to HA's.
func mqttMessageAttribute(desc, payload string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: desc + " Unset, Home Assistant's default is used: topic `" + mqttStatusTopic +
			"`, payload `" + payload + "`.",
		Optional: true,
		Computed: true,
		Default: objectdefault.StaticValue(types.ObjectValueMust(mqttMessageTypes, map[string]attr.Value{
			"enabled": types.BoolValue(true),
			"topic":   types.StringValue(mqttStatusTopic),
			"payload": types.StringValue(payload),
			"qos":     types.Int64Value(mqttDefaultMessageQoS),
			"retain":  types.BoolValue(false),
		})),
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether the message is sent. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"topic": schema.StringAttribute{
				Description: "Topic. Defaults to `" + mqttStatusTopic + "`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(mqttStatusTopic),
			},
			"payload": schema.StringAttribute{
				Description: "Payload. Defaults to `" + payload + "`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(payload),
			},
			"qos": schema.Int64Attribute{
				Description: "Quality of service: 0, 1 or 2. Defaults to `0`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(mqttDefaultMessageQoS),
				Validators:  []validator.Int64{int64Between{0, 2}},
			},
			"retain": schema.BoolAttribute{
				Description: "Whether the broker retains the message. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

type mqttModel struct {
	entryModel
	Broker          types.String      `tfsdk:"broker"`
	Port            types.Int64       `tfsdk:"port"`
	Username        types.String      `tfsdk:"username"`
	Password        types.String      `tfsdk:"password"`
	ClientID        types.String      `tfsdk:"client_id"`
	Keepalive       types.Int64       `tfsdk:"keepalive"`
	Protocol        types.String      `tfsdk:"protocol"`
	Transport       types.String      `tfsdk:"transport"`
	WSPath          types.String      `tfsdk:"ws_path"`
	WSHeaders       types.Map         `tfsdk:"ws_headers"`
	TLS             *mqttTLSModel     `tfsdk:"tls"`
	Discovery       types.Bool        `tfsdk:"discovery"`
	DiscoveryPrefix types.String      `tfsdk:"discovery_prefix"`
	BirthMessage    *mqttMessageModel `tfsdk:"birth_message"`
	WillMessage     *mqttMessageModel `tfsdk:"will_message"`
}

type mqttTLSModel struct {
	CACertificate     types.String `tfsdk:"ca_certificate"`
	ClientCertificate types.String `tfsdk:"client_certificate"`
	ClientKey         types.String `tfsdk:"client_key"`
	Insecure          types.Bool   `tfsdk:"insecure"`
}

type mqttMessageModel struct {
	Enabled types.Bool   `tfsdk:"enabled"`
	Topic   types.String `tfsdk:"topic"`
	Payload types.String `tfsdk:"payload"`
	QoS     types.Int64  `tfsdk:"qos"`
	Retain  types.Bool   `tfsdk:"retain"`
}

func (m *mqttModel) entry() *entryModel { return &m.entryModel }

func (m *mqttModel) imported() bool { return m.Broker.IsNull() }

// flowInput answers the `broker` form of every release in the support window. Certificates are
// files, which the form takes as uploads.
func (m *mqttModel) flowInput(ctx context.Context) (map[string]any, map[string]string, diag.Diagnostics) {
	values := map[string]any{
		"broker":          m.Broker.ValueString(),
		"port":            m.Port.ValueInt64(),
		"keepalive":       m.Keepalive.ValueInt64(),
		"protocol":        m.Protocol.ValueString(),
		"transport":       m.Transport.ValueString(),
		"set_ca_cert":     "off",
		"set_client_cert": false,
		"tls_insecure":    false,
	}
	setString(values, "username", m.Username)
	setString(values, "password", m.Password)
	setString(values, "client_id", m.ClientID)
	setString(values, "ws_path", m.WSPath)
	var diags diag.Diagnostics
	if !m.WSHeaders.IsNull() {
		headers := map[string]string{}
		diags.Append(m.WSHeaders.ElementsAs(ctx, &headers, false)...)
		b, err := json.Marshal(headers)
		if err != nil {
			diags.AddAttributeError(path.Root("ws_headers"), "Invalid ws_headers", err.Error())
		}
		values["ws_headers"] = string(b)
	}

	files := map[string]string{}
	if t := m.TLS; t != nil {
		values["tls_insecure"] = t.Insecure.ValueBool()
		values["set_ca_cert"] = "auto"
		if !t.CACertificate.IsNull() {
			values["set_ca_cert"] = "custom"
			files["certificate"] = t.CACertificate.ValueString()
		}
		if !t.ClientCertificate.IsNull() {
			values["set_client_cert"] = true
			files["client_cert"] = t.ClientCertificate.ValueString()
			files["client_key"] = t.ClientKey.ValueString()
		}
	}
	return values, files, diags
}

func (m *mqttModel) optionInput() map[string]any {
	options := map[string]any{
		"discovery":        m.Discovery.ValueBool(),
		"discovery_prefix": m.DiscoveryPrefix.ValueString(),
	}
	for prefix, msg := range map[string]*mqttMessageModel{"birth": m.BirthMessage, "will": m.WillMessage} {
		if msg == nil {
			continue
		}
		options[prefix+"_enable"] = msg.Enabled.ValueBool()
		options[prefix+"_topic"] = msg.Topic.ValueString()
		options[prefix+"_payload"] = msg.Payload.ValueString()
		options[prefix+"_qos"] = msg.QoS.ValueInt64()
		options[prefix+"_retain"] = msg.Retain.ValueBool()
	}
	return options
}

// mqttAnswer adds `advanced_options = true` to a form of HA before 2026.9 that hides fields the
// resource sets to other than HA's defaults, so HA shows them next.
func mqttAnswer(base client.FlowAnswer, values map[string]any, files map[string]string) client.FlowAnswer {
	return func(ctx context.Context, form client.FlowForm) (map[string]any, error) {
		data, err := base(ctx, form)
		if err != nil || !form.Has(mqttAdvancedOptions) {
			return data, err
		}
		for k, v := range values {
			if def, ok := mqttAdvancedDefaults[k]; !form.Has(k) && (!ok || v != def) {
				data[mqttAdvancedOptions] = true
			}
		}
		for k := range files {
			if !form.Has(k) {
				data[mqttAdvancedOptions] = true
			}
		}
		return data, nil
	}
}

// validateMQTT checks that the WebSocket settings come with `transport = "websockets"`, and that
// the client certificate comes with its key.
func validateMQTT(ctx context.Context, req resource.ValidateConfigRequest) diag.Diagnostics {
	var diags diag.Diagnostics
	var transport, wsPath types.String
	var wsHeaders types.Map
	diags.Append(req.Config.GetAttribute(ctx, path.Root("transport"), &transport)...)
	diags.Append(req.Config.GetAttribute(ctx, path.Root("ws_path"), &wsPath)...)
	diags.Append(req.Config.GetAttribute(ctx, path.Root("ws_headers"), &wsHeaders)...)
	if diags.HasError() || transport.IsUnknown() {
		return diags
	}
	if transport.ValueString() != mqttTransportWS {
		for name, v := range map[string]attr.Value{"ws_path": wsPath, "ws_headers": wsHeaders} {
			if !v.IsNull() {
				diags.AddAttributeError(path.Root(name), "Invalid Attribute Combination",
					fmt.Sprintf("%s needs transport = %q.", name, mqttTransportWS))
			}
		}
	}

	var cert, key types.String
	diags.Append(req.Config.GetAttribute(ctx, path.Root("tls").AtName("client_certificate"), &cert)...)
	diags.Append(req.Config.GetAttribute(ctx, path.Root("tls").AtName("client_key"), &key)...)
	if diags.HasError() || cert.IsUnknown() || key.IsUnknown() {
		return diags
	}
	if cert.IsNull() != key.IsNull() {
		diags.AddAttributeError(path.Root("tls"), "Invalid Attribute Combination",
			"client_certificate and client_key must be set together.")
	}
	return diags
}
