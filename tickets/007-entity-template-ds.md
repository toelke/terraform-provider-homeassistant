---
status: todo
depends_on: [005]
adrs: [0015]
spec: [data-sources.md]
---

# data.homeassistant_entity and data.homeassistant_template

## Acceptance criteria
- [ ] `attributes` is dynamic and keeps numbers and booleans.
- [ ] A missing entity is an error with a hint.
- [ ] A template render error (HTTP error, or a 200 body with `Error rendering template`) is an
      error diagnostic.
- [ ] Acceptance tests use the `sun.sun` entity and a simple template.
