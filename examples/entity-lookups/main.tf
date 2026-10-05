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

# Instead of hard-coding entity IDs (which for Zigbee devices are often just the IEEE address),
# look them up by domain, area and friendly name. Everything else in a configuration then uses
# local.entity.<key>, e.g. as the `e` variable of templatefile.

locals {
  entity_lookups = merge(local.entity_lookups_named, {
    # The bulbs of the living room's string lights are named A to H; generate their lookups.
    for b in split("", "ABCDEFGH") : "string_light_${lower(b)}" => { domain = "light", area = "living_room", name = b }
  })

  # key => what to look for; `name` is a glob on the friendly name, `area` an area ID or null
  entity_lookups_named = {
    office_climate     = { domain = "climate", area = "office", name = "Office" }
    office_window      = { domain = "binary_sensor", area = "office", name = "Office Window" }
    office_temperature = { domain = "sensor", area = "office", name = "Office T/H Temperature" }
    living_room_cover  = { domain = "cover", area = "living_room", name = "Patio Door Blind" }
    solar_power        = { domain = "sensor", area = "balcony", name = "Solar Total power" }
    grid_power         = { domain = "sensor", area = "basement", name = "Electricity Meter Power" }
    weather            = { domain = "weather", area = null, name = "OpenWeatherMap" }
    person_parent_a    = { domain = "person", area = null, name = "Parent A" }
  }

  # Entities whose friendly name is shared with a hidden duplicate, which a name lookup cannot
  # tell apart: pinned by entity ID, but still checked to exist.
  entity_pinned = {
    dishwasher_state = "sensor.dishwasher_operation_state"
  }

  # Entities whose only name is their Zigbee address: give them a name, and use them by key.
  entity_named = {
    office_trv                = { entity_id = "climate.0x0000000000000001", name = "Office TRV" }
    living_room_switch_left   = { entity_id = "switch.0x0000000000000002_left", name = "Living Room Switch Left" }
    living_room_switch_center = { entity_id = "switch.0x0000000000000002_center", name = "Living Room Switch Center" }
    living_room_switch_right  = { entity_id = "switch.0x0000000000000002_right", name = "Living Room Switch Right" }
  }

  # key => entity ID, from all three sources
  entity = merge(
    { for k, d in data.homeassistant_entities.lookup : k => one(d.entity_ids) },
    { for k, d in data.homeassistant_entity.pinned : k => d.entity_id },
    { for k, r in homeassistant_entity_settings.named : k => r.entity_id },
  )
}

data "homeassistant_entities" "lookup" {
  for_each = local.entity_lookups

  domain       = each.value.domain
  area         = each.value.area
  name_pattern = each.value.name

  # Fail the plan, with a readable message, when a device was renamed or moved, instead of
  # silently writing a null entity ID into automations and dashboards.
  lifecycle {
    postcondition {
      condition     = length(self.entity_ids) == 1
      error_message = "Expected exactly one ${each.value.domain} named \"${each.value.name}\"${each.value.area == null ? "" : " in area ${each.value.area}"}, found ${jsonencode(self.entity_ids)}."
    }
  }
}

# Fails the plan if the entity does not exist.
data "homeassistant_entity" "pinned" {
  for_each = local.entity_pinned

  entity_id = each.value
}

# Only `name` is managed; icon, area and everything else stay with the UI.
resource "homeassistant_entity_settings" "named" {
  for_each = local.entity_named

  entity_id = each.value.entity_id
  name      = each.value.name
}

# Devices can be looked up by name too, e.g. for `notify.send_message` with a device target.
data "homeassistant_device" "hallway_tablet" {
  name = "Hallway Tablet"
}

# Using the lookups: all string-light bulbs off at midnight, found by their key prefix.
resource "homeassistant_automation" "string_lights_off" {
  id = "string_lights_off_at_midnight"
  config = {
    alias       = "Living room: string lights off at midnight"
    description = ""
    triggers    = [provider::homeassistant::time_trigger("00:00:00")]
    conditions  = []
    actions = [provider::homeassistant::action("light.turn_off", {
      target = { entity_id = [for k, v in local.entity : v if startswith(k, "string_light_")] }
    })]
    mode = "single"
  }
}

output "entity" {
  description = "Entity IDs by lookup key."
  value       = local.entity
}

output "hallway_tablet_device_id" {
  value = data.homeassistant_device.hallway_tablet.device_id
}
