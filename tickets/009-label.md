---
status: done
depends_on: [004]
adrs: [0009]
spec: [registries.md]
---

# homeassistant_label (and the generated-ID machinery)

## Scope
The first generated-ID resource. Build the reusable pieces here:
- A slugify that matches HA (test vectors taken from HA's `slugify` tests).
- Create-then-rename for an optional `id`.
- The collision check, which deletes the object and returns an actionable error.

## Acceptance criteria
- [x] CRUD and import acceptance tests.
- [x] Acceptance: a collision (a label with the same name already exists) errors and leaves no
      `_2` behind.
- [x] Acceptance: with `id` set, the ID is kept and the name equals `name`.
- [x] Renaming is an in-place update.
