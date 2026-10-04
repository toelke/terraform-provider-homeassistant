package resources

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewInputDatetime returns the `homeassistant_input_datetime` resource.
func NewInputDatetime() resource.Resource {
	return newHelper(helperSpec{
		domain:      "input_datetime",
		description: "A date and/or time helper (`input_datetime`).",
		fields: map[string]schema.Attribute{
			"has_date": schema.BoolAttribute{
				Description: "Whether the helper holds a date. At least one of `has_date` and `has_time` must " +
					"be true. Defaults to `false`.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"has_time": schema.BoolAttribute{
				Description: "Whether the helper holds a time of day. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"initial": schema.StringAttribute{
				Description: "Value the helper starts with when Home Assistant starts: `2026-12-24` with only " +
					"a date, `18:30` or `18:30:00` with only a time, `2026-12-24 18:30:00` with both. Unset, it " +
					"restores the value it had before the restart. The provider never manages the current value.",
				Optional: true,
			},
		},
		validate: validateInputDatetime,
	})()
}

// validateInputDatetime checks that at least one of `has_date` and `has_time` is true, and that
// `initial` has the matching format.
func validateInputDatetime(attrs map[string]attr.Value) diag.Diagnostics {
	var diags diag.Diagnostics
	hasDate, dateSet, dateKnown := configValue[types.Bool](attrs, "has_date")
	hasTime, timeSet, timeKnown := configValue[types.Bool](attrs, "has_time")
	if !dateKnown || !timeKnown {
		return diags
	}
	date, tod := dateSet && hasDate.ValueBool(), timeSet && hasTime.ValueBool()
	if !date && !tod {
		diags.AddAttributeError(path.Root("has_date"), "Invalid Attribute Combination",
			"At least one of has_date and has_time must be true.")
		return diags
	}
	if initial, set, known := configValue[types.String](attrs, "initial"); set && known {
		if want, ok := validDatetime(initial.ValueString(), date, tod); !ok {
			diags.AddAttributeError(path.Root("initial"), "Invalid Attribute Value",
				fmt.Sprintf("initial must have the format %s, got %q.", want, initial.ValueString()))
		}
	}
	return diags
}

// validDatetime reports whether s is a valid initial value for an input_datetime with a date,
// a time, or both. want describes the expected format.
func validDatetime(s string, date, tod bool) (want string, ok bool) {
	const dateLayout = "2006-01-02"
	timeLayouts := []string{"15:04:05", "15:04"}
	var layouts []string
	switch {
	case date && tod:
		want = "YYYY-MM-DD HH:MM:SS"
		for _, sep := range []string{" ", "T"} {
			for _, t := range timeLayouts {
				layouts = append(layouts, dateLayout+sep+t)
			}
		}
	case date:
		want, layouts = "YYYY-MM-DD", []string{dateLayout}
	default:
		want, layouts = "HH:MM:SS", timeLayouts
	}
	for _, l := range layouts {
		if _, err := time.Parse(l, s); err == nil {
			return want, true
		}
	}
	return want, false
}
