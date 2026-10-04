# Home Assistant derives the ID from the name: "ground_floor".
resource "homeassistant_floor" "ground" {
  name    = "Ground Floor"
  level   = 0
  icon    = "mdi:home-floor-0"
  aliases = ["Downstairs"]
}

# Pin the ID, so it does not depend on the name.
resource "homeassistant_floor" "basement" {
  id    = "basement"
  name  = "Cellar"
  level = -1
}
