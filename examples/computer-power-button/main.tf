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

variable "notify_action" {
  description = "Action that sends a message, e.g. notify.mobile_app_<phone>."
  type        = string
  default     = "notify.notify"
}

locals {
  # Computers behind a smart plug that measures power. Each plug is switch.<plug>,
  # and its power sensor is sensor.<plug>_energy_power.
  computers = {
    desktop = { label = "Desktop", plug = "desktop_plug" }
    server  = { label = "Home server", plug = "server_plug" }
  }
}

# A dashboard button per computer.
resource "homeassistant_input_button" "turn_on" {
  for_each = local.computers

  id   = "turn_on_${each.key}"
  name = "Turn on ${each.value.label}"
  icon = "mdi:power"
}

# One script for all computers; the automations pass the computer in as a variable.
# mode: parallel lets it run for several computers at once.
resource "homeassistant_script" "power_cycle_computer" {
  id = "power_cycle_computer"
  config = yamldecode(templatefile("${path.module}/power_cycle_computer.yaml.tftpl", {
    # plug => label, for the message at the end
    labels        = { for k, c in local.computers : c.plug => c.label }
    notify_action = var.notify_action
  }))
}

# Pressing a computer's button starts the script for that computer.
resource "homeassistant_automation" "turn_on" {
  for_each = local.computers

  id = "turn_on_${each.key}"
  config = {
    alias       = "Turn on ${each.value.label}"
    description = "Starts the power-cycle script for this computer"
    triggers    = [provider::homeassistant::state_trigger(homeassistant_input_button.turn_on[each.key].entity_id)]
    conditions  = []
    actions = [provider::homeassistant::action("script.turn_on", {
      target = { entity_id = homeassistant_script.power_cycle_computer.entity_id }
      data   = { variables = { computer = each.value.plug } }
    })]
    mode = "single"
  }
}
