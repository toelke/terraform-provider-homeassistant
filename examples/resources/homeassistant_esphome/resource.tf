# An ESPHome device with an API encryption key. Changing host, port or password reconfigures the
# entry in place; changing noise_psk replaces it.
resource "homeassistant_esphome" "kitchen" {
  host      = "kitchen-sensor.local"
  noise_psk = var.kitchen_api_key

  # Let the device perform Home Assistant actions (homeassistant.action in its YAML).
  allow_service_calls = true
}

variable "kitchen_api_key" {
  type      = string
  sensitive = true
}
