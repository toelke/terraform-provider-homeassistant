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
  e_base = {
    string_lights      = "light.living_room_string_lights"
    string_light_a     = "light.string_light_a"
    string_light_b     = "light.string_light_b"
    string_light_c     = "light.string_light_c"
    piano_lamp         = "light.piano_lamp"
    christmas_tree     = "light.christmas_tree"
    child_room_ceiling = "light.child_room_ceiling"
  }

  # Zigbee2MQTT publishes button presses of these remotes on zigbee2mqtt/<IEEE address>/action.
  remotes = {
    child_room_door = "zigbee2mqtt/0x0000000000000010/action" # Aqara wireless double switch
    living_room     = "zigbee2mqtt/0x0000000000000011/action" # IKEA Styrbar
  }

  # One scene per file in scenes/, keyed by file name, with its `id:` line as the resource id.
  # They see only e_base: a scene cannot refer to itself.
  scenes = {
    for f in fileset("${path.module}/scenes", "*.yaml") :
    trimsuffix(f, ".yaml") => yamldecode(templatefile("${path.module}/scenes/${f}", { e = local.e_base }))
  }

  # What the automations see: the base entities plus the scenes and the helper created here.
  e = merge(local.e_base, {
    scene_tv              = homeassistant_scene.this["tv"].entity_id
    scene_bright          = homeassistant_scene.this["bright"].entity_id
    scene_christmas       = homeassistant_scene.this["christmas"].entity_id
    child_room_light_lock = homeassistant_input_boolean.child_room_light_lock.entity_id
  })

  automations = {
    for f in fileset("${path.module}/automations", "*.yaml") :
    trimsuffix(f, ".yaml") => yamldecode(templatefile("${path.module}/automations/${f}", { e = local.e, remotes = local.remotes }))
  }
}

resource "homeassistant_scene" "this" {
  for_each = local.scenes

  id     = each.value.id
  config = { for k, v in each.value : k => v if k != "id" }
}

# Set by the parents' switch outside the child's room: while it is on, the ceiling light there
# turns itself off again.
resource "homeassistant_input_boolean" "child_room_light_lock" {
  id   = "child_room_light_lock"
  name = "Child's room light lock"
  icon = "mdi:lightbulb-off"
}

resource "homeassistant_automation" "this" {
  for_each = local.automations

  id     = each.value.id
  config = { for k, v in each.value : k => v if k != "id" }
}
