# The connection to an MQTT broker over TLS. Home Assistant allows only one.
resource "homeassistant_mqtt" "broker" {
  broker   = "mqtt.home.local"
  port     = 8883
  username = "homeassistant"
  password = var.mqtt_password

  tls = {
    ca_certificate = file("${path.module}/ca.pem")
  }

  # Options: set through the options flow, without reconnecting.
  discovery_prefix = "homeassistant"
  birth_message    = { topic = "homeassistant/status", payload = "online", retain = true }
}

variable "mqtt_password" {
  type      = string
  sensitive = true
}
