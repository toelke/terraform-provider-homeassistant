# A button that automations can trigger on. The entity is input_button.doorbell.
resource "homeassistant_input_button" "doorbell" {
  name = "Doorbell"
  icon = "mdi:bell"
}
