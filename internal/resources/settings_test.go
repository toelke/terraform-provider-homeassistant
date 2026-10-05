package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlagChange(t *testing.T) {
	null, yes, no := types.BoolNull(), types.BoolValue(true), types.BoolValue(false)
	tests := []struct {
		name     string
		to, from types.Bool
		want     any // "unchanged" if the field is not written
	}{
		{"set by the user", yes, no, "user"},
		{"set when it was unmanaged", yes, null, "user"},
		{"already set by the user", yes, yes, "unchanged"},
		{"cleared where the user set it", no, yes, nil},
		{"false where the user did not set it", no, no, "unchanged"},
		{"reset on unmanage where the user set it", null, yes, nil},
		{"left alone on unmanage where false", null, no, "unchanged"},
		{"unmanaged", null, null, "unchanged"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := map[string]any{}
			flagChange(fields, "disabled_by", tt.to, tt.from)
			got, ok := fields["disabled_by"]
			if !ok {
				got = "unchanged"
			}
			if got != tt.want {
				t.Errorf("disabled_by = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegistryFlag(t *testing.T) {
	user, entry := "user", "config_entry"
	if got := registryFlag(types.BoolValue(false), types.BoolNull(), &user); !got.ValueBool() {
		t.Errorf(`newly managed flag with "user" = %v, want true`, got)
	}
	if got := registryFlag(types.BoolValue(false), types.BoolNull(), &entry); got.IsNull() || got.ValueBool() {
		t.Errorf(`newly managed flag with "config_entry" = %v, want false`, got)
	}
	if got := registryFlag(types.BoolValue(false), types.BoolValue(true), nil); !got.ValueBool() {
		t.Errorf("managed flag = %v, want it kept", got)
	}
	if got := registryFlag(types.BoolNull(), types.BoolNull(), &user); !got.IsNull() {
		t.Errorf("unmanaged flag = %v, want null", got)
	}
}
