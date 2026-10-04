package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// NewScene returns the `homeassistant_scene` resource.
func NewScene() resource.Resource {
	return &configItemResource{spec: configItemSpec{
		domain:   "scene",
		newItems: client.NewScenes,
		description: "A scene, stored in `scenes.yaml` like one made in the scene editor. " +
			"Automations and scripts activate it with the action `scene.turn_on`.",
		idDescription: "Config ID of the scene, e.g. `movie_night`: lowercase letters, digits, and " +
			"underscores. Changing it replaces the scene. Import with this ID.",
		configDescription: "The scene (`name`, `icon`, `entities`, …), as an HCL object or " +
			"`yamldecode(file(\"scene.yaml\"))`. `entities` maps entity IDs to the state to set, " +
			"e.g. `{ \"light.tv\" = { state = \"on\", brightness = 80 } }`. YAML copied from the " +
			"scene editor works unchanged once its `id` line is removed. Home Assistant validates " +
			"it on apply.",
		entityIDDescription: "Entity ID of the scene, e.g. `scene.movie_night`. Home Assistant " +
			"derives it from the name when the scene is first created, and keeps it afterwards.",
	}}
}
