# A dropdown that starts at "eco" whenever Home Assistant starts. The entity is
# input_select.heating_mode.
resource "homeassistant_input_select" "heating_mode" {
  name    = "Heating Mode"
  options = ["off", "eco", "comfort"]
  initial = "eco"
}
