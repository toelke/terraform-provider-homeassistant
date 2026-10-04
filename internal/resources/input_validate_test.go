package resources

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// checkDiags fails unless diags has exactly one error, whose detail contains want, or no
// error if want is empty.
func checkDiags(t *testing.T, diags diag.Diagnostics, want string) {
	t.Helper()
	errs := diags.Errors()
	switch {
	case want == "" && len(errs) > 0:
		t.Errorf("unexpected errors: %v", errs)
	case want != "" && (len(errs) != 1 || !strings.Contains(errs[0].Detail(), want)):
		t.Errorf("errors = %v, want one containing %q", errs, want)
	}
}

func strs(ss ...string) types.List {
	elems := make([]attr.Value, len(ss))
	for i, s := range ss {
		elems[i] = types.StringValue(s)
	}
	return types.ListValueMust(types.StringType, elems)
}

func TestValidateInputNumber(t *testing.T) {
	f := types.Float64Value
	for _, tc := range []struct {
		name  string
		attrs map[string]attr.Value
		want  string
	}{
		{"valid", map[string]attr.Value{"min": f(0), "max": f(10), "step": f(0.5), "initial": f(10)}, ""},
		{"min equals max", map[string]attr.Value{"min": f(5), "max": f(5)}, "max must be greater than min"},
		{"min above max", map[string]attr.Value{"min": f(6), "max": f(5)}, "max must be greater than min"},
		{"zero step", map[string]attr.Value{"min": f(0), "max": f(1), "step": f(0)}, "step must be greater than 0"},
		{"negative step", map[string]attr.Value{"min": f(0), "max": f(1), "step": f(-1)}, "step must be greater than 0"},
		{"initial below", map[string]attr.Value{"min": f(0), "max": f(1), "initial": f(-0.1)}, "initial must be between"},
		{"initial above", map[string]attr.Value{"min": f(0), "max": f(1), "initial": f(2)}, "initial must be between"},
		{"unknown max", map[string]attr.Value{"min": f(6), "max": types.Float64Unknown(), "initial": f(100)}, ""},
		{"null step", map[string]attr.Value{"min": f(0), "max": f(1), "step": types.Float64Null()}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) { checkDiags(t, validateInputNumber(tc.attrs), tc.want) })
	}
}

func TestValidateInputText(t *testing.T) {
	i, s := types.Int64Value, types.StringValue
	for _, tc := range []struct {
		name  string
		attrs map[string]attr.Value
		want  string
	}{
		{"valid", map[string]attr.Value{"min": i(2), "max": i(255), "initial": s("abc"), "pattern": s(`^[a-z]+$`)}, ""},
		{"defaults", map[string]attr.Value{"min": types.Int64Null(), "max": types.Int64Null(), "initial": s("")}, ""},
		{"negative min", map[string]attr.Value{"min": i(-1)}, "min must not be negative"},
		{"max too large", map[string]attr.Value{"max": i(256)}, "max must be at most 255"},
		{"min above max", map[string]attr.Value{"min": i(5), "max": i(4)}, "max must not be less than min"},
		{"min above default max", map[string]attr.Value{"min": i(101), "max": types.Int64Null()}, "max must not be less than min"},
		{"min equals max", map[string]attr.Value{"min": i(4), "max": i(4)}, ""},
		{"initial too short", map[string]attr.Value{"min": i(2), "initial": s("a")}, "between 2 and 100 characters"},
		{"initial too long", map[string]attr.Value{"max": i(3), "initial": s("äöüß")}, "between 0 and 3 characters long, got 4"},
		{"initial counts runes", map[string]attr.Value{"max": i(4), "initial": s("äöüß")}, ""},
		{"unknown max", map[string]attr.Value{"max": types.Int64Unknown(), "initial": s("abc")}, ""},
		{"bad pattern", map[string]attr.Value{"pattern": s(`[a-z`)}, "pattern is not a valid regular expression"},
		{"lookahead", map[string]attr.Value{"pattern": s(`^(?=.*\d)\w+$`)}, ""},
		{"python named groups", map[string]attr.Value{"pattern": s(`^(?P<c>\w)(?P=c)$`)}, ""},
		{"unknown pattern", map[string]attr.Value{"pattern": types.StringUnknown()}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) { checkDiags(t, validateInputText(tc.attrs), tc.want) })
	}
}

func TestValidateInputSelect(t *testing.T) {
	s := types.StringValue
	for _, tc := range []struct {
		name  string
		attrs map[string]attr.Value
		want  string
	}{
		{"valid", map[string]attr.Value{"options": strs("a", "b"), "initial": s("b")}, ""},
		{"no initial", map[string]attr.Value{"options": strs("a"), "initial": types.StringNull()}, ""},
		{"empty", map[string]attr.Value{"options": strs()}, "options must not be empty"},
		{"duplicate", map[string]attr.Value{"options": strs("a", "b", "a")}, `"a" appears more than once`},
		{"initial not an option", map[string]attr.Value{"options": strs("a", "b"), "initial": s("c")}, `initial must be one of options, got "c"`},
		{"unknown options", map[string]attr.Value{"options": types.ListUnknown(types.StringType), "initial": s("c")}, ""},
		{"unknown option", map[string]attr.Value{
			"options": types.ListValueMust(types.StringType, []attr.Value{s("a"), types.StringUnknown()}),
			"initial": s("c"),
		}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) { checkDiags(t, validateInputSelect(tc.attrs), tc.want) })
	}
}

func TestValidateInputDatetime(t *testing.T) {
	b, s := types.BoolValue, types.StringValue
	for _, tc := range []struct {
		name  string
		attrs map[string]attr.Value
		want  string
	}{
		{"neither", map[string]attr.Value{"has_date": b(false), "has_time": b(false)}, "At least one of has_date and has_time"},
		{"neither set", map[string]attr.Value{"has_date": types.BoolNull(), "has_time": types.BoolNull()}, "At least one of has_date and has_time"},
		{"unknown", map[string]attr.Value{"has_date": types.BoolUnknown(), "has_time": types.BoolNull()}, ""},
		{"date", map[string]attr.Value{"has_date": b(true), "initial": s("2026-12-24")}, ""},
		{"date with time", map[string]attr.Value{"has_date": b(true), "initial": s("2026-12-24 18:30")}, "format YYYY-MM-DD,"},
		{"invalid date", map[string]attr.Value{"has_date": b(true), "initial": s("2026-02-30")}, "format YYYY-MM-DD,"},
		{"time", map[string]attr.Value{"has_time": b(true), "initial": s("18:30")}, ""},
		{"time with seconds", map[string]attr.Value{"has_time": b(true), "initial": s("18:30:15")}, ""},
		{"invalid time", map[string]attr.Value{"has_time": b(true), "initial": s("25:00")}, "format HH:MM:SS,"},
		{"both", map[string]attr.Value{"has_date": b(true), "has_time": b(true), "initial": s("2026-12-24 18:30:00")}, ""},
		{"both with T", map[string]attr.Value{"has_date": b(true), "has_time": b(true), "initial": s("2026-12-24T18:30")}, ""},
		{"both with date only", map[string]attr.Value{"has_date": b(true), "has_time": b(true), "initial": s("2026-12-24")}, "format YYYY-MM-DD HH:MM:SS,"},
	} {
		t.Run(tc.name, func(t *testing.T) { checkDiags(t, validateInputDatetime(tc.attrs), tc.want) })
	}
}
