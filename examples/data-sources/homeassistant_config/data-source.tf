data "homeassistant_config" "this" {}

output "home_assistant_version" {
  value = data.homeassistant_config.this.version
}

output "has_mqtt" {
  value = contains(data.homeassistant_config.this.components, "mqtt")
}
