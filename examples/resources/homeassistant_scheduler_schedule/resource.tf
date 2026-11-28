# Needs the Scheduler integration (scheduler-component, e.g. from HACS) in Home Assistant.

# A weekly heating plan for one room: warm in the morning and evening on workdays.
resource "homeassistant_scheduler_schedule" "office_heating" {
  name     = "Office heating"
  weekdays = ["workday"]
  tags     = ["heating"]
  timeslots = [
    {
      start   = "06:30:00"
      stop    = "08:00:00"
      actions = [{ action = "climate.set_temperature", entity_id = "climate.office", data = jsonencode({ temperature = 21 }) }]
    },
    {
      start   = "08:00:00"
      stop    = "17:00:00"
      actions = [{ action = "climate.set_temperature", entity_id = "climate.office", data = jsonencode({ temperature = 17 }) }]
    },
  ]
}

# Lights on half an hour before sunset, unless the room is bright enough.
resource "homeassistant_scheduler_schedule" "porch_light" {
  name = "Porch light"
  timeslots = [{
    start          = "sunset-00:30:00"
    condition_type = "and"
    conditions     = [{ entity_id = "sensor.porch_illuminance", match_type = "below", value = "50" }]
    actions        = [{ action = "light.turn_on", entity_id = "light.porch" }]
  }]

  # Enforce that the schedule stays switched on. Leave it unset to manage the switch in Home Assistant.
  enabled = true
}
