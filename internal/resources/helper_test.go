package resources

import (
	"context"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// inputBooleanType returns the object type of `homeassistant_input_boolean`.
func inputBooleanType(t *testing.T) types.ObjectType {
	t.Helper()
	resp := &resource.SchemaResponse{}
	NewInputBoolean().Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.Schema.Type().(types.ObjectType)
}

func TestHelperFieldsLeavesOutUnset(t *testing.T) {
	typ := inputBooleanType(t)
	obj := types.ObjectValueMust(typ.AttrTypes, map[string]attr.Value{
		"id":        types.StringValue("guest_mode"),
		"entity_id": types.StringUnknown(),
		"name":      types.StringValue("Guest mode"),
		"icon":      types.StringNull(),
		"initial":   types.BoolValue(false),
	})
	got, err := helperFields(obj)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"name": "Guest mode", "initial": false}
	if !maps.Equal(got, want) {
		t.Errorf("fields = %v, want %v", got, want)
	}
}

func TestHelperObject(t *testing.T) {
	typ := inputBooleanType(t)
	r := NewInputBoolean().(*helperResource)
	got, diags := r.helperObject(typ, client.Helper{"id": "guest_mode", "name": "Guest mode", "initial": true})
	if diags.HasError() {
		t.Fatal(diags)
	}
	want := types.ObjectValueMust(typ.AttrTypes, map[string]attr.Value{
		"id":        types.StringValue("guest_mode"),
		"entity_id": types.StringValue("input_boolean.guest_mode"),
		"name":      types.StringValue("Guest mode"),
		"icon":      types.StringNull(),
		"initial":   types.BoolValue(true),
	})
	if !got.Equal(want) {
		t.Errorf("object = %v, want %v", got, want)
	}
}
