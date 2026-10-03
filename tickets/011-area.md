---
status: todo
depends_on: [010]
adrs: [0009]
spec: [registries.md]
---

# homeassistant_area

## Acceptance criteria
- [ ] CRUD and import, including `floor_id`, `labels`, `aliases`, `picture`, and
      `temperature_entity_id`/`humidity_entity_id`.
- [ ] Moving an area between floors is an in-place update.
