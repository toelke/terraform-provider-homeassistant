# Every schedule of the Scheduler integration, e.g. to find what to import.
data "homeassistant_scheduler_schedules" "all" {}

# schedule_id => name of the heating schedules. schedule_id is unique; names may repeat or be null.
output "heating_schedules" {
  value = {
    for s in data.homeassistant_scheduler_schedules.all.schedules : s.schedule_id => s.name
    if contains(s.tags, "heating")
  }
}
