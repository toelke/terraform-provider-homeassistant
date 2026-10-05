# A time condition has no required arguments; options must set after, before, or weekday.
locals {
  # Same as { condition = "time", after = "22:00:00", before = "06:00:00" }
  night = provider::homeassistant::time_condition({ after = "22:00:00", before = "06:00:00" })
}
