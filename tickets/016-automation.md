---
status: done
depends_on: [005, 015]
adrs: [0005, 0008, 0012]
spec: [automations.md]
---

# homeassistant_automation

## Acceptance criteria
- [x] CRUD and import. After apply, `entity_id` is found via `attributes.id`, waiting out the
      asynchronous reload.
- [x] An invalid config surfaces HA's 400 message.
- [x] An `id` key inside `config` is rejected at plan time.
- [x] A config written with `yamldecode(file(...))` round-trips with an empty second plan.
- [x] A change made in the UI (POST via REST between steps) is detected as drift.
- [x] The docs show `initial_state = false` for disabling as code.
