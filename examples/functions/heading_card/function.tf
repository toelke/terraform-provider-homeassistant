locals {
  # Same as { type = "heading", heading = "Quick actions" }
  quick_actions_heading = provider::homeassistant::heading_card("Quick actions")
}
