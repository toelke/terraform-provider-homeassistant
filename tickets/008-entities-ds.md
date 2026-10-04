---
status: done
depends_on: [007]
adrs: [0015]
spec: [data-sources.md]
---

# data.homeassistant_entities

## Acceptance criteria
- [x] Filters `domain`, `area` (ID or name), `label` (ID or name), `device_class`, and
      `name_pattern` each narrow the result; covered by an acceptance test that uses helpers and
      areas created through the API.
- [x] There is no `state` filter.
- [x] Output is sorted.
- [x] Values interpolated into the template are escaped.
