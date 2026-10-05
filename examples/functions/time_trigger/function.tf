# Time triggers: a fixed time, or the time held by an entity.
locals {
  # Same as { trigger = "time", at = "07:30:00" }
  morning = provider::homeassistant::time_trigger("07:30:00")
  wake_up = provider::homeassistant::time_trigger(
    ["input_datetime.wake_up", { entity_id = "sensor.phone_next_alarm", offset = "-00:10:00" }],
    { weekday = ["mon", "tue", "wed", "thu", "fri"] },
  )
}
