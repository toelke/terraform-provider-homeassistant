resource "homeassistant_area" "bedroom" {
  name = "Bedroom"
}

resource "homeassistant_label" "managed" {
  name = "Managed"
}

# Only these attributes are managed. The icon, for example, can still be changed in the UI.
resource "homeassistant_entity_settings" "ceiling" {
  entity_id = "light.bedroom_ceiling"
  name      = "Ceiling Light"
  area_id   = homeassistant_area.bedroom.id
  labels    = [homeassistant_label.managed.id]
}

# Hide a sensor nobody needs to see. Give a slow integration more time to create it.
resource "homeassistant_entity_settings" "signal" {
  entity_id = "sensor.bedroom_ceiling_signal_strength"
  hidden    = true

  timeouts = {
    create = "5m"
  }
}
