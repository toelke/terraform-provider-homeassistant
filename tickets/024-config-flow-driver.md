---
status: todo
depends_on: [005]
adrs: [0004, 0014]
spec: [integrations.md]
---

# Config-flow driver

## Acceptance criteria
- [ ] Unit tests against a fake HTTP server cover: a single step, multiple steps, an unexpected
      `step_id` (flow aborted via DELETE), form errors, `abort` reasons, reconfigure
      (`entry_id` passed, ends with `reconfigure_successful`), and unsupported result types.
