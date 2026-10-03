# 7. No fragment resource; reuse through HCL, with builder functions later

Date: 2026-10-03 · Status: Accepted

## Context

The original spec had `homeassistant_dashboard_fragment`, a resource that never calls an API and
only renders cards to a YAML string, so that card collections can be reused across dashboards. As
a resource it adds state, a lifecycle, and import for what is pure computation. With ADR-0005,
dashboard config is a plain HCL object.

## Decision

- There is no fragment resource. Reuse of sections and cards is done with HCL `locals`, module
  outputs, and `for` expressions. The docs show this pattern explicitly.
- Later, provider-defined **builder functions** (e.g. `provider::homeassistant::tile_card(...)`,
  `provider::homeassistant::state_trigger(...)`) return plain objects for use in any dynamic
  config. They are pure, typed, and validated at plan time.

## Consequences

- Reuse comes with nothing to implement. Builder functions are an additive feature that can ship
  in any release.
- Builder functions require OpenTofu or Terraform ≥ 1.8 (ADR-0016).
