package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// NewScript returns the `homeassistant_script` resource.
func NewScript() resource.Resource {
	return &configItemResource{spec: configItemSpec{
		domain:   "script",
		newItems: client.NewScripts,
		description: "A script, stored in `scripts.yaml` like one made in the script editor. " +
			"Other automations and scripts run it with the action `script.<id>`.",
		idDescription: "Config ID of the script, e.g. `goodnight`: lowercase letters, digits, and " +
			"underscores. It is also the object ID of the script's entity. Changing it replaces " +
			"the script. Import with this ID.",
		configDescription: "The script (`alias`, `description`, `mode`, `icon`, `fields`, " +
			"`sequence`, …), as an HCL object or `yamldecode(file(\"script.yaml\"))`. YAML " +
			"copied from the script editor works unchanged. Home Assistant validates it on apply. " +
			"When it saves, Home Assistant renames `service` to `action` in the sequence; the " +
			"provider keeps your spelling, and the plan stays empty.",
		entityIDDescription: "Entity ID of the script, `script.<id>`.",
	}}
}
