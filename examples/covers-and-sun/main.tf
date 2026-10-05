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

# These automations are written in HCL instead of YAML, with the provider's builder functions.
# Each returns a plain object, the same as writing it out (see the comments), so they can be
# mixed freely with literal objects for the trigger and condition types that have no builder.

locals {
  days = {
    weekdays = ["mon", "tue", "wed", "thu", "fri"]
    weekend  = ["sat", "sun"]
  }

  e = {
    sun_elevation = "sensor.sun_solar_elevation"
    sun_azimuth   = "sensor.sun_solar_azimuth"
    office_blind  = "cover.office_blind"
    patio_blind   = "cover.patio_door_blind"
    awning        = "cover.terrace_awning"
    av_receiver   = "binary_sensor.av_receiver"
    hall_ceiling  = "light.hallway_ceiling"
  }

  ground_floor_areas = ["living_room", "kitchen", "hallway_ground"]

  # The office faces east: on workdays, the morning sun shines on the screens.
  office_sun_azimuth_limit = 104
}

# Morning sun on the office screens: lower the blind to 40 %.
resource "homeassistant_automation" "office_blind_down" {
  id = "office_blind_sun_down"
  config = {
    alias       = "Office: lower blind for the morning sun"
    description = ""
    # { trigger = "numeric_state", entity_id = "sensor.sun_solar_elevation", above = 4.3 }
    triggers = [provider::homeassistant::numeric_state_trigger(local.e.sun_elevation, { above = 4.3 })]
    conditions = [
      # { condition = "time", weekday = [...] }
      provider::homeassistant::time_condition({ weekday = local.days.weekdays }),
      { condition = "numeric_state", entity_id = local.e.sun_azimuth, below = local.office_sun_azimuth_limit },
    ]
    actions = [provider::homeassistant::action("cover.set_cover_position", {
      target = { entity_id = local.e.office_blind }
      data   = { position = 40 }
    })]
    mode = "single"
  }
}

# Once the sun has moved on, open the blind again.
resource "homeassistant_automation" "office_blind_up" {
  id = "office_blind_sun_up"
  config = {
    alias       = "Office: raise blind after the morning sun"
    description = ""
    triggers    = [provider::homeassistant::numeric_state_trigger(local.e.sun_azimuth, { above = local.office_sun_azimuth_limit })]
    conditions = [
      provider::homeassistant::time_condition({ weekday = local.days.weekdays }),
      { condition = "numeric_state", entity_id = local.e.sun_elevation, above = 4.3 },
    ]
    actions = [provider::homeassistant::action("cover.open_cover", {
      target = { entity_id = local.e.office_blind }
    })]
    mode = "restart"
  }
}

# Retract the awning an hour before sunset, so it is not left out overnight.
resource "homeassistant_automation" "awning_in" {
  id = "awning_in_before_sunset"
  config = {
    alias       = "Terrace: retract awning before sunset"
    description = ""
    triggers    = [{ trigger = "sun", event = "sunset", offset = "-01:00:00" }]
    conditions  = []
    actions     = [provider::homeassistant::action("cover.open_cover", { target = { entity_id = local.e.awning } })]
    mode        = "single"
  }
}

# The first light on the ground floor in the morning means someone is up: open the patio blind.
resource "homeassistant_automation" "good_morning" {
  id = "good_morning_patio_blind"
  config = {
    alias       = "Ground floor: first light in the morning opens the patio blind"
    description = ""
    triggers = [provider::homeassistant::state_trigger(
      sort(flatten([for d in data.homeassistant_entities.ground_floor_lights : d.entity_ids])),
      { from = "off", to = "on" },
    )]
    conditions = [{
      condition = "or"
      conditions = [
        provider::homeassistant::time_condition({ after = "06:30:00", before = "07:30:00", weekday = local.days.weekdays }),
        provider::homeassistant::time_condition({ after = "06:30:00", before = "08:30:00", weekday = local.days.weekend }),
      ]
    }]
    actions = [provider::homeassistant::action("cover.open_cover", { target = { entity_id = local.e.patio_blind } })]
    # single: the first light wins, the others are ignored while the blind opens
    mode = "single"
  }
}

# The lights of every ground-floor area, looked up instead of listed.
data "homeassistant_entities" "ground_floor_lights" {
  for_each = toset(local.ground_floor_areas)

  domain = "light"
  area   = each.value
}

# Late in the evening, the receiver going off for two minutes means the TV evening is over:
# light the hallway on the way to bed.
resource "homeassistant_automation" "tv_off_hall_light" {
  id = "tv_off_hall_light"
  config = {
    alias       = "Living room: receiver off in the evening lights the hallway"
    description = ""
    triggers = [provider::homeassistant::state_trigger(local.e.av_receiver, {
      from  = "on"
      to    = "off"
      "for" = { minutes = 2 } # quoted: HCL would read `{ for` as a for expression
    })]
    conditions = [provider::homeassistant::time_condition({ after = "21:00:00" })]
    actions    = [provider::homeassistant::action("light.turn_on", { target = { entity_id = local.e.hall_ceiling } })]
    mode       = "single"
  }
}
