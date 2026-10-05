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
  description = "Action that sends a message to the household, e.g. notify.mobile_app_<phone> or a rest_command for a messenger."
  type        = string
  default     = "notify.notify"
}

variable "laundry_days" {
  description = "Weekdays on which the washing machine suggestion is sent."
  type        = list(string)
  default     = ["wed", "thu"]
}

# "Already done today" flags, so each suggestion or start happens at most once a day.
resource "homeassistant_input_boolean" "washer_suggested" {
  id   = "solar_washer_suggested"
  name = "Solar washing machine suggested"
  icon = "mdi:washing-machine"
}

resource "homeassistant_input_boolean" "dishwasher_started" {
  id   = "dishwasher_started_today"
  name = "Dishwasher started today"
  icon = "mdi:dishwasher"
}

locals {
  e = {
    # power of the balcony solar inverter, in W
    solar_power = "sensor.balcony_solar_power"
    # power at the electricity meter, in W; negative while exporting to the grid
    grid_power           = "sensor.electricity_meter_power"
    dishwasher_program   = "select.dishwasher_active_program"
    dishwasher_remote    = "binary_sensor.dishwasher_remote_start_allowed"
    dishwasher_state     = "sensor.dishwasher_operation_state"
    coffee_machine_power = "sensor.delonghi_power"
    kitchen_ceiling      = "light.kitchen_ceiling"
    household_todo       = "todo.household"
    washer_suggested     = homeassistant_input_boolean.washer_suggested.entity_id
    dishwasher_started   = homeassistant_input_boolean.dishwasher_started.entity_id
  }

  vars = {
    e             = local.e
    notify_action = var.notify_action
    laundry_days  = var.laundry_days
  }

  automations = {
    for f in fileset("${path.module}/automations", "*.yaml") :
    trimsuffix(f, ".yaml") => yamldecode(templatefile("${path.module}/automations/${f}", local.vars))
  }
}

resource "homeassistant_automation" "this" {
  for_each = local.automations

  id     = each.value.id
  config = { for k, v in each.value : k => v if k != "id" }
}
