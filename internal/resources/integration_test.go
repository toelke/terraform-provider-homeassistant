package resources

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

func mustDyn(t *testing.T, js string) dyntype.Value {
	t.Helper()
	v, err := dyntype.FromJSON([]byte(js))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestFlowStepsMergesSensitiveSteps(t *testing.T) {
	steps := mustDyn(t, `{"user": {"host": "h", "port": 80}, "confirm": {}}`)
	sensitive := mustDyn(t, `{"user": {"password": "hunter2"}, "credentials": {"token": "t"}}`)

	got, diags := flowSteps(steps, sensitive)
	if diags.HasError() {
		t.Fatal(diags)
	}
	want := []client.FlowStep{
		{StepID: "confirm", Data: map[string]any{}},
		{StepID: "credentials", SensitiveData: map[string]any{"token": "t"}},
		{StepID: "user", Data: map[string]any{"host": "h", "port": float64(80)}, SensitiveData: map[string]any{"password": "hunter2"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("flowSteps = %#v\nwant %#v", got, want)
	}

	got, diags = flowSteps(steps, dyntype.NewNull())
	if diags.HasError() || len(got) != 2 || got[1].SensitiveData != nil {
		t.Errorf("without sensitive_steps: %#v, %v", got, diags)
	}
}

func TestStepsValidator(t *testing.T) {
	cases := map[string]struct {
		value     dyntype.Value
		sensitive bool
		wantErr   string
	}{
		"steps":                {value: mustDyn(t, `{"user": {"host": "h"}}`)},
		"empty step":           {value: mustDyn(t, `{"user": {}}`)},
		"null":                 {value: dyntype.NewNull()},
		"unknown":              {value: dyntype.Value{DynamicValue: basetypes.NewDynamicUnknown()}},
		"no steps":             {value: mustDyn(t, `{}`), wantErr: "At least one step"},
		"no sensitive steps":   {value: mustDyn(t, `{}`), sensitive: true},
		"not an object":        {value: mustDyn(t, `["user"]`), wantErr: "Must be an object"},
		"step not an object":   {value: mustDyn(t, `{"user": "h"}`), wantErr: `Step "user" must be an object of fields, got`},
		"secret not an object": {value: mustDyn(t, `{"user": "hunter2"}`), sensitive: true, wantErr: `Step "user" must be an object of fields.`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var resp validator.DynamicResponse
			stepsValidator{sensitive: tc.sensitive}.ValidateDynamic(t.Context(),
				validator.DynamicRequest{ConfigValue: tc.value.DynamicValue}, &resp)
			if tc.wantErr == "" {
				if resp.Diagnostics.HasError() {
					t.Errorf("unexpected error: %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Detail(), tc.wantErr) {
				t.Fatalf("diagnostics = %v, want %q", resp.Diagnostics, tc.wantErr)
			}
			if tc.sensitive && strings.Contains(resp.Diagnostics[0].Detail(), "hunter2") {
				t.Errorf("diagnostic leaks the sensitive value: %s", resp.Diagnostics[0].Detail())
			}
		})
	}
}
