# Scheduler integration (custom)

Schedules of the Scheduler integration (`scheduler`, nielsfaber/scheduler-component), the
backend of the scheduler card, a custom integration usually installed through HACS. Supported:
version 3.3 (ADR-0031).

## API

| Operation | Call |
|---|---|
| List | WS `scheduler` → list of schedules |
| Read | WS `scheduler/item` `{schedule_id}` → schedule, or `null` |
| Create | WS `call_service` `scheduler.add` (fields below). Returns nothing. |
| Update | WS `call_service` `scheduler.edit` `{entity_id, …fields}` |
| Delete | WS `call_service` `scheduler.remove` `{entity_id}` |
| Enable / disable | WS `call_service` `switch.turn_on` / `switch.turn_off` `{entity_id}` |

A schedule as `scheduler/item` returns it (3.3.9):

```json
{"schedule_id": "3d22e6", "name": "Office", "entity_id": "switch.schedule_office",
 "weekdays": ["mon", "tue"], "start_date": null, "end_date": null, "repeat_type": "repeat",
 "enabled": true, "tags": ["heating"],
 "timeslots": [{"start": "06:30:00", "stop": "08:00:00", "conditions": [], "condition_type": null,
   "track_conditions": false,
   "actions": [{"service": "climate.set_temperature", "entity_id": "climate.office",
     "service_data": {"temperature": 21}}]}],
 "next_entries": [0], "timestamps": ["…"]}
```

- `scheduler.add` returns no ID. The provider lists the IDs before and after the call; exactly one
  new ID must appear. Adds from one provider process are serialised for that. A failed list after
  a successful add is retried (3 attempts). If the ID still cannot be told, the error says that
  the schedule was created and how to find and import it; the provider never deletes a schedule
  it cannot identify.
- Edits refresh the switch, which serves the reads, in a task of its own. After each write, the
  provider reads until the schedule holds what it wrote, up to `timeouts.create` or
  `timeouts.update` (default 60 s), then stores that in state.
- Renaming through `scheduler.edit` removes the switch and does not add it again (3.3.9), so a new
  `name` replaces the schedule.
- The services reject `null` for `stop`, `condition_type`, an action's `entity_id`, and an empty
  `conditions`, so the provider leaves unset ones out.
- Without the integration, the commands fail with `unknown_command`, and the services with
  `not_found` ("Service scheduler.… not found."). Both become one error, "Scheduler integration
  missing", with how to install it.

## `homeassistant_scheduler_schedule`

```hcl
resource "homeassistant_scheduler_schedule" "office_heating" {
  name     = "Office heating"
  weekdays = ["workday"]
  timeslots = [{
    start   = "06:30:00"
    stop    = "08:00:00"
    actions = [{ action = "climate.set_temperature", entity_id = "climate.office", data = jsonencode({ temperature = 21 }) }]
  }]
}
```

| Attribute | Kind | Notes |
|---|---|---|
| `schedule_id` | computed | Import ID. Kept across updates |
| `entity_id` | computed | `switch.schedule_<slug of name>` |
| `name` | required | Change replaces. No leading or trailing whitespace |
| `weekdays` | optional, default `["daily"]` | `mon`…`sun`, `daily`, `workday`, `weekend`; unique |
| `start_date`, `end_date` | optional | `YYYY-MM-DD`; `end_date` not before `start_date` |
| `repeat_type` | optional, default `repeat` | `repeat` or `pause`; `single` rejected (ADR-0031) |
| `tags` | optional set, default empty | |
| `enabled` | optional | Unset: not managed. Set: enforced through the switch (ADR-0031) |
| `timeouts` | optional | `create`, `update`: how long to wait for the write to read back (60 s) |
| `timeslots` | required list, ≥ 1 | see below |

A timeslot has `start` (required), `stop`, `actions` (required, ≥ 1), `conditions` (default
empty), `condition_type` (`and`/`or`, unset means `or` to the integration) and
`track_conditions` (default `false`). An action has `action` (`<domain>.<name>`), `entity_id`, and
`data` (a JSON object as text, default `{}`, compared semantically). A condition has
`entity_id`, `attribute`, `match_type` (`is`, `not`, `below`, `above`) and `value`, a string that
must be a number in JSON syntax (`20`, `-0.5`, not `.5` or `NaN`) with `below` and `above` and is
then sent as a number. The integration stores it as a number, so on read a `below` or `above`
value that is the same number keeps its spelling (`20.50` stays `20.50`). `is` and `not` values
are strings, compared as strings; any change to them is drift.

Plan-time validation:

- Times are `HH:MM:SS`, or `sunrise`/`sunset` with `+`/`-` and an offset `HH:MM:SS`.
- Timeslots with fixed times must not overlap. A `stop` of `00:00:00` is the end of the day, and a
  `stop` at or before `start` continues past midnight. Two single times must differ. Timeslots
  relative to the sun are not checked.
- Creating a schedule checks that the integration is set up (`scheduler`), so the plan fails
  without it.

Read: `null` from `scheduler/item` removes the resource from state. Changes made in the card are
drift. `enabled` is read only when set.

## `data.homeassistant_scheduler_schedules`

Lists every schedule, sorted by `name`, then `schedule_id`: `schedule_id`, `name`, `entity_id`,
`enabled`, `weekdays`, `tags`, and `entities` (the entities the actions act on). Without the
integration, reading fails with the "Scheduler integration missing" error, like every other data
source fails when what it reads does not exist.

## Tests

`internal/scheduleracc` runs the acceptance tests in a container with Scheduler 3.3.9 installed
(`acctest.Install(acctest.Scheduler)` downloads the pinned release from GitHub and checks its
SHA-256). The packages `resources` and `datasources` test the behaviour without the integration.
