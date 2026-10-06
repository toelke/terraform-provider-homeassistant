package resources

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewInputSelect returns the `homeassistant_input_select` resource.
func NewInputSelect() resource.Resource {
	return newHelper(helperSpec{
		domain:      "input_select",
		description: "A dropdown helper (`input_select`): one value out of a list of options.",
		fields: map[string]schema.Attribute{
			"options": schema.ListAttribute{
				Description: "The options, in the order the UI shows them. Must be non-empty and unique.",
				ElementType: types.StringType,
				Required:    true,
			},
			"initial": schema.StringAttribute{
				Description: "Option the helper starts with when Home Assistant starts; one of `options`. Unset, " +
					"it restores the option it had before the restart. The provider never manages the current " +
					"option.",
				Optional: true,
			},
		},
		validate: validateInputSelect,
	})
}

// validateInputSelect checks that `options` is non-empty and unique, and that `initial` is one
// of them.
func validateInputSelect(attrs map[string]attr.Value) diag.Diagnostics {
	var diags diag.Diagnostics
	options, set, known := configValue[types.List](attrs, "options")
	if !set || !known {
		return diags
	}
	if len(options.Elements()) == 0 {
		diags.AddAttributeError(path.Root("options"), "Invalid Attribute Value", "options must not be empty.")
		return diags
	}
	seen := map[string]bool{}
	allKnown := true
	for i, e := range options.Elements() {
		switch {
		case e.IsUnknown():
			allKnown = false
		case e.IsNull():
			diags.AddAttributeError(path.Root("options").AtListIndex(i), "Invalid Attribute Value",
				"options must not contain null.")
		default:
			s := e.(types.String).ValueString()
			if seen[s] {
				diags.AddAttributeError(path.Root("options").AtListIndex(i), "Invalid Attribute Value",
					fmt.Sprintf("options must be unique, but %q appears more than once.", s))
			}
			seen[s] = true
		}
	}
	if initial, set, known := configValue[types.String](attrs, "initial"); set && known && allKnown && !seen[initial.ValueString()] {
		diags.AddAttributeError(path.Root("initial"), "Invalid Attribute Value",
			fmt.Sprintf("initial must be one of options, got %q.", initial.ValueString()))
	}
	return diags
}
