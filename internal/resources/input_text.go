package resources

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Home Assistant's defaults and bounds for the length of an input_text.
const (
	inputTextMinDefault = 0
	inputTextMaxDefault = 100
	inputTextMaxLength  = 255
)

// NewInputText returns the `homeassistant_input_text` resource.
func NewInputText() resource.Resource {
	return newHelper(helperSpec{
		domain:      "input_text",
		description: "A text helper (`input_text`): a line of text, optionally restricted by length and a pattern.",
		fields: map[string]schema.Attribute{
			"min": schema.Int64Attribute{
				Description: "Minimum length, from 0 to `max`. Defaults to `0`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(inputTextMinDefault),
			},
			"max": schema.Int64Attribute{
				Description: "Maximum length, from `min` to 255. Defaults to `100`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(inputTextMaxDefault),
			},
			"initial": schema.StringAttribute{
				Description: "Value the helper starts with when Home Assistant starts. Its length must be " +
					"between `min` and `max`. Unset, it restores the value it had before the restart. The " +
					"provider never manages the current value.",
				Optional: true,
			},
			"pattern": schema.StringAttribute{
				Description: "Regular expression (Python syntax) that a value must match.",
				Optional:    true,
			},
			"mode": schema.StringAttribute{
				Description: "How the UI shows the helper: `text` or `password`. Defaults to `text`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("text"),
				Validators:  []validator.String{oneOfValidator{"text", "password"}},
			},
		},
		validate: validateInputText,
	})()
}

// validateInputText checks `0 ≤ min ≤ max ≤ 255`, that `initial` has a length in that range,
// and that `pattern` compiles.
func validateInputText(attrs map[string]attr.Value) diag.Diagnostics {
	var diags diag.Diagnostics
	lo, loSet, loKnown := configValue[types.Int64](attrs, "min")
	hi, hiSet, hiKnown := configValue[types.Int64](attrs, "max")
	minLen, maxLen := int64(inputTextMinDefault), int64(inputTextMaxDefault)
	if loSet {
		minLen = lo.ValueInt64()
	}
	if hiSet {
		maxLen = hi.ValueInt64()
	}
	if loKnown && minLen < 0 {
		diags.AddAttributeError(path.Root("min"), "Invalid Attribute Value",
			fmt.Sprintf("min must not be negative, got %d.", minLen))
	}
	if hiKnown && maxLen > inputTextMaxLength {
		diags.AddAttributeError(path.Root("max"), "Invalid Attribute Value",
			fmt.Sprintf("max must be at most %d, got %d.", inputTextMaxLength, maxLen))
	}
	rangeKnown := loKnown && hiKnown
	if rangeKnown && minLen > maxLen {
		diags.AddAttributeError(path.Root("max"), "Invalid Attribute Value",
			fmt.Sprintf("max must not be less than min, got min = %d and max = %d.", minLen, maxLen))
	}
	if initial, set, known := configValue[types.String](attrs, "initial"); set && known && rangeKnown {
		if n := int64(utf8.RuneCountInString(initial.ValueString())); n < minLen || n > maxLen {
			diags.AddAttributeError(path.Root("initial"), "Invalid Attribute Value", fmt.Sprintf(
				"initial must be between %d and %d characters long, got %d.", minLen, maxLen, n))
		}
	}
	if pattern, set, known := configValue[types.String](attrs, "pattern"); set && known {
		if err := compilePythonRegexp(pattern.ValueString()); err != nil {
			diags.AddAttributeError(path.Root("pattern"), "Invalid Attribute Value",
				fmt.Sprintf("pattern is not a valid regular expression: %v.", err))
		}
	}
	return diags
}

// pythonBackref matches Python's named backreference `(?P=name)`.
var pythonBackref = regexp.MustCompile(`\(\?P=(\w+)\)`)

// compilePythonRegexp reports whether HA can compile the pattern with Python's `re`. regexp2
// knows lookarounds and backreferences, which Go's regexp lacks; its RE2 option accepts
// `(?P<name>...)`, and Python's `(?P=name)` is rewritten into regexp2's `\k<name>`.
func compilePythonRegexp(pattern string) error {
	_, err := regexp2.Compile(pythonBackref.ReplaceAllString(pattern, `\k<$1>`), regexp2.RE2)
	return err
}
