---
status: todo
depends_on: [004, 015]
adrs: [0005, 0006, 0007]
spec: [dashboards.md]
---

# homeassistant_dashboard

## Acceptance criteria
- [ ] Acceptance: create → appears in `lovelace/dashboards/list` → change the title (metadata
      update only) → change the config → import by `url_path` → destroy → gone.
- [ ] A `url_path` without a hyphen is rejected at plan time.
- [ ] Docs example: one section local reused in two dashboards, plus a per-area generated
      section.
