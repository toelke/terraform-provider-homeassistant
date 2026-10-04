# A four-digit code, hidden in the UI. The entity is input_text.door_code.
resource "homeassistant_input_text" "door_code" {
  name    = "Door Code"
  min     = 4
  max     = 4
  pattern = "^[0-9]{4}$"
  mode    = "password"
}
