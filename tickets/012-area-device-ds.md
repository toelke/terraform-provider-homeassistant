---
status: done
depends_on: [011]
adrs: []
spec: [registries.md]
---

# data sources: areas, area, devices, device

## Acceptance criteria
- [x] `homeassistant_area`: exactly one of `area_id`/`name`; zero or multiple matches is an
      error.
- [x] `homeassistant_devices` filters by `integration`, `manufacturer`, `model`, `area_id`, and
      `label`.
- [x] The acceptance test needs at least one device. Use an integration that HA can set up with
      no external hardware (e.g. a config-flow-based demo or helper integration), created through
      the harness.
