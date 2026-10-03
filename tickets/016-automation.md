---
status: todo
depends_on: [005, 015]
adrs: [0005, 0008, 0012]
spec: [automations.md]
---

# homeassistant_automation

## Acceptance criteria
- [ ] CRUD and import. After apply, `entity_id` is found via `attributes.id`, waiting out the
      asynchronous reload.
- [ ] An invalid config surfaces HA's 400 message.
- [ ] An `id` key inside `config` is rejected at plan time.
- [ ] A config written with `yamldecode(file(...))` round-trips with an empty second plan.
- [ ] A change made in the UI (POST via REST between steps) is detected as drift.
- [ ] The docs show `initial_state = false` for disabling as code.
