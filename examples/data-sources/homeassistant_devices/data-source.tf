# Every Hue device in the living room.
data "homeassistant_devices" "living_room_hue" {
  integration = "hue"
  area_id     = "living_room"
}

output "living_room_hue_devices" {
  value = { for d in data.homeassistant_devices.living_room_hue.devices : d.device_id => coalesce(d.name_by_user, d.name) }
}
