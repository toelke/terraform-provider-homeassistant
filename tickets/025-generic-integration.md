---
status: todo
depends_on: [024, 004]
adrs: [0014]
spec: [integrations.md]
---

# homeassistant_integration and data.homeassistant_integration

## Acceptance criteria
- [ ] Acceptance: create a hardware-free integration through steps, read it, import it, and
      destroy it via REST DELETE.
- [ ] Deleting the entry outside Tofu leads to recreation on the next apply.
- [ ] `sensitive_data` is redacted in plan output.
- [ ] The data source matches by domain and optional title; ambiguity is an error.
