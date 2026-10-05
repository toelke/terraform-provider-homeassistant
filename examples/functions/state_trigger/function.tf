# State triggers; further trigger options go in an optional trailing object.
locals {
  # Same as { trigger = "state", entity_id = "binary_sensor.hall_motion", to = "on" }
  motion = provider::homeassistant::state_trigger("binary_sensor.hall_motion", { to = "on" })
  # Quote "for": HCL reads `{ for` as the start of a for expression.
  doors_open = provider::homeassistant::state_trigger(
    ["binary_sensor.front_door", "binary_sensor.back_door"],
    { "for" = { minutes = 5 }, to = "on" },
  )
}
