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
  description = "Action that sends a message to the household, e.g. a rest_command for a messenger group."
  type        = string
  default     = "notify.notify"
}

locals {
  # The people of the household. `phone` is the mobile app's notify service, `media` the phone's
  # media session sensor, both from the Home Assistant companion app.
  people = {
    alex = {
      name    = "Alex"
      person  = "person.alex"
      tracker = "device_tracker.alex_phone"
      phone   = "notify.mobile_app_alex_phone"
      media   = "sensor.alex_phone_media_session"
    }
    sam = {
      name    = "Sam"
      person  = "person.sam"
      tracker = "device_tracker.sam_phone"
      phone   = "notify.mobile_app_sam_phone"
      media   = "sensor.sam_phone_media_session"
    }
  }

  e = {
    front_door      = "binary_sensor.front_door_lock"
    uptime          = "sensor.uptime"
    porch_spotlight = "light.porch_spotlight"
  }
}

# The wall tablet in the hallway, which shows notifications sent to its device.
data "homeassistant_device" "hallway_tablet" {
  name = "Hallway Tablet"
}

resource "homeassistant_script" "porch_spotlight" {
  id = "porch_spotlight_2_minutes"
  config = {
    alias = "Porch spotlight for 2 minutes"
    icon  = "mdi:light-flood-down"
    mode  = "restart"
    sequence = [
      provider::homeassistant::action("light.turn_on", { target = { entity_id = local.e.porch_spotlight } }),
      provider::homeassistant::delay({ minutes = 2 }),
      provider::homeassistant::action("light.turn_off", { target = { entity_id = local.e.porch_spotlight } }),
    ]
  }
}

# "<name> is home": a message to the household, and light on the way in. One per person.
resource "homeassistant_automation" "person_home" {
  for_each = local.people

  id = "person_home_${each.key}"
  config = yamldecode(templatefile("${path.module}/templates/person-home.yaml.tftpl", {
    name          = each.value.name
    person        = each.value.person
    notify_action = var.notify_action
    spotlight     = homeassistant_script.porch_spotlight.entity_id
  }))
}

# Lets a phone ring at full volume, even when muted. One script per person, run from a dashboard.
resource "homeassistant_script" "find_phone" {
  for_each = local.people

  id = "find_phone_${each.key}"
  config = {
    alias = "Find ${each.value.name}'s phone"
    icon  = "mdi:cellphone-sound"
    sequence = [provider::homeassistant::action(each.value.phone, {
      data = {
        message = "TTS"
        data    = { ttl = 0, priority = "high", media_stream = "alarm_stream_max", tts_text = "Here I am" }
      }
    })]
  }
}

# Sam's arrival, announced to Alex: by voice on Alex's phone if Alex is home listening to
# something (pausing it first), otherwise on the hallway tablet if anyone is home.
resource "homeassistant_automation" "announce_arrival" {
  id = "announce_sam_home"
  config = yamldecode(templatefile("${path.module}/templates/announce-arrival.yaml.tftpl", {
    arriving  = local.people.sam
    listener  = local.people.alex
    people    = [for p in local.people : p.person]
    tablet_id = data.homeassistant_device.hallway_tablet.device_id
  }))
}

# Tells the household whenever the front door is locked or unlocked. The uptime condition skips
# the state changes that every restart of Home Assistant produces.
resource "homeassistant_automation" "front_door_lock" {
  id = "front_door_lock"
  config = {
    alias       = "Front door: locked or unlocked"
    description = ""
    triggers    = [provider::homeassistant::state_trigger(local.e.front_door)]
    conditions = [{
      condition      = "template"
      value_template = "{{ as_timestamp(now()) - as_timestamp(states('${local.e.uptime}')) >= 120 }}"
    }]
    actions = [provider::homeassistant::action(var.notify_action, {
      data = {
        message = "The front door is now {{ 'unlocked' if is_state('${local.e.front_door}', 'on') else 'locked' }}."
      }
    })]
    mode = "single"
  }
}
