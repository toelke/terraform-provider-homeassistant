# A script with a field, run from an automation as the action `script.goodnight`.
resource "homeassistant_script" "goodnight" {
  id = "goodnight"
  config = {
    alias = "Goodnight"
    mode  = "single"
    icon  = "mdi:weather-night"
    fields = {
      brightness = {
        name     = "Night light brightness"
        default  = 10
        selector = { number = { min = 0, max = 100, unit_of_measurement = "%" } }
      }
    }
    sequence = [
      {
        action = "light.turn_off"
        target = { area_id = "living_room" }
      },
      {
        action = "light.turn_on"
        target = { area_id = "bedroom" }
        data   = { brightness_pct = "{{ brightness }}" }
      },
    ]
  }
}

resource "homeassistant_automation" "bedtime_button" {
  id = "bedtime_button"
  config = {
    alias    = "Bedtime button"
    triggers = [{ trigger = "state", entity_id = "event.bedside_button" }]
    actions = [{
      action = homeassistant_script.goodnight.entity_id
      data   = { brightness = 20 }
    }]
  }
}

# From a file copied out of the script editor ("Edit in YAML").
resource "homeassistant_script" "leaving" {
  id     = "leaving_home"
  config = yamldecode(file("${path.module}/scripts/leaving.yaml"))
}
