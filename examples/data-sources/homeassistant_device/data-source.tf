# Look a device up by the name shown in the UI. A name the user gave wins over the
# integration's name.
data "homeassistant_device" "desk_lamp" {
  name = "Desk Lamp"
}

output "desk_lamp_model" {
  value = data.homeassistant_device.desk_lamp.model
}
