---
status: done
depends_on: [024, 004]
adrs: [0014]
spec: [integrations.md]
---

# homeassistant_integration and data.homeassistant_integration

## Acceptance criteria
- [x] Acceptance: create a hardware-free integration through steps, read it, import it, and
      destroy it via REST DELETE.
- [x] Deleting the entry outside Tofu leads to recreation on the next apply.
- [x] `sensitive_steps` is redacted in plan output.
- [x] The data source matches by domain and optional title; ambiguity is an error.
