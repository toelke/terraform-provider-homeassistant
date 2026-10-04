# Actions defined once and reused in several automations.
locals {
  night_light = {
    action = "light.turn_on"
    target = { area_id = "bedroom" }
    data   = { brightness_pct = 20 }
  }
}

resource "homeassistant_automation" "motion_bedroom" {
  id = "motion_bedroom"
  config = {
    alias = "Bedroom — lights on motion"
    mode  = "restart"
    triggers = [{
      trigger   = "state"
      entity_id = "binary_sensor.bedroom_motion"
      to        = "on"
    }]
    conditions = [{
      condition = "time"
      after     = "22:00:00"
      before    = "08:00:00"
    }]
    actions = [local.night_light]
  }
}

resource "homeassistant_automation" "bedroom_button" {
  id = "bedroom_button"
  config = {
    alias    = "Bedroom — night light button"
    triggers = [{ trigger = "state", entity_id = "event.bedroom_button" }]
    actions  = [local.night_light]
  }
}

# From a file copied out of the automation editor ("Edit in YAML"), without its `id:` line.
resource "homeassistant_automation" "away" {
  id     = "away_mode"
  config = yamldecode(file("${path.module}/automations/away.yaml"))
}

# Disabled as code: `initial_state = false` keeps it off after every restart and reload.
# Turning an automation on or off in the UI is runtime state, which the provider ignores.
resource "homeassistant_automation" "vacation_lights" {
  id = "vacation_lights"
  config = {
    alias         = "Vacation — random lights"
    initial_state = false
    triggers      = [{ trigger = "time_pattern", minutes = "/30" }]
    actions = [{
      action = "light.toggle"
      target = { entity_id = "light.living_room" }
    }]
  }
}
