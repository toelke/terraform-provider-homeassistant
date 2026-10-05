# Presence and notifications

The household's people are one map, and everything per person is generated from it with
`for_each`: a "<name> is home" automation from a shared template, which messages the household
and runs a script that lights the porch for two minutes, and a "find my phone" script that makes
the phone ring at full volume through the companion app. A second template announces one
person's arrival to the other: by voice on the phone, after pausing whatever it plays, or else as
a notification on the hallway tablet, whose `device_id` comes from the `homeassistant_device`
data source instead of being copied from the UI. Scripts and the front-door automation are
written in HCL with the `action`, `delay` and `state_trigger` provider functions, and the
template's `%{ for }` loop builds an `or` condition over every person. The message action is a
variable, so the same configuration works with `notify.*` or a messenger's `rest_command`.
