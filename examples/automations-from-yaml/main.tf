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
  # Entity IDs that the YAML files refer to as ${e.<key>}. ../entity-lookups shows how to look
  # them up instead of writing them down.
  e = {
    bathroom_presence = "binary_sensor.bathroom_presence"
    bathroom_ceiling  = "light.bathroom_ceiling"
    av_receiver       = "binary_sensor.av_receiver"
    subwoofer         = "switch.subwoofer"
    oven_child_lock   = "switch.oven_child_lock"
    tablet_battery    = "sensor.hallway_tablet_battery"
    tablet_charger    = "switch.hallway_tablet_charger"
  }

  # One automation per file in automations/, keyed by file name. Each file is the YAML that the
  # automation editor shows ("Edit in YAML"), `id:` line included, rendered with templatefile so
  # that it can say ${e.<key>} instead of an entity ID.
  automations = {
    for f in fileset("${path.module}/automations", "*.yaml") :
    trimsuffix(f, ".yaml") => yamldecode(templatefile("${path.module}/automations/${f}", { e = local.e }))
  }
}

resource "homeassistant_automation" "this" {
  for_each = local.automations

  # The `id:` line becomes the resource's id. `config` must not contain it.
  id     = each.value.id
  config = { for k, v in each.value : k => v if k != "id" }
}
