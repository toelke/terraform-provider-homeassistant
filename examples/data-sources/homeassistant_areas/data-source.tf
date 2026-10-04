data "homeassistant_areas" "all" {}

# Area IDs by name.
output "area_ids" {
  value = { for a in data.homeassistant_areas.all.areas : a.name => a.area_id }
}
