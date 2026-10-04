---
status: done
depends_on: [005]
adrs: [0015]
spec: [data-sources.md]
---

# data.homeassistant_entity and data.homeassistant_template

## Acceptance criteria
- [x] `attributes` is dynamic and keeps numbers and booleans.
- [x] A missing entity is an error with a hint.
- [x] A template render error (HTTP error, or a 200 body with `Error rendering template`) is an
      error diagnostic.
- [x] Acceptance tests use the `sun.sun` entity and a simple template.
