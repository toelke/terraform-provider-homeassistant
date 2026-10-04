---
status: done
depends_on: [004, 015]
adrs: [0005, 0006, 0007]
spec: [dashboards.md]
---

# homeassistant_dashboard

## Acceptance criteria
- [x] Acceptance: create → appears in `lovelace/dashboards/list` → change the title (metadata
      update only) → change the config → import by `url_path` → destroy → gone.
- [x] A `url_path` without a hyphen is rejected at plan time.
- [x] Docs example: one section local reused in two dashboards, plus a per-area generated
      section.
