package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// NewInputBoolean returns the `homeassistant_input_boolean` resource.
func NewInputBoolean() resource.Resource {
	return newHelper(helperSpec{
		domain:      "input_boolean",
		description: "A toggle helper (`input_boolean`): an on/off entity that automations can read and switch.",
		fields: map[string]schema.Attribute{
			"initial": schema.BoolAttribute{
				Description: "State the helper starts with when Home Assistant starts. Unset, it restores " +
					"the state it had before the restart. The provider never manages the current state.",
				Optional: true,
			},
		},
	})
}
