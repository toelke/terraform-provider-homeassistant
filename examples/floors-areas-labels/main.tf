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

# --- Floors and areas -------------------------------------------------------------------------

locals {
  # floor ID => settings. Outdoors is a "floor" without a level.
  floors = {
    attic    = { name = "Attic", level = 2 }
    upper    = { name = "Upper Floor", level = 1 }
    ground   = { name = "Ground Floor", level = 0 }
    basement = { name = "Basement", level = -1 }
    outdoors = { name = "Outdoors", level = null }
  }

  # area ID => settings; floor_id is a key of local.floors, or null for areas that span floors
  areas = {
    attic          = { name = "Attic", floor_id = "attic" }
    bedroom        = { name = "Bedroom", floor_id = "upper" }
    child_room     = { name = "Child's Room", floor_id = "upper" }
    nursery        = { name = "Nursery", floor_id = "upper" }
    office         = { name = "Office", floor_id = "upper" }
    hallway_upper  = { name = "Upper Hallway", floor_id = "upper" }
    hallway_ground = { name = "Ground Floor Hallway", floor_id = "ground" }
    kitchen        = { name = "Kitchen", floor_id = "ground" }
    living_room    = { name = "Living Room", floor_id = "ground" }
    vestibule      = { name = "Vestibule", floor_id = "ground" }
    laundry_room   = { name = "Laundry Room", floor_id = "basement" }
    server_room    = { name = "Server Room", floor_id = "basement" }
    balcony        = { name = "Balcony", floor_id = "outdoors" }
    terrace        = { name = "Terrace", floor_id = "outdoors" }
    stairwell      = { name = "Stairwell", floor_id = null }
  }
}

resource "homeassistant_floor" "this" {
  for_each = local.floors

  id    = each.key
  name  = each.value.name
  level = each.value.level
}

resource "homeassistant_area" "this" {
  for_each = local.areas

  # Setting `id` keeps the area ID stable and readable, whatever the display name.
  id       = each.key
  name     = each.value.name
  floor_id = each.value.floor_id == null ? null : homeassistant_floor.this[each.value.floor_id].id
}

# Areas and floors can be targeted directly in automations: "everything on the upper floor off".
resource "homeassistant_script" "upper_floor_off" {
  id = "upper_floor_off"
  config = {
    alias = "Upper floor: all lights off"
    icon  = "mdi:home-floor-1"
    sequence = [provider::homeassistant::action("light.turn_off", {
      target = { floor_id = homeassistant_floor.this["upper"].id }
    })]
  }
}

# --- Labels -----------------------------------------------------------------------------------

# A label that marks entities the wall-tablet dashboard should not list one by one.
resource "homeassistant_label" "hide_on_tablet" {
  id          = "hide_on_tablet"
  name        = "Hide on tablet"
  icon        = "mdi:tablet-dashboard"
  description = "Entities that the wall tablet does not list one by one"
}

# The single bulbs of the living room's string lights are named A, B, C, …: a one-character glob
# finds them all.
data "homeassistant_entities" "string_light_bulbs" {
  domain       = "light"
  area         = homeassistant_area.this["living_room"].id
  name_pattern = "?"
}

# Label each bulb. `labels` is the complete list of labels on the entity.
resource "homeassistant_entity_settings" "hide_on_tablet" {
  for_each = toset(data.homeassistant_entities.string_light_bulbs.entity_ids)

  entity_id = each.value
  labels    = [homeassistant_label.hide_on_tablet.id]
}

# Labels work as a filter in data sources too: all living-room lights except the labelled ones.
data "homeassistant_entities" "living_room_lights" {
  domain = "light"
  area   = homeassistant_area.this["living_room"].id
}

data "homeassistant_entities" "hidden_on_tablet" {
  label = homeassistant_label.hide_on_tablet.id

  depends_on = [homeassistant_entity_settings.hide_on_tablet]
}

resource "homeassistant_dashboard" "tablet" {
  url_path = "wall-tablet"
  title    = "Wall Tablet"
  icon     = "mdi:tablet-dashboard"
  config = {
    views = [{
      type  = "sections"
      title = "Living Room"
      sections = [provider::homeassistant::grid_section(
        [
          for e in sort(setsubtract(data.homeassistant_entities.living_room_lights.entity_ids, data.homeassistant_entities.hidden_on_tablet.entity_ids)) :
          provider::homeassistant::tile_card(e)
        ],
        { heading = "Lights" },
      )]
    }]
  }
}
