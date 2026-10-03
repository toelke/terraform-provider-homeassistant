package dyntype

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// One case per known normalisation rule, plus the real changes it must not hide.
func TestSemanticallyEqualJSON(t *testing.T) {
	cases := map[string]struct {
		prior, read string
		want        bool
	}{
		"identical":                {`{"alias":"x"}`, `{"alias":"x"}`, true},
		"key order":                {`{"alias":"x","mode":"single"}`, `{"mode":"single","alias":"x"}`, true},
		"nested key order":         {`{"a":{"x":1,"y":[{"p":1,"q":2}]}}`, `{"a":{"y":[{"q":2,"p":1}],"x":1}}`, true},
		"integer vs float":         {`{"brightness":5}`, `{"brightness":5.0}`, true},
		"exponent":                 {`{"n":100}`, `{"n":1e2}`, true},
		"fraction":                 {`{"n":0.1}`, `{"n":0.10}`, true},
		"nested list of objects":   {`{"triggers":[{"trigger":"state","to":"on"}]}`, `{"triggers":[{"to":"on","trigger":"state"}]}`, true},
		"null dropped by HA":       {`{"alias":"x","description":null}`, `{"alias":"x"}`, true},
		"nested null dropped":      {`{"actions":[{"action":"light.turn_on","data":null}]}`, `{"actions":[{"action":"light.turn_on"}]}`, true},
		"empty object kept":        {`{"data":{}}`, `{"data":{}}`, true},
		"changed string":           {`{"alias":"x"}`, `{"alias":"y"}`, false},
		"changed number":           {`{"brightness":5}`, `{"brightness":5.5}`, false},
		"number vs string":         {`{"brightness":5}`, `{"brightness":"5"}`, false},
		"bool vs string":           {`{"enabled":true}`, `{"enabled":"true"}`, false},
		"list order":               {`{"entity_id":["a","b"]}`, `{"entity_id":["b","a"]}`, false},
		"list element added":       {`{"entity_id":["a"]}`, `{"entity_id":["a","b"]}`, false},
		"key added":                {`{"alias":"x"}`, `{"alias":"x","mode":"single"}`, false},
		"key removed":              {`{"alias":"x","mode":"single"}`, `{"alias":"x"}`, false},
		"null in list kept":        {`{"l":[null,1]}`, `{"l":[1]}`, false},
		"empty list vs absent":     {`{"conditions":[]}`, `{}`, false},
		"empty object vs absent":   {`{"data":{}}`, `{}`, false},
		"deep change":              {`{"a":[{"b":{"c":1}}]}`, `{"a":[{"b":{"c":2}}]}`, false},
		"large integers differ":    {`{"n":12345678901234567890}`, `{"n":12345678901234567891}`, false},
		"object vs list":           {`{"data":{}}`, `{"data":[]}`, false},
		"top-level scalar differs": {`1`, `2`, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := SemanticallyEqualJSON([]byte(tc.prior), []byte(tc.read))
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("SemanticallyEqualJSON(%s, %s) = %v, want %v", tc.prior, tc.read, got, tc.want)
			}
		})
	}
}

// The prior value as the user writes it in HCL, compared with what HA returns.
func TestDynamicSemanticEquals(t *testing.T) {
	ctx := context.Background()
	obj := func(attrs map[string]attr.Value) attr.Value {
		typs := make(map[string]attr.Type, len(attrs))
		for k, v := range attrs {
			typs[k] = v.Type(ctx)
		}
		return types.ObjectValueMust(typs, attrs)
	}
	tuple := func(elems ...attr.Value) attr.Value {
		typs := make([]attr.Type, len(elems))
		for i, v := range elems {
			typs[i] = v.Type(ctx)
		}
		return types.TupleValueMust(typs, elems)
	}

	hcl := NewValue(obj(map[string]attr.Value{
		"alias":       types.StringValue("Hall light"),
		"description": types.DynamicNull(),
		"triggers": tuple(obj(map[string]attr.Value{
			"trigger": types.StringValue("state"),
			"for":     obj(map[string]attr.Value{"minutes": types.NumberValue(big.NewFloat(5))}),
		})),
		"actions": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")}),
	}))

	cases := map[string]struct {
		prior Value
		read  string
		want  bool
	}{
		"HA representation": {hcl, `{"actions":["a"],"triggers":[{"for":{"minutes":5.0},"trigger":"state"}],"alias":"Hall light"}`, true},
		"changed in HA UI":  {hcl, `{"actions":["a"],"triggers":[{"for":{"minutes":6},"trigger":"state"}],"alias":"Hall light"}`, false},
		"alias renamed":     {hcl, `{"actions":["a"],"triggers":[{"for":{"minutes":5},"trigger":"state"}],"alias":"Hall"}`, false},
		"float64 fraction":  {NewValue(obj(map[string]attr.Value{"n": types.Float64Value(0.1)})), `{"n":0.1}`, true},
		"int64":             {NewValue(obj(map[string]attr.Value{"n": types.Int64Value(7)})), `{"n":7.0}`, true},
		"null prior":        {NewNull(), `{}`, false},
		"unknown prior":     {NewUnknown(), `{}`, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			read, err := FromJSON([]byte(tc.read))
			if err != nil {
				t.Fatal(err)
			}
			got, diags := tc.prior.DynamicSemanticEquals(ctx, read)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestJSONRoundTrip(t *testing.T) {
	in := `{"actions":[{"action":"light.turn_on","data":null}],"alias":"x","big":12345678901234567890,"n":0.5,"on":true,"z":[]}`
	v, err := FromJSON([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	out, err := v.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != in {
		t.Errorf("round trip:\n got %s\nwant %s", out, in)
	}
}

// A value built from HA's JSON must survive encoding for the wire, including nulls inside objects.
func TestFromJSONEncodesForTerraform(t *testing.T) {
	ctx := context.Background()
	v, err := FromJSON([]byte(`{"a":null,"b":[1,"x",null,{"c":true}],"d":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	tv, err := v.ToTerraformValue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	dv, err := tfprotov6.NewDynamicValue(tftypes.DynamicPseudoType, tv)
	if err != nil {
		t.Fatal(err)
	}
	back, err := dv.Unmarshal(tftypes.DynamicPseudoType)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Type{}.ValueFromTerraform(ctx, back)
	if err != nil {
		t.Fatal(err)
	}
	eq, diags := v.DynamicSemanticEquals(ctx, got.(Value))
	if diags.HasError() || !eq {
		t.Errorf("value changed in transit: %v %v", got, diags)
	}
}

func TestJSONRejectsNullAndUnknown(t *testing.T) {
	for name, v := range map[string]Value{"null": NewNull(), "unknown": NewUnknown()} {
		if _, err := v.JSON(); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestFromJSONRejectsInvalid(t *testing.T) {
	for _, in := range []string{``, `{`, `{} {}`} {
		if _, err := FromJSON([]byte(in)); err == nil {
			t.Errorf("FromJSON(%q): expected an error", in)
		}
	}
}

func TestTypeRoundTrip(t *testing.T) {
	ctx := context.Background()
	tv := tftypes.NewValue(tftypes.String, "x")
	v, err := Type{}.ValueFromTerraform(ctx, tv)
	if err != nil {
		t.Fatal(err)
	}
	dv, ok := v.(Value)
	if !ok {
		t.Fatalf("got %T, want Value", v)
	}
	if !dv.Type(ctx).Equal(Type{}) {
		t.Error("value type is not Type")
	}
	if !dv.Equal(NewValue(types.StringValue("x"))) {
		t.Error("value differs after round trip")
	}
}
