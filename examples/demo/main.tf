terraform {
  required_providers {
    homeassistant = {
      source  = "toelke/homeassistant"
      version = "= 0.1.0"
    }
  }
}

# Reads HOMEASSISTANT_URL and HOMEASSISTANT_TOKEN, as printed by ./up.sh.
provider "homeassistant" {}

# --- Areas ------------------------------------------------------------------------------------

locals {
  areas = {
    living_room = "Living Room"
    kitchen     = "Kitchen"
    bedroom     = "Bedroom"
    office      = "Office"
    hallway     = "Hallway"
  }

  # Which area each of the demo integration's lights is in.
  light_areas = {
    "light.ceiling_lights"              = "living_room"
    "light.living_room_rgbww_lights"    = "living_room"
    "light.kitchen_lights"              = "kitchen"
    "light.bed_light"                   = "bedroom"
    "light.office_rgbw_lights"          = "office"
    "light.entrance_color_white_lights" = "hallway"
  }
}

resource "homeassistant_floor" "ground" {
  name  = "Ground Floor"
  level = 0
}

# Onboarding creates the first three areas; adopt them instead of creating duplicates.
import {
  for_each = toset(["living_room", "kitchen", "bedroom"])
  to       = homeassistant_area.room[each.key]
  id       = each.key
}

resource "homeassistant_area" "room" {
  for_each = local.areas
  id       = each.key
  name     = each.value
  floor_id = homeassistant_floor.ground.id
}

# Put each light into its area. Only the area is managed; the UI still owns name and icon.
resource "homeassistant_entity_settings" "light" {
  for_each  = local.light_areas
  entity_id = each.key
  area_id   = homeassistant_area.room[each.value].id
}

# --- Automations, scripts, scenes -------------------------------------------------------------

# Every file in automations/ is an automation, as copied from the UI's YAML editor. The file
# name is its ID.
resource "homeassistant_automation" "yaml" {
  for_each = {
    for f in fileset(path.module, "automations/*.yaml") : trimsuffix(basename(f), ".yaml") => f
  }
  id     = each.key
  config = yamldecode(file("${path.module}/${each.value}"))

  depends_on = [homeassistant_script.all_lights_off]
}

resource "homeassistant_script" "all_lights_off" {
  id = "all_lights_off"
  config = {
    alias    = "All lights off"
    icon     = "mdi:lightbulb-group-off"
    sequence = [provider::homeassistant::action("light.turn_off", { target = { entity_id = "all" } })]
  }
}

resource "homeassistant_scene" "movie_night" {
  id = "movie_night"
  config = {
    name = "Movie night"
    icon = "mdi:movie-open"
    entities = {
      "light.ceiling_lights"           = { state = "off" }
      "light.living_room_rgbww_lights" = { state = "on", brightness = 60 }
    }
  }
}

# --- Dashboards -------------------------------------------------------------------------------

# Every light in each area, as Home Assistant sees it after the assignments above.
data "homeassistant_entities" "lights" {
  for_each = homeassistant_area.room
  domain   = "light"
  area     = each.value.id

  depends_on = [homeassistant_entity_settings.light]
}

locals {
  # Defined once, used on both dashboards.
  quick_actions = provider::homeassistant::grid_section([
    { type = "button", entity = homeassistant_script.all_lights_off.entity_id, name = "All off" },
    { type = "button", entity = homeassistant_scene.movie_night.entity_id, name = "Movie night" },
    provider::homeassistant::tile_card("lock.front_door"),
  ], { heading = "Quick actions" })

  # One section per area, with a tile for every light in it.
  light_sections = {
    for area, d in data.homeassistant_entities.lights : area => provider::homeassistant::grid_section(
      [for e in d.entity_ids : provider::homeassistant::tile_card(e, {
        features = [{ type = "light-brightness" }]
      })],
      { heading = "Lights" },
    )
  }
}

resource "homeassistant_dashboard" "home" {
  url_path = "dashboard-home"
  title    = "Home"
  icon     = "mdi:home-variant"
  config = {
    views = [for area, name in local.areas : {
      type     = "sections"
      title    = name
      path     = replace(area, "_", "-")
      sections = [local.light_sections[area], local.quick_actions]
    }]
  }
}

resource "homeassistant_dashboard" "wall_tablet" {
  url_path = "wall-tablet"
  title    = "Wall Tablet"
  icon     = "mdi:tablet"
  config = {
    views = [{
      type  = "sections"
      title = "Hallway"
      sections = [
        local.quick_actions,
        provider::homeassistant::grid_section([
          provider::homeassistant::tile_card("lock.kitchen_door"),
          provider::homeassistant::tile_card("cover.garage_door"),
        ], { heading = "Doors" }),
      ]
    }]
  }
}
