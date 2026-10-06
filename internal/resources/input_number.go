package resources

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewInputNumber returns the `homeassistant_input_number` resource.
func NewInputNumber() resource.Resource {
	return newHelper(helperSpec{
		domain:      "input_number",
		description: "A number helper (`input_number`): a number in a range, shown as a slider or an input box.",
		fields: map[string]schema.Attribute{
			"min": schema.Float64Attribute{
				Description: "Lowest value.",
				Required:    true,
			},
			"max": schema.Float64Attribute{
				Description: "Highest value. Must be greater than `min`.",
				Required:    true,
			},
			"step": schema.Float64Attribute{
				Description: "Step between values. Must be greater than 0. Defaults to `1`.",
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(1),
			},
			"initial": schema.Float64Attribute{
				Description: "Value the helper starts with when Home Assistant starts, between `min` and `max`. " +
					"Unset, it restores the value it had before the restart. The provider never manages the " +
					"current value.",
				Optional: true,
			},
			"mode": schema.StringAttribute{
				Description: "How the UI shows the helper: `slider` or `box`. Defaults to `slider`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("slider"),
				Validators:  []validator.String{stringvalidator.OneOf("slider", "box")},
			},
			"unit_of_measurement": schema.StringAttribute{
				Description: "Unit of the value, e.g. `°C`.",
				Optional:    true,
			},
		},
		validate: validateInputNumber,
	})
}

// validateInputNumber checks `min < max`, `step > 0`, and that `initial` is within the range.
func validateInputNumber(attrs map[string]attr.Value) diag.Diagnostics {
	var diags diag.Diagnostics
	lo, loSet, loKnown := configValue[types.Float64](attrs, "min")
	hi, hiSet, hiKnown := configValue[types.Float64](attrs, "max")
	rangeKnown := loSet && loKnown && hiSet && hiKnown
	if rangeKnown && lo.ValueFloat64() >= hi.ValueFloat64() {
		diags.AddAttributeError(path.Root("max"), "Invalid Attribute Value", fmt.Sprintf(
			"max must be greater than min, got min = %v and max = %v.", lo.ValueFloat64(), hi.ValueFloat64()))
	}
	if step, set, known := configValue[types.Float64](attrs, "step"); set && known && step.ValueFloat64() <= 0 {
		diags.AddAttributeError(path.Root("step"), "Invalid Attribute Value", fmt.Sprintf(
			"step must be greater than 0, got %v.", step.ValueFloat64()))
	}
	if initial, set, known := configValue[types.Float64](attrs, "initial"); set && known && rangeKnown {
		if v := initial.ValueFloat64(); v < lo.ValueFloat64() || v > hi.ValueFloat64() {
			diags.AddAttributeError(path.Root("initial"), "Invalid Attribute Value", fmt.Sprintf(
				"initial must be between min (%v) and max (%v), got %v.", lo.ValueFloat64(), hi.ValueFloat64(), v))
		}
	}
	return diags
}
