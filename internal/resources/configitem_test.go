package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestConfigItemIDValidator(t *testing.T) {
	cases := map[string]struct {
		value   types.String
		wantErr bool
	}{
		"slug":               {types.StringValue("motion_bedroom"), false},
		"digits":             {types.StringValue("1700000000000"), false},
		"leading underscore": {types.StringValue("_x"), false},
		"uppercase":          {types.StringValue("Motion"), true},
		"hyphen":             {types.StringValue("motion-bedroom"), true},
		"space":              {types.StringValue("motion bedroom"), true},
		"empty":              {types.StringValue(""), true},
		"null":               {types.StringNull(), false},
		"unknown":            {types.StringUnknown(), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var resp validator.StringResponse
			configItemIDValidator.ValidateString(t.Context(), validator.StringRequest{ConfigValue: tc.value}, &resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("error = %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestConfigItemConfigValidator(t *testing.T) {
	object := func(attrs map[string]attr.Value) types.Dynamic {
		typs := make(map[string]attr.Type, len(attrs))
		for k, v := range attrs {
			typs[k] = v.Type(t.Context())
		}
		return types.DynamicValue(types.ObjectValueMust(typs, attrs))
	}
	cases := map[string]struct {
		value   types.Dynamic
		wantErr bool
	}{
		"object":            {object(map[string]attr.Value{"alias": types.StringValue("A")}), false},
		"object with id":    {object(map[string]attr.Value{"id": types.StringValue("a")}), true},
		"unknown key value": {object(map[string]attr.Value{"alias": types.StringUnknown()}), false},
		"map with id": {
			types.DynamicValue(types.MapValueMust(types.StringType, map[string]attr.Value{"id": types.StringValue("a")})),
			true,
		},
		"string":     {types.DynamicValue(types.StringValue("x")), true},
		"null":       {types.DynamicNull(), false},
		"unknown":    {types.DynamicUnknown(), false},
		"wraps null": {types.DynamicValue(types.ObjectNull(nil)), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var resp validator.DynamicResponse
			configItemConfigValidator{}.ValidateDynamic(t.Context(), validator.DynamicRequest{ConfigValue: tc.value}, &resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("error = %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}
