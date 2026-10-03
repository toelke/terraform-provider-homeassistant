# 15. Data source adjustments

Date: 2026-10-03 · Status: Accepted

## Context

The original `homeassistant_entities` data source could filter by current `state`. That makes
plans, and every `for_each` over the result, depend on whether a light happens to be on.
`homeassistant_entity.attributes` forced every value to a string.

## Decision

- Remove the `state` filter from `homeassistant_entities`.
- `attributes` (on `homeassistant_entity` and in `homeassistant_entities.entities`) is a dynamic
  value that keeps JSON types.
- `area` and `label` filters accept an ID or a name, and are resolved through the template API
  (`area_entities`, `label_entities`).
- `homeassistant_entity` errors when the entity is missing. A 404 message suggests checking the
  ID.

## Consequences

Data source results are stable across plans unless the registry actually changes.
