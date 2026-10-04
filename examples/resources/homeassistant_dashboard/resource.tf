# A section defined once and reused in two dashboards: edit the local, both dashboards change.
locals {
  quick_actions = {
    type = "grid"
    cards = [
      { type = "heading", heading = "Quick actions" },
      { type = "button", entity = "script.good_morning" },
    ]
  }
}

resource "homeassistant_dashboard" "home" {
  url_path = "my-home"
  title    = "My Home"
  icon     = "mdi:home-variant"
  config = {
    views = [{
      type     = "sections"
      title    = "Overview"
      sections = [local.quick_actions]
    }]
  }
}

resource "homeassistant_dashboard" "wall_tablet" {
  url_path        = "wall-tablet"
  title           = "Wall Tablet"
  show_in_sidebar = false
  config = {
    views = [{
      type  = "sections"
      title = "Hallway"
      sections = [
        local.quick_actions,
        { type = "grid", cards = [{ type = "tile", entity = "lock.front_door" }] },
      ]
    }]
  }
}

# A section generated per area: one view per area, with its lights plus the shared quick actions.
data "homeassistant_entities" "lights" {
  for_each = toset(["kitchen", "bedroom"])
  domain   = "light"
  area     = each.key
}

locals {
  light_sections = {
    for area, d in data.homeassistant_entities.lights : area => {
      type = "grid"
      cards = concat(
        [{ type = "heading", heading = title(area) }],
        [for e in d.entity_ids : { type = "tile", entity = e }],
      )
    }
  }
}

resource "homeassistant_dashboard" "rooms" {
  url_path = "rooms-by-area"
  title    = "Rooms"
  icon     = "mdi:floor-plan"
  config = {
    views = [for area, s in local.light_sections : {
      type     = "sections"
      title    = title(area)
      sections = [s, local.quick_actions]
    }]
  }
}
