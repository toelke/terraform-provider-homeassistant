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
    weather          = "weather.home"
    weather_warning  = "sensor.weather_warning_level"
    oven_child_lock  = "switch.oven_child_lock"
    office_climate   = "climate.office"
    office_co2       = "sensor.office_co2"
    office_ceiling   = "light.office_ceiling"
    office_desk_lamp = "light.office_desk_lamp"
    office_blind     = "cover.office_blind"
    living_ceiling   = "light.living_room_ceiling"
    living_strings   = "light.living_room_string_lights"
    living_blind     = "cover.patio_door_blind"
    kitchen_climate  = "climate.kitchen"
    kitchen_ceiling  = "light.kitchen_ceiling"
    kitchen_counter  = "light.kitchen_counter"
    solar_power      = "sensor.balcony_solar_power"
  }

  people = {
    alex = { name = "Alex", person = "person.alex", travel_time = "sensor.alex_travel_time_home" }
    sam  = { name = "Sam", person = "person.sam", travel_time = "sensor.sam_travel_time_home" }
  }

  # Variables available to every dashboard and view file.
  dashboard_vars = {
    e = local.e
    # A markdown card's content per person, rendered from one template.
    person_card = { for k, p in local.people : k => templatefile("${path.module}/person-card.md.tftpl", p) }
    # A card shared by several views, defined once.
    alerts_card = yamldecode(templatefile("${path.module}/cards/alerts.yaml", { e = local.e }))
  }

  # The room views, one file each, in the order of their file name prefix.
  room_views = [
    for f in sort(fileset("${path.module}/views", "*.yaml")) :
    yamldecode(templatefile("${path.module}/views/${f}", local.dashboard_vars))
  ]

  # A navigation button per room view, made from the view's own title, icon and path.
  room_buttons = [
    for v in local.room_views : {
      type       = "button"
      name       = v.title
      icon       = v.icon
      show_name  = true
      show_icon  = true
      tap_action = { action = "navigate", navigation_path = "/dashboard-rooms/${v.path}" }
    }
  ]
}

resource "homeassistant_dashboard" "rooms" {
  url_path = "dashboard-rooms"
  title    = "Rooms"
  icon     = "mdi:floor-plan"
  config = {
    views = concat(
      [yamldecode(templatefile("${path.module}/rooms-home.yaml", { room_buttons = local.room_buttons }))],
      local.room_views,
    )
  }
}

resource "homeassistant_dashboard" "tablet" {
  url_path = "wall-tablet"
  title    = "Wall Tablet"
  icon     = "mdi:tablet-dashboard"
  config   = yamldecode(templatefile("${path.module}/tablet.yaml", local.dashboard_vars))
}
