package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// NewAutomation returns the `homeassistant_automation` resource.
func NewAutomation() resource.Resource {
	return &configItemResource{spec: configItemSpec{
		domain:   "automation",
		newItems: client.NewAutomations,
		description: "An automation, stored in `automations.yaml` like one made in the automation " +
			"editor. Whether it is enabled is runtime state and not managed; to disable it as " +
			"code, set `initial_state = false` inside `config`.",
		idDescription: "Config ID of the automation, e.g. `motion_bedroom`: lowercase letters, " +
			"digits, and underscores. Changing it replaces the automation. Creating fails if one with this ID already exists; import it instead.",
		configDescription: "The automation (`alias`, `description`, `mode`, `triggers`, " +
			"`conditions`, `actions`, …), as an HCL object or " +
			"`yamldecode(file(\"automation.yaml\"))`. YAML copied from the automation " +
			"editor works unchanged once its `id` line is removed. Home Assistant validates " +
			"it on apply. When it saves, Home Assistant renames old keys (`trigger`, " +
			"`condition`, and `action` at the top level become plurals, `service` becomes " +
			"`action`); the provider keeps your spelling, and the plan stays empty.",
		entityIDDescription: "Entity ID of the automation, e.g. `automation.bedroom_lights_on_motion`. " +
			"Home Assistant derives it from the alias when the automation is first created, " +
			"and keeps it afterwards.",
	}}
}
