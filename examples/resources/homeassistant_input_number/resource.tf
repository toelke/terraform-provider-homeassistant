# A target temperature, shown as an input box. The entity is
# input_number.thermostat_target.
resource "homeassistant_input_number" "thermostat_target" {
  name                = "Thermostat Target"
  icon                = "mdi:thermometer"
  min                 = 5
  max                 = 30
  step                = 0.5
  mode                = "box"
  unit_of_measurement = "°C"
}
