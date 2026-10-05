# An automation built from the trigger, condition, and action functions.
resource "homeassistant_automation" "hall_light" {
  id = "hall_light"
  config = {
    alias      = "Hall light on motion"
    mode       = "restart"
    triggers   = [provider::homeassistant::state_trigger("binary_sensor.hall_motion", { to = "on" })]
    conditions = [provider::homeassistant::state_condition("sun.sun", "below_horizon")]
    actions = [
      provider::homeassistant::action("light.turn_on", { target = { entity_id = "light.hall" } }),
      # Same as { delay = { minutes = 2 } }; "00:02:00" or 120 work too.
      provider::homeassistant::delay({ minutes = 2 }),
      provider::homeassistant::action("light.turn_off", { target = { entity_id = "light.hall" } }),
    ]
  }
}
