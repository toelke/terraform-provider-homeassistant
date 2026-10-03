---
status: todo
depends_on: [007]
adrs: [0015]
spec: [data-sources.md]
---

# data.homeassistant_entities

## Acceptance criteria
- [ ] Filters `domain`, `area` (ID or name), `label` (ID or name), `device_class`, and
      `name_pattern` each narrow the result; covered by an acceptance test that uses helpers and
      areas created through the API.
- [ ] There is no `state` filter.
- [ ] Output is sorted.
- [ ] Values interpolated into the template are escaped.
