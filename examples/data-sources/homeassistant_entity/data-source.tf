data "homeassistant_entity" "sun" {
  entity_id = "sun.sun"
}

output "sun_state" {
  value = data.homeassistant_entity.sun.state
}

# Attributes keep their JSON types, so numbers can be compared directly.
output "sun_is_high" {
  value = data.homeassistant_entity.sun.attributes.elevation > 30
}
