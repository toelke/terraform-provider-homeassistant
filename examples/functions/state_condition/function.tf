# State conditions: one state, or a list of states any of which matches.
locals {
  # Same as { condition = "state", entity_id = "person.alice", state = "home" }
  alice_home = provider::homeassistant::state_condition("person.alice", "home")
  armed      = provider::homeassistant::state_condition("alarm_control_panel.home", ["armed_away", "armed_night"])
}
