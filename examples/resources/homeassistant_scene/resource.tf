# A scene, activated by an automation.
resource "homeassistant_scene" "movie_night" {
  id = "movie_night"
  config = {
    name = "Movie night"
    icon = "mdi:movie-open"
    entities = {
      "light.living_room_ceiling" = { state = "off" }
      "light.tv_backlight"        = { state = "on", brightness = 60, color_temp_kelvin = 2700 }
      "media_player.tv"           = { state = "on", source = "HDMI 1" }
    }
  }
}

resource "homeassistant_automation" "movie_night_button" {
  id = "movie_night_button"
  config = {
    alias    = "Movie night button"
    triggers = [{ trigger = "state", entity_id = "event.sofa_button" }]
    actions = [{
      action = "scene.turn_on"
      target = { entity_id = homeassistant_scene.movie_night.entity_id }
    }]
  }
}

# From a file copied out of the scene editor ("Edit in YAML"), with its `id` line removed.
resource "homeassistant_scene" "evening" {
  id     = "evening"
  config = yamldecode(file("${path.module}/scenes/evening.yaml"))
}
