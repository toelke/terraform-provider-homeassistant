---
status: done
depends_on: []
adrs: [0002, 0016, 0018]
spec: [overview.md]
---

# Scaffold provider and configuration

## Scope
- Go module `github.com/toelke/terraform-provider-homeassistant`, `main.go`, and a
  plugin-framework provider with protocol v6.
- Provider schema: `url`, `token` (sensitive), `insecure`, `timeout`, each with a
  `HOMEASSISTANT_*` environment fallback.
- `Configure` builds an `HAClient` (REST + lazy WS, both stubbed in this ticket) and hands it to
  resources and data sources.

## Acceptance criteria
- [x] `tofu init` with a dev override loads the provider.
- [x] A missing `url` or `token` gives a clear diagnostic naming the argument and environment
      variable.
- [x] Unit test: configuration precedence (argument over environment).
