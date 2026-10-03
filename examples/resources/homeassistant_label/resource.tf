# Home Assistant derives the ID from the name: "managed_by_tofu".
resource "homeassistant_label" "managed" {
  name  = "Managed by Tofu"
  color = "indigo"
  icon  = "mdi:robot"
}

# Pin the ID, so it does not depend on the name.
resource "homeassistant_label" "critical" {
  id          = "critical"
  name        = "Critical Devices"
  color       = "#d32f2f"
  description = "Devices that must never be offline."
}
