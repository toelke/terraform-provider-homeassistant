# 31. Schedules of the Scheduler integration, a custom integration

Date: 2026-11-28 · Status: Accepted · Makes an exception to ADR-0012

## Context

Issue #118 asks for `homeassistant_scheduler_schedule`: the schedules of the Scheduler integration
(nielsfaber/scheduler-component, the backend of the scheduler card). It is the first resource for
a custom integration, one that HACS installs rather than Home Assistant itself. That raises
questions that no earlier ADR answers:

- **Versions.** The integration is released on its own schedule. Its storage format and API have
  changed between major versions. Version 3.3 has kept its WebSocket commands, services and
  schemas since 2024.
- **Tests.** Acceptance tests run a stock Home Assistant image (ADR-0017), which lacks the
  integration.
- **`enabled`.** The issue asks for an `enabled` attribute that turns the schedule's switch
  (`switch.schedule_*`) on or off. ADR-0012 says the provider never manages runtime state such as
  whether an automation is enabled. Unlike an automation's, a schedule's `enabled` is a field of
  the schedule in the integration's own storage, next to its timeslots; the switch writes it there.
  A `pause` schedule turns its switch off itself after it ran.
- **`repeat_type = "single"`.** The integration deletes such a schedule after it ran. The next
  apply would create it again, so it would run again.
- **`actions`.** The integration stores an action as `{service, entity_id, service_data}`.

Checked against version 3.3.9 in Home Assistant 2026.11: `scheduler.add` does not return the new
schedule's ID, and renaming a schedule through `scheduler.edit` removes its switch and does not
add it again, so later reads return stale data until Home Assistant restarts.

The options put to the maintainer:

1. `enabled`: (a) optional, unset means unmanaged; (b) left out, following ADR-0012; (c) always
   managed, default `true`.
2. Actions: (a) `{ action, entity_id, data }` with `data` as JSON text, in today's naming; (b) the
   integration's names `{ service, entity_id, service_data }`.
3. Versions and tests: support 3.3, and in the tests (a) vendor a copy, (b) download the pinned
   release, or (c) do without acceptance tests.
4. `single`: (a) reject it at plan time; (b) allow it and document that it is created again.

The maintainer chose 1a, 2a, 3b and 4a.

## Decision

- `homeassistant_scheduler_schedule` and the data source `homeassistant_scheduler_schedules`
  support version 3.3 of the Scheduler integration. They write through its services
  `scheduler.add`, `scheduler.edit` and `scheduler.remove`, and read through its WebSocket
  commands `scheduler` and `scheduler/item`.
- The provider and every other resource work without the integration. Nothing is checked at
  configure time. Planning to create a schedule, reading or importing one, and reading the data
  source fail with an error that names the integration and how to install it.
- `enabled` is optional. Unset, the provider neither writes nor reads it, so switching the
  schedule in Home Assistant is no drift. Set, it is enforced through `switch.turn_on` and
  `switch.turn_off`, and a different value read back is drift. This is the one exception to
  ADR-0012: the value is part of the schedule's stored configuration.
- An action is `{ action, entity_id, data }`, with `data` a JSON object as text, like the `data`
  of `homeassistant_schedule`. The provider maps it to the integration's names.
- `repeat_type` accepts `repeat` and `pause`. `single` is rejected at plan time.
- Every change but `name` updates the schedule in place and keeps its `schedule_id`. A new `name`
  replaces it, because renaming in place loses the switch.
- The provider finds a new schedule's ID by comparing the list before and after `scheduler.add`,
  and serialises its own adds for that.
- Acceptance tests that need the integration live in their own package, whose container gets the
  pinned 3.3.9 release, downloaded from GitHub and checked against its SHA-256
  (`acctest.Install`). The other packages test the behaviour without the integration.

## Consequences

- A heating plan built in the card can be imported by `schedule_id` and kept in code.
- Acceptance tests need network access to GitHub, as the upgrade tests need it to the registry.
- A new major version of the integration needs its own check, and maybe a new ADR.
- Renaming a schedule gives it a new `schedule_id` and recreates its switch with a new entity ID.
- `enabled = true` on a `pause` schedule turns it on again at every apply after it ran.
