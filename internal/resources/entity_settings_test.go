package resources

import (
	"context"
	"encoding/json"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// appearingEntity is a WSCommander whose entity is in the registry from the after-th `get` on.
type appearingEntity struct {
	after int
	gets  int
}

func (f *appearingEntity) Command(_ context.Context, typ string, params map[string]any, result any) error {
	if typ != "config/entity_registry/get" {
		return &client.WSError{Code: "unknown_command", Message: typ}
	}
	f.gets++
	if f.gets < f.after {
		return &client.WSError{Code: "not_found", Message: "Entity not found"}
	}
	b, _ := json.Marshal(map[string]any{"entity_id": params["entity_id"], "platform": "moon"})
	return json.Unmarshal(b, result)
}

func fastPolls(t *testing.T) {
	t.Helper()
	interval, maxInterval := pollInterval, maxPollInterval
	pollInterval, maxPollInterval = time.Millisecond, 2*time.Millisecond
	t.Cleanup(func() { pollInterval, maxPollInterval = interval, maxInterval })
}

func TestAwaitEntityWaitsUntilItAppears(t *testing.T) {
	fastPolls(t)
	f := &appearingEntity{after: 4}
	r := &entitySettingsResource{entities: client.Entities(f)}
	entry, err := r.awaitEntity(context.Background(), "sensor.moon_phase", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if f.gets != 4 || entry.Platform != "moon" {
		t.Errorf("gets = %d, entry = %+v", f.gets, entry)
	}
}

func TestAwaitEntityTimesOut(t *testing.T) {
	fastPolls(t)
	r := &entitySettingsResource{entities: client.Entities(&appearingEntity{after: 1 << 30})}
	_, err := r.awaitEntity(context.Background(), "sensor.typo", 20*time.Millisecond)
	if err == nil {
		t.Fatal("no error")
	}
	for _, want := range []string{"sensor.typo", "did not appear", "unique ID", "timeouts.create"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestEntitySettingsChanges(t *testing.T) {
	ctx := context.Background()
	labels := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("managed")})
	user := "user"
	managed := entitySettingsModel{
		EntityID: types.StringValue("sensor.moon_phase"),
		Name:     types.StringValue("Moon"),
		Labels:   labels,
		Hidden:   types.BoolValue(true),
		Disabled: types.BoolValue(false),
	}

	tests := []struct {
		name     string
		to, from entitySettingsModel
		want     map[string]any
	}{
		{
			name: "create writes only the set fields",
			to:   managed,
			want: map[string]any{"name": "Moon", "labels": []string{"managed"}, "hidden_by": &user, "disabled_by": nil},
		},
		{
			name: "destroy resets only the set fields",
			from: managed,
			want: map[string]any{"name": nil, "labels": []string{}, "hidden_by": nil, "disabled_by": nil},
		},
		{
			name: "update writes what changed",
			to: func() entitySettingsModel {
				m := managed
				m.Name = types.StringNull()
				m.Icon = types.StringValue("mdi:moon-full")
				return m
			}(),
			from: managed,
			want: map[string]any{"name": nil, "icon": "mdi:moon-full"},
		},
		{
			name: "no changes",
			to:   managed,
			from: managed,
			want: map[string]any{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := tt.to.changes(ctx, tt.from)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !maps.EqualFunc(got, tt.want, func(a, b any) bool {
				ja, _ := json.Marshal(a)
				jb, _ := json.Marshal(b)
				return string(ja) == string(jb)
			}) {
				t.Errorf("changes = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntitySettingsRefreshIsFieldGranular(t *testing.T) {
	user, integration, name, icon := "user", "integration", "Moon", "mdi:moon-full"
	entry := client.EntityEntry{
		EntityID: "sensor.moon_phase", Name: &name, Icon: &icon, Labels: []string{"other"},
		HiddenBy: &integration, DisabledBy: &user, Platform: "moon", OriginalName: &name,
	}
	m := entitySettingsModel{
		EntityID: types.StringValue("sensor.moon_phase"),
		Name:     types.StringValue("Old"),
		Hidden:   types.BoolValue(true),
		Disabled: types.BoolValue(false),
	}
	if diags := m.refresh(context.Background(), entry); diags.HasError() {
		t.Fatal(diags)
	}

	if m.Name.ValueString() != "Moon" || m.Platform.ValueString() != "moon" || m.OriginalName.ValueString() != "Moon" {
		t.Errorf("managed and computed attributes not read: %+v", m)
	}
	if !m.Icon.IsNull() || !m.Labels.IsNull() || !m.AreaID.IsNull() {
		t.Errorf("unmanaged attributes were read: %+v", m)
	}
	if m.Hidden.ValueBool() {
		t.Error(`hidden_by "integration" read as hidden = true`)
	}
	if !m.Disabled.ValueBool() {
		t.Error(`disabled_by "user" read as disabled = false`)
	}
	if !m.DeviceID.IsNull() {
		t.Errorf("device_id = %v, want null", m.DeviceID)
	}
}
