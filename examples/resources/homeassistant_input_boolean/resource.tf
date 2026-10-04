# Home Assistant derives the ID from the name: "guest_mode", so the entity is
# input_boolean.guest_mode.
resource "homeassistant_input_boolean" "guest_mode" {
  name = "Guest Mode"
  icon = "mdi:account-group"
}

# Pin the ID, and start switched off after every Home Assistant restart.
resource "homeassistant_input_boolean" "vacation" {
  id      = "vacation"
  name    = "Away on Vacation"
  initial = false
}
