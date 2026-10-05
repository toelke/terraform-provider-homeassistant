package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// fakeConfigFlow serves a config flow that starts with the first reply and answers each submit
// with the next one. It records the submitted data and the uploaded files.
type fakeConfigFlow struct {
	t         *testing.T
	replies   []string
	submitted []map[string]any
	uploads   []string
}

func (f *fakeConfigFlow) run(answer func(*client.RESTClient) client.FlowAnswer) (string, error) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/file_upload":
			file, _, err := r.FormFile("file")
			if err != nil {
				f.t.Fatalf("upload: %v", err)
			}
			b, _ := io.ReadAll(file)
			f.uploads = append(f.uploads, string(b))
			_, _ = fmt.Fprintf(w, `{"file_id":"file%d"}`, len(f.uploads))
			return
		case r.Method == http.MethodDelete:
			f.t.Errorf("flow aborted")
			return
		case r.URL.Path != "/api/config/config_entries/flow":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				f.t.Fatal(err)
			}
			f.submitted = append(f.submitted, body)
		}
		if len(f.replies) == 0 {
			f.t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(f.replies[0]))
		f.replies = f.replies[1:]
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	rest := client.New(client.Config{URL: u, Token: "t", Timeout: 5 * time.Second}).REST
	return client.NewConfigFlows(rest).RunAnswer(context.Background(), "test", "", answer(rest))
}

func form(stepID, schema string) string {
	return `{"type":"form","flow_id":"f1","step_id":"` + stepID + `","errors":{},"data_schema":` + schema + `}`
}

const created = `{"type":"create_entry","flow_id":"f1","result":{"entry_id":"E1"}}`

func TestESPHomeFlowInput(t *testing.T) {
	m := esphomeModel{
		Host:              types.StringValue("kitchen.local"),
		Port:              types.Int64Value(6053),
		NoisePSK:          types.StringValue("key="),
		Password:          types.StringNull(),
		AllowServiceCalls: types.BoolValue(true),
		SubscribeLogs:     types.BoolValue(false),
	}
	values, files, diags := m.flowInput(context.Background())
	if diags.HasError() {
		t.Fatal(diags)
	}
	if want := map[string]any{"host": "kitchen.local", "port": int64(6053), "noise_psk": "key="}; !reflect.DeepEqual(values, want) {
		t.Errorf("values = %v", values)
	}
	if len(files) != 0 {
		t.Errorf("files = %v", files)
	}
	if want := map[string]any{"allow_service_calls": true, "subscribe_logs": false}; !reflect.DeepEqual(m.optionInput(), want) {
		t.Errorf("options = %v", m.optionInput())
	}
}

func TestESPHomeAnswersEveryForm(t *testing.T) {
	m := esphomeModel{
		Host:     types.StringValue("kitchen.local"),
		Port:     types.Int64Value(6053),
		NoisePSK: types.StringValue("key="),
		Password: types.StringValue("hunter2"),
	}
	values, files, _ := m.flowInput(context.Background())
	f := &fakeConfigFlow{t: t, replies: []string{
		form("user", `[{"name":"host","required":true},{"name":"port","optional":true,"default":6053}]`),
		form("encryption_key", `[{"name":"noise_psk","required":true}]`),
		form("authenticate", `[{"name":"password","required":true}]`),
		created,
	}}

	id, err := f.run(func(rest *client.RESTClient) client.FlowAnswer { return client.FieldAnswer(rest, values, files) })
	if err != nil || id != "E1" {
		t.Fatalf("id = %q, err = %v", id, err)
	}
	want := []map[string]any{
		{"host": "kitchen.local", "port": float64(6053)},
		{"noise_psk": "key="},
		{"password": "hunter2"},
	}
	if !reflect.DeepEqual(f.submitted, want) {
		t.Errorf("submitted = %v", f.submitted)
	}
}

func TestESPHomeMissingKeyIsNamed(t *testing.T) {
	m := esphomeModel{Host: types.StringValue("kitchen.local"), Port: types.Int64Value(6053)}
	values, files, _ := m.flowInput(context.Background())
	answer := client.FieldAnswer(nil, values, files)

	_, err := answer(context.Background(), client.FlowForm{
		Flow: "config flow for esphome", StepID: "encryption_key",
		Fields: []client.FormField{{Name: "noise_psk", Required: true}},
	})
	if want := `config flow for esphome asked for "noise_psk" in step "encryption_key", which is not set`; err == nil || err.Error() != want {
		t.Errorf("err = %v", err)
	}
}

// mqtt returns an MQTT model with HA's defaults, as the plan has it.
func mqtt() mqttModel {
	msg := func(payload string) *mqttMessageModel {
		return &mqttMessageModel{
			Enabled: types.BoolValue(true), Topic: types.StringValue(mqttStatusTopic),
			Payload: types.StringValue(payload), QoS: types.Int64Value(0), Retain: types.BoolValue(false),
		}
	}
	return mqttModel{
		Broker:          types.StringValue("mqtt.local"),
		Port:            types.Int64Value(1883),
		Username:        types.StringNull(),
		Password:        types.StringNull(),
		ClientID:        types.StringNull(),
		Keepalive:       types.Int64Value(60),
		Protocol:        types.StringValue("3.1.1"),
		Transport:       types.StringValue("tcp"),
		WSPath:          types.StringNull(),
		WSHeaders:       types.MapNull(types.StringType),
		Discovery:       types.BoolValue(true),
		DiscoveryPrefix: types.StringValue("homeassistant"),
		BirthMessage:    msg("online"),
		WillMessage:     msg("offline"),
	}
}

func TestMQTTFlowInput(t *testing.T) {
	base := map[string]any{
		"broker": "mqtt.local", "port": int64(1883), "keepalive": int64(60), "protocol": "3.1.1",
		"transport": "tcp", "set_ca_cert": "off", "set_client_cert": false, "tls_insecure": false,
	}
	with := func(kv ...any) map[string]any {
		m := map[string]any{}
		for k, v := range base {
			m[k] = v
		}
		for i := 0; i < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	}
	for name, tc := range map[string]struct {
		edit   func(*mqttModel)
		values map[string]any
		files  map[string]string
	}{
		"defaults": {edit: func(*mqttModel) {}, values: base},
		"credentials": {
			edit: func(m *mqttModel) {
				m.Username, m.Password, m.ClientID = types.StringValue("ha"), types.StringValue("pw"), types.StringValue("c1")
			},
			values: with("username", "ha", "password", "pw", "client_id", "c1"),
		},
		"websockets": {
			edit: func(m *mqttModel) {
				m.Transport, m.WSPath = types.StringValue("websockets"), types.StringValue("/mqtt")
				m.WSHeaders = types.MapValueMust(types.StringType, map[string]attr.Value{"X-Auth": types.StringValue("a")})
			},
			values: with("transport", "websockets", "ws_path", "/mqtt", "ws_headers", `{"X-Auth":"a"}`),
		},
		"tls with system CAs": {
			edit: func(m *mqttModel) {
				m.TLS = &mqttTLSModel{CACertificate: types.StringNull(), ClientCertificate: types.StringNull(), ClientKey: types.StringNull(), Insecure: types.BoolValue(true)}
			},
			values: with("set_ca_cert", "auto", "tls_insecure", true),
		},
		"tls with certificates": {
			edit: func(m *mqttModel) {
				m.TLS = &mqttTLSModel{
					CACertificate: types.StringValue("CA"), ClientCertificate: types.StringValue("CERT"),
					ClientKey: types.StringValue("KEY"), Insecure: types.BoolValue(false),
				}
			},
			values: with("set_ca_cert", "custom", "set_client_cert", true),
			files:  map[string]string{"certificate": "CA", "client_cert": "CERT", "client_key": "KEY"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := mqtt()
			tc.edit(&m)
			values, files, diags := m.flowInput(context.Background())
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !reflect.DeepEqual(values, tc.values) {
				t.Errorf("values = %v\nwant %v", values, tc.values)
			}
			if tc.files == nil {
				tc.files = map[string]string{}
			}
			if !reflect.DeepEqual(files, tc.files) {
				t.Errorf("files = %v", files)
			}
		})
	}
}

func TestMQTTOptionInput(t *testing.T) {
	m := mqtt()
	m.WillMessage.Enabled = types.BoolValue(false)
	want := map[string]any{
		"discovery": true, "discovery_prefix": "homeassistant",
		"birth_enable": true, "birth_topic": "homeassistant/status", "birth_payload": "online", "birth_qos": int64(0), "birth_retain": false,
		"will_enable": false, "will_topic": "homeassistant/status", "will_payload": "offline", "will_qos": int64(0), "will_retain": false,
	}
	if got := m.optionInput(); !reflect.DeepEqual(got, want) {
		t.Errorf("options = %v", got)
	}
}

// The broker forms of HA 2026.4: the advanced fields and then the file fields appear only after
// a post that asks for them.
const (
	mqttFormBasic = `[{"name":"broker","required":true,"default":null},{"name":"port","required":true,"default":1883},
		{"name":"username","optional":true},{"name":"password","optional":true},{"name":"advanced_options","optional":true}]`
	mqttFormAdvanced = `[{"name":"broker","required":true,"default":"mqtt.local"},{"name":"port","required":true,"default":1883},
		{"name":"username","optional":true},{"name":"password","optional":true},
		{"name":"client_id","optional":true},{"name":"keepalive","optional":true},{"name":"set_client_cert","optional":true,"default":false},
		{"name":"set_ca_cert","optional":true,"default":"off"},{"name":"tls_insecure","optional":true},
		{"name":"protocol","optional":true},{"name":"transport","optional":true}]`
	mqttFormFiles = `[{"name":"broker","required":true,"default":"mqtt.local"},{"name":"port","required":true,"default":1883},
		{"name":"username","optional":true},{"name":"password","optional":true},
		{"name":"client_id","optional":true},{"name":"keepalive","optional":true},{"name":"set_client_cert","optional":true,"default":true},
		{"name":"client_cert","optional":true},{"name":"client_key","optional":true},{"name":"client_key_password","optional":true},
		{"name":"set_ca_cert","optional":true,"default":"custom"},{"name":"certificate","optional":true},{"name":"tls_insecure","optional":true},
		{"name":"protocol","optional":true},{"name":"transport","optional":true}]`
	// The broker form of HA 2026.9: one form with the section other_settings.
	mqttFormSection = `[{"name":"broker","required":true},{"name":"port","required":true,"default":1883},
		{"name":"protocol","required":true,"default":"3.1.1"},{"name":"username","optional":true},{"name":"password","optional":true},
		{"name":"other_settings","required":true,"type":"expandable","expanded":false,"schema":[
			{"name":"client_id","optional":true},{"name":"keepalive","optional":true},{"name":"set_client_cert","required":true},
			{"name":"client_cert","optional":true},{"name":"client_key","optional":true},{"name":"client_key_password","optional":true},
			{"name":"set_ca_cert","required":true},{"name":"certificate","optional":true},{"name":"tls_insecure","optional":true},
			{"name":"transport","required":true,"default":"tcp"},{"name":"ws_path","optional":true},{"name":"ws_headers","optional":true}]}]`
)

// mqttRun answers the forms with the MQTT model's input, as the resource does.
func mqttRun(t *testing.T, m mqttModel, forms ...string) *fakeConfigFlow {
	t.Helper()
	values, files, _ := m.flowInput(context.Background())
	f := &fakeConfigFlow{t: t}
	for _, schema := range forms {
		f.replies = append(f.replies, form("broker", schema))
	}
	f.replies = append(f.replies, created)
	if _, err := f.run(func(rest *client.RESTClient) client.FlowAnswer {
		return mqttAnswer(client.FieldAnswer(rest, values, files), values, files)
	}); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestMQTTDefaultsNeedNoAdvancedForm(t *testing.T) {
	f := mqttRun(t, mqtt(), mqttFormBasic)
	want := []map[string]any{{"broker": "mqtt.local", "port": float64(1883)}}
	if !reflect.DeepEqual(f.submitted, want) {
		t.Errorf("submitted = %v", f.submitted)
	}
}

func TestMQTTAdvancedForms(t *testing.T) {
	m := mqtt()
	m.Keepalive = types.Int64Value(30)
	m.TLS = &mqttTLSModel{
		CACertificate: types.StringValue("CA"), ClientCertificate: types.StringValue("CERT"),
		ClientKey: types.StringValue("KEY"), Insecure: types.BoolValue(false),
	}

	f := mqttRun(t, m, mqttFormBasic, mqttFormAdvanced, mqttFormFiles)

	advanced := map[string]any{
		"broker": "mqtt.local", "port": float64(1883), "keepalive": float64(30), "protocol": "3.1.1", "transport": "tcp",
		"set_ca_cert": "custom", "set_client_cert": true, "tls_insecure": false,
	}
	files := map[string]any{"client_cert": "file1", "client_key": "file2", "certificate": "file3"}
	withFiles := map[string]any{}
	for _, m := range []map[string]any{advanced, files} {
		for k, v := range m {
			withFiles[k] = v
		}
	}
	want := []map[string]any{
		{"broker": "mqtt.local", "port": float64(1883), "advanced_options": true},
		advanced,
		withFiles,
	}
	if !reflect.DeepEqual(f.submitted, want) {
		t.Errorf("submitted = %v\nwant      %v", f.submitted, want)
	}
	// Uploads follow the order of the form's fields.
	if !reflect.DeepEqual(f.uploads, []string{"CERT", "KEY", "CA"}) {
		t.Errorf("uploads = %v", f.uploads)
	}
}

func TestMQTTSectionForm(t *testing.T) {
	m := mqtt()
	m.Username, m.Password = types.StringValue("ha"), types.StringValue("pw")
	m.TLS = &mqttTLSModel{
		CACertificate: types.StringNull(), ClientCertificate: types.StringNull(),
		ClientKey: types.StringNull(), Insecure: types.BoolValue(false),
	}

	f := mqttRun(t, m, mqttFormSection)

	want := []map[string]any{{
		"broker": "mqtt.local", "port": float64(1883), "protocol": "3.1.1", "username": "ha", "password": "pw",
		"other_settings": map[string]any{
			"keepalive": float64(60), "set_client_cert": false, "set_ca_cert": "auto", "tls_insecure": false, "transport": "tcp",
		},
	}}
	if !reflect.DeepEqual(f.submitted, want) {
		t.Errorf("submitted = %v", f.submitted)
	}
}

// mqttConfig is a configuration of homeassistant_mqtt with the given attributes; the others are
// null.
func mqttConfig(t *testing.T, attrs map[string]tftypes.Value) tfsdk.Config {
	t.Helper()
	var resp resource.SchemaResponse
	NewMQTT().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	typ := resp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for name, at := range typ.AttributeTypes {
		vals[name] = tftypes.NewValue(at, nil)
	}
	for name, v := range attrs {
		vals[name] = v
	}
	return tfsdk.Config{Schema: resp.Schema, Raw: tftypes.NewValue(typ, vals)}
}

func TestValidateMQTT(t *testing.T) {
	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
	tlsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"ca_certificate": tftypes.String, "client_certificate": tftypes.String, "client_key": tftypes.String, "insecure": tftypes.Bool,
	}}
	tls := func(cert, key tftypes.Value) tftypes.Value {
		return tftypes.NewValue(tlsType, map[string]tftypes.Value{
			"ca_certificate": tftypes.NewValue(tftypes.String, nil), "client_certificate": cert,
			"client_key": key, "insecure": tftypes.NewValue(tftypes.Bool, nil),
		})
	}
	null := tftypes.NewValue(tftypes.String, nil)
	for name, tc := range map[string]struct {
		attrs map[string]tftypes.Value
		want  string
	}{
		"minimal":                   {attrs: map[string]tftypes.Value{"broker": str("b")}},
		"websockets":                {attrs: map[string]tftypes.Value{"transport": str("websockets"), "ws_path": str("/mqtt")}},
		"ws_path without websocket": {attrs: map[string]tftypes.Value{"ws_path": str("/mqtt")}, want: `ws_path needs transport = "websockets"`},
		"unknown transport":         {attrs: map[string]tftypes.Value{"transport": tftypes.NewValue(tftypes.String, tftypes.UnknownValue), "ws_path": str("/")}},
		"client certificate":        {attrs: map[string]tftypes.Value{"tls": tls(str("C"), str("K"))}},
		"tls without client cert":   {attrs: map[string]tftypes.Value{"tls": tls(null, null)}},
		"certificate without key":   {attrs: map[string]tftypes.Value{"tls": tls(str("C"), null)}, want: "must be set together"},
		"key without certificate":   {attrs: map[string]tftypes.Value{"tls": tls(null, str("K"))}, want: "must be set together"},
	} {
		t.Run(name, func(t *testing.T) {
			diags := validateMQTT(context.Background(), resource.ValidateConfigRequest{Config: mqttConfig(t, tc.attrs)})
			checkDiags(t, diags, tc.want)
		})
	}
}
