package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewInputButton returns the `homeassistant_input_button` resource.
func NewInputButton() resource.Resource {
	return newHelper(helperSpec{
		domain:      "input_button",
		description: "A button helper (`input_button`): a button that automations can react to when it is pressed.",
	})
}
