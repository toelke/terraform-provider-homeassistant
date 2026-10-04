resource "homeassistant_floor" "ground" {
  name  = "Ground Floor"
  level = 0
}

resource "homeassistant_label" "managed" {
  name = "Managed"
}

# Home Assistant derives the ID from the name: "living_room".
resource "homeassistant_area" "living_room" {
  name     = "Living Room"
  floor_id = homeassistant_floor.ground.id
  icon     = "mdi:sofa"
  aliases  = ["Lounge"]
  labels   = [homeassistant_label.managed.id]
  picture  = "/local/living_room.jpg"

  # Must be sensors with device class temperature and humidity.
  temperature_entity_id = "sensor.living_room_temperature"
  humidity_entity_id    = "sensor.living_room_humidity"
}

# Pin the ID, so it does not depend on the name.
resource "homeassistant_area" "office" {
  id   = "office"
  name = "Home Office"
}
