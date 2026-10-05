terraform {
  required_providers {
    homeassistant = {
      source = "toelke/homeassistant"
    }
  }
}

# Leave both unset to use HOMEASSISTANT_URL and HOMEASSISTANT_TOKEN instead.
variable "ha_url" {
  type    = string
  default = null
}

variable "ha_token" {
  type      = string
  sensitive = true
  default   = null
}

provider "homeassistant" {
  url   = var.ha_url   # e.g. http://homeassistant.local:8123
  token = var.ha_token # long-lived access token of an admin user
}

locals {
  # Entity IDs; ../entity-lookups shows how to look them up instead.
  e = {
    office_climate       = "climate.office"
    office_trv           = "climate.office_trv"
    kitchen_climate      = "climate.kitchen"
    kitchen_trv          = "climate.kitchen_trv"
    office_temperature   = "sensor.office_temperature"
    office_window        = "binary_sensor.office_window"
    child_temperature    = "sensor.child_room_temperature"
    child_window         = "binary_sensor.child_room_window"
    bedroom_temperature  = "sensor.bedroom_temperature"
    bedroom_window       = "binary_sensor.bedroom_window"
    living_room_light_w  = "light.living_room_west"
    living_room_light_m  = "light.living_room_middle"
    living_room_light_e  = "light.living_room_east"
    living_room_switch_l = "switch.0x0000000000000002_left"
    living_room_switch_c = "switch.0x0000000000000002_center"
    living_room_switch_r = "switch.0x0000000000000002_right"
    hobby_room_light_w   = "light.hobby_room_west"
    hobby_room_light_m   = "light.hobby_room_middle"
    hobby_room_light_e   = "light.hobby_room_east"
    hobby_room_switch_l  = "switch.0x0000000000000003_left"
    hobby_room_switch_c  = "switch.0x0000000000000003_center"
    hobby_room_switch_r  = "switch.0x0000000000000003_right"
  }

  # Smart bulbs behind a wall switch with one relay channel per bulb, which stays powered (in
  # "decoupled" mode). Each pair is a bulb and the relay channel that represents it.
  wall_switches = {
    living_room = {
      room = "Living room"
      pairs = [
        { light = local.e.living_room_light_w, switch = local.e.living_room_switch_l },
        { light = local.e.living_room_light_m, switch = local.e.living_room_switch_c },
        { light = local.e.living_room_light_e, switch = local.e.living_room_switch_r },
      ]
    }
    hobby_room = {
      room = "Hobby room"
      pairs = [
        { light = local.e.hobby_room_light_w, switch = local.e.hobby_room_switch_l },
        { light = local.e.hobby_room_light_m, switch = local.e.hobby_room_switch_c },
        { light = local.e.hobby_room_light_e, switch = local.e.hobby_room_switch_r },
      ]
    }
  }

  # template name => instance => template variables. Each instance becomes one automation from
  # templates/<template>.yaml.tftpl.
  templated_automations = {
    # keep the set-point of a room's climate entity and of its radiator thermostat in sync
    climate-sync = {
      office  = { room = "Office", climate = local.e.office_climate, trv = local.e.office_trv }
      kitchen = { room = "Kitchen", climate = local.e.kitchen_climate, trv = local.e.kitchen_trv }
    }
    # the wall switch follows the bulbs, and the bulbs follow the wall switch
    light-to-switch = local.wall_switches
    switch-to-light = local.wall_switches
    # a blueprint instance per room; blueprint inputs other than the sensors go in `inputs`
    cold-window = {
      office = { room = "Office", temperature = local.e.office_temperature, window = local.e.office_window, inputs = {} }
      child_room = {
        room        = "Child's room"
        temperature = local.e.child_temperature
        window      = local.e.child_window
        inputs      = { temperature_threshold = 17, alert_for = { hours = 0, minutes = 2, seconds = 0 } }
      }
      bedroom = { room = "Bedroom", temperature = local.e.bedroom_temperature, window = local.e.bedroom_window, inputs = {} }
    }
  }

  # "<template>_<instance>" => automation config; "-" in template names becomes "_" for the ID
  automations = merge([
    for t, instances in local.templated_automations : {
      for k, v in instances :
      "${replace(t, "-", "_")}_${k}" => yamldecode(templatefile("${path.module}/templates/${t}.yaml.tftpl", v))
    }
  ]...)
}

resource "homeassistant_automation" "templated" {
  for_each = local.automations

  id     = each.key
  config = each.value
}
