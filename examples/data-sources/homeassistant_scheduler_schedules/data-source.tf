# Every schedule of the Scheduler integration, e.g. to find what to import.
data "homeassistant_scheduler_schedules" "all" {}

output "heating_schedules" {
  value = {
    for s in data.homeassistant_scheduler_schedules.all.schedules : s.name => s.schedule_id
    if contains(s.tags, "heating")
  }
}
