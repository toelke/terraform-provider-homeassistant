# Look an area up by name, e.g. to put a device or an entity in it.
data "homeassistant_area" "kitchen" {
  name = "Kitchen"
}

output "kitchen_floor" {
  value = data.homeassistant_area.kitchen.floor_id
}
