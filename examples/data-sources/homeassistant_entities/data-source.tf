# Every light in the bedroom. `area` takes an area ID or name.
data "homeassistant_entities" "bedroom_lights" {
  domain = "light"
  area   = "Bedroom"
}

output "bedroom_light_ids" {
  value = data.homeassistant_entities.bedroom_lights.entity_ids
}

# Temperature sensors whose name starts with "Outdoor".
data "homeassistant_entities" "outdoor_temperatures" {
  device_class = "temperature"
  name_pattern = "Outdoor*"
}

output "outdoor_temperature_names" {
  value = { for id, e in data.homeassistant_entities.outdoor_temperatures.entities : id => e.friendly_name }
}
