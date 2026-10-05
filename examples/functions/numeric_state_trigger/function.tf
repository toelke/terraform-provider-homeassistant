# A numeric state trigger; options must set above, below, or both.
locals {
  too_warm = provider::homeassistant::numeric_state_trigger("sensor.living_room_temperature", {
    above = 25
    "for" = "00:10:00"
  })
}
