---
status: todo
depends_on: [009]
adrs: [0009, 0013]
spec: [helpers.md]
---

# homeassistant_input_boolean (and the generic helper implementation)

## Scope
A generic helper resource over `<domain>/list|create|update|delete`, reusing the generated-ID
machinery, with `input_boolean` as its first instance.

## Acceptance criteria
- [ ] CRUD, import, the collision check, and an optional `id`.
- [ ] `entity_id` is `input_boolean.<id>`.
