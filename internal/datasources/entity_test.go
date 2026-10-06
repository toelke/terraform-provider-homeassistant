package datasources

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

func TestEntityIDValidator(t *testing.T) {
	cases := map[string]struct {
		value   types.String
		wantErr bool
	}{
		"valid":          {types.StringValue("light.bedroom_ceiling"), false},
		"no dot":         {types.StringValue("sun"), true},
		"uppercase":      {types.StringValue("Sun.sun"), true},
		"path traversal": {types.StringValue("../config"), true},
		"slash":          {types.StringValue("sun.sun/x"), true},
		"null":           {types.StringNull(), false},
		"unknown":        {types.StringUnknown(), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var resp validator.StringResponse
			entityIDValidator.ValidateString(t.Context(), validator.StringRequest{ConfigValue: tc.value}, &resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("error = %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
			}
			// The error shows what a valid value looks like.
			if resp.Diagnostics.HasError() && !strings.Contains(resp.Diagnostics[0].Detail(), "\"light.bedroom_ceiling\"") {
				t.Errorf("detail = %q, want an example", resp.Diagnostics[0].Detail())
			}
		})
	}
}

func TestEntityModelKeepsAttributeTypes(t *testing.T) {
	m, err := entityModelFrom("sensor.temp", &client.EntityState{
		State: "21.5",
		Attributes: json.RawMessage(`{"friendly_name":"Temp","device_class":"temperature",` +
			`"unit_of_measurement":"°C","precision":1,"rising":true,"options":["a","b"]}`),
		LastChanged: "2026-10-04T10:00:00+00:00",
		LastUpdated: "2026-10-04T10:01:00+00:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.State.ValueString() != "21.5" || m.FriendlyName.ValueString() != "Temp" ||
		m.DeviceClass.ValueString() != "temperature" || m.UnitOfMeasurement.ValueString() != "°C" ||
		m.LastChanged.ValueString() != "2026-10-04T10:00:00+00:00" ||
		m.LastUpdated.ValueString() != "2026-10-04T10:01:00+00:00" {
		t.Errorf("model = %+v", m)
	}

	obj, ok := m.Attributes.UnderlyingValue().(basetypes.ObjectValue)
	if !ok {
		t.Fatalf("attributes = %T, want object", m.Attributes.UnderlyingValue())
	}
	attrs := obj.Attributes()
	if n, ok := attrs["precision"].(basetypes.NumberValue); !ok || n.ValueBigFloat().Cmp(big.NewFloat(1)) != 0 {
		t.Errorf("precision = %#v, want number 1", attrs["precision"])
	}
	if b, ok := attrs["rising"].(basetypes.BoolValue); !ok || !b.ValueBool() {
		t.Errorf("rising = %#v, want bool true", attrs["rising"])
	}
	if _, ok := attrs["options"].(basetypes.TupleValue); !ok {
		t.Errorf("options = %#v, want tuple", attrs["options"])
	}
}

func TestEntityModelMissingStringAttributesAreNull(t *testing.T) {
	m, err := entityModelFrom("sun.sun", &client.EntityState{
		State:      "above_horizon",
		Attributes: json.RawMessage(`{"device_class":5}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !m.FriendlyName.IsNull() || !m.UnitOfMeasurement.IsNull() {
		t.Errorf("absent attributes not null: %+v", m)
	}
	if !m.DeviceClass.IsNull() {
		t.Errorf("non-string device_class = %v, want null", m.DeviceClass)
	}
}

func TestEntityModelWithoutAttributes(t *testing.T) {
	m, err := entityModelFrom("sun.sun", &client.EntityState{State: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Attributes.UnderlyingValue().(basetypes.ObjectValue); !ok {
		t.Errorf("attributes = %#v, want empty object", m.Attributes)
	}
}
