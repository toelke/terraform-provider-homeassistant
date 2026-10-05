package resources

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/dyntype"
)

func TestURLPathValidator(t *testing.T) {
	cases := map[string]struct {
		value   types.String
		wantErr bool
	}{
		"hyphen":    {types.StringValue("my-home"), false},
		"no hyphen": {types.StringValue("home"), true},
		"null":      {types.StringNull(), false},
		"unknown":   {types.StringUnknown(), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var resp validator.StringResponse
			urlPathValidator{}.ValidateString(t.Context(), validator.StringRequest{ConfigValue: tc.value}, &resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("error = %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestDashboardMetaChanged(t *testing.T) {
	base := dashboardModel{
		Title:         types.StringValue("Home"),
		Icon:          types.StringNull(),
		ShowInSidebar: types.BoolValue(true),
		RequireAdmin:  types.BoolValue(false),
	}
	if base.metaChanged(base) {
		t.Error("identical metadata reported as changed")
	}
	// The config and IDs are not metadata.
	other := base
	other.DashboardID = types.StringValue("my_home")
	if base.metaChanged(other) {
		t.Error("dashboard_id reported as a metadata change")
	}
	for name, mutate := range map[string]func(*dashboardModel){
		"title":           func(m *dashboardModel) { m.Title = types.StringValue("Other") },
		"icon":            func(m *dashboardModel) { m.Icon = types.StringValue("mdi:home") },
		"show_in_sidebar": func(m *dashboardModel) { m.ShowInSidebar = types.BoolValue(false) },
		"require_admin":   func(m *dashboardModel) { m.RequireAdmin = types.BoolValue(true) },
	} {
		changed := base
		mutate(&changed)
		if !changed.metaChanged(base) {
			t.Errorf("%s change not detected", name)
		}
	}
}

// A dashboard state without a baseline (from v0.1.0, or after import) compares the config read
// back by semantic equality: key order and number format don't matter, a changed card does.
func TestDashboardConfigWithoutBaselineIsSemanticEquality(t *testing.T) {
	prior, err := dyntype.FromJSON([]byte(`{"views":[{"title":"Home","cards":[{"type":"tile","entity":"sun.sun","columns":6}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		read  string
		equal bool
	}{
		"same config":      {`{"views":[{"cards":[{"columns":6.0,"entity":"sun.sun","type":"tile"}],"title":"Home"}]}`, true},
		"edited in the UI": {`{"views":[{"title":"Home","cards":[{"type":"tile","entity":"sun.sun","columns":12}]}]}`, false},
		"view added in HA": {`{"views":[{"title":"Home","cards":[{"type":"tile","entity":"sun.sun","columns":6}]},{"title":"New"}]}`, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := refreshedConfig(prior, json.RawMessage(tc.read), nil)
			if err != nil {
				t.Fatal(err)
			}
			if eq, diags := prior.DynamicSemanticEquals(t.Context(), got); diags.HasError() || eq != tc.equal {
				t.Errorf("semantically equal = %v (%v), want %v", eq, diags, tc.equal)
			}
		})
	}
}
