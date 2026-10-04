# A time of day only. The entity is input_datetime.wake_up.
resource "homeassistant_input_datetime" "wake_up" {
  name     = "Wake Up"
  has_time = true
  initial  = "06:30"
}

# A date and a time.
resource "homeassistant_input_datetime" "party" {
  name     = "Party"
  has_date = true
  has_time = true
  initial  = "2026-12-31 20:00:00"
}
