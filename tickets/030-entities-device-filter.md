---
status: done
depends_on: [008, 012]
issues: [30]
adrs: [0022]
spec: [data-sources.md]
---

# data.homeassistant_entities: `device_id` filter

From issue #30. Duplicate friendly names usually belong to different devices.

## Scope
- New optional `device_id` (string) filter, resolved through `device_entities()` like `area` and
  `label`. It takes an ID only. An unknown device matches nothing.
- Docs: an example combining it with `data.homeassistant_device` to look up by device name.
- `CHANGELOG.md` entry under `Unreleased`, referencing #30.

## Acceptance criteria
- [x] Acceptance test: two entities with the same friendly name on two devices. `device_id`
      returns only the entity of that device.
- [x] An unknown device ID returns no entities and no error.
- [x] The device ID is escaped like the `area` and `label` values (unit test).
- [x] The PR body contains "Closes #30" only if ticket 029 is already merged; otherwise "Part of
      #30".
