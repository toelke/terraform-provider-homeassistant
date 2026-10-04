# The one config entry of a domain.
data "homeassistant_integration" "met" {
  domain = "met"
}

# Pick one of several entries by its title.
data "homeassistant_integration" "chores" {
  domain = "local_calendar"
  title  = "Chores"
}

output "met_state" {
  value = data.homeassistant_integration.met.state
}
