// Package dyntype implements the custom framework type for dynamic config (ADR-0005). Values
// are compared by semantic equality (ADR-0006): a value read back from Home Assistant that
// normalises to the same JSON as the prior value keeps the prior value, so the plan stays empty.
package dyntype

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ basetypes.DynamicTypable                    = Type{}
	_ basetypes.DynamicValuableWithSemanticEquals = Value{}
)

// Type is the attribute type of a dynamic config attribute. Use it as the CustomType of a
// schema.DynamicAttribute.
type Type struct {
	basetypes.DynamicType
}

func (t Type) String() string {
	return "dyntype.Type"
}

func (t Type) Equal(o attr.Type) bool {
	_, ok := o.(Type)
	return ok
}

func (t Type) ValueType(context.Context) attr.Value {
	return Value{}
}

func (t Type) ValueFromDynamic(_ context.Context, in basetypes.DynamicValue) (basetypes.DynamicValuable, diag.Diagnostics) {
	return Value{DynamicValue: in}, nil
}

func (t Type) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	v, err := t.DynamicType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	dv, ok := v.(basetypes.DynamicValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", v)
	}
	return Value{DynamicValue: dv}, nil
}

// Value is a dynamic config value.
type Value struct {
	basetypes.DynamicValue
}

// NewValue wraps a known framework value, e.g. an object built in a test.
func NewValue(v attr.Value) Value {
	return Value{DynamicValue: basetypes.NewDynamicValue(v)}
}

// NewNull returns a null dynamic config value.
func NewNull() Value {
	return Value{DynamicValue: basetypes.NewDynamicNull()}
}

// NewUnknown returns an unknown dynamic config value.
func NewUnknown() Value {
	return Value{DynamicValue: basetypes.NewDynamicUnknown()}
}

func (v Value) Type(context.Context) attr.Type {
	return Type{}
}

func (v Value) Equal(o attr.Value) bool {
	other, ok := o.(Value)
	return ok && v.DynamicValue.Equal(other.DynamicValue)
}

// DynamicSemanticEquals reports whether both values normalise to the same JSON. The framework
// calls it with the prior value as receiver and the value read back from HA as argument.
func (v Value) DynamicSemanticEquals(_ context.Context, newValuable basetypes.DynamicValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	other, ok := newValuable.(Value)
	if !ok {
		diags.AddError("Semantic equality check error",
			fmt.Sprintf("Expected value type %T but got %T. Please report this to the provider developers.", v, newValuable))
		return false, diags
	}
	if !isWhollyKnown(v.DynamicValue) || !isWhollyKnown(other.DynamicValue) {
		return v.Equal(other), nil
	}
	a, err := v.goValue()
	if err != nil {
		diags.AddError("Semantic equality check error", err.Error())
		return false, diags
	}
	b, err := other.goValue()
	if err != nil {
		diags.AddError("Semantic equality check error", err.Error())
		return false, diags
	}
	eq, err := semanticallyEqual(a, b)
	if err != nil {
		diags.AddError("Semantic equality check error", err.Error())
		return false, diags
	}
	return eq, nil
}

func isWhollyKnown(v basetypes.DynamicValue) bool {
	if v.IsNull() || v.IsUnknown() || v.IsUnderlyingValueNull() || v.IsUnderlyingValueUnknown() {
		return false
	}
	tv, err := v.ToTerraformValue(context.Background())
	return err == nil && tv.IsFullyKnown()
}
