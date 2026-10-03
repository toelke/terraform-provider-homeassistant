---
status: done
depends_on: [001, 003]
adrs: [0004, 0011]
spec: [overview.md, data-sources.md]
---

# REST client and data.homeassistant_config

## Scope
- REST client with typed errors (`ErrNotFound`, `ErrUnauthorized`, `ErrUnreachable`) and a
  bearer token. An unreachable host produces the `-refresh=false` hint (ADR-0011).
- `data.homeassistant_config`.

## Acceptance criteria
- [x] Unit tests for error mapping.
- [x] Acceptance: reads `version` and matches the container tag.
