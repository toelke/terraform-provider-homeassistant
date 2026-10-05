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

# Exactly one match: skip hidden entities, such as a stale duplicate left behind by MQTT
# discovery that shares the friendly name. There is no `disabled` filter: HA removes disabled
# entities from the states, so they never show up here.
data "homeassistant_entities" "hall_lamp" {
  area         = "Hall"
  name_pattern = "Hall Lamp"
  hidden       = false

  lifecycle {
    postcondition {
      condition     = length(self.entity_ids) == 1
      error_message = "Expected exactly one visible \"Hall Lamp\" in the hall, found ${length(self.entity_ids)}."
    }
  }
}

output "hall_lamp_id" {
  value = one(data.homeassistant_entities.hall_lamp.entity_ids)
}
