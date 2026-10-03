# 14. Integrations: generic resource first, typed later, secrets split

Date: 2026-10-03 · Status: Accepted

## Context

Config entries are created through REST config flows. Their input data **cannot be read back**:
`config_entries/get` returns domain, title, state, and similar, but not the data. Changing data
needs a reconfigure flow, which only some integrations implement. Hue needs a physical button
press during apply. Input data often contains secrets.

## Decision

- `homeassistant_integration` (generic) comes first. It takes a `domain` and ordered `step`
  blocks. Each step has a `step_id`, a plain `data` map, and a `sensitive_data` map (marked
  sensitive). The two maps are merged when the step is submitted.
- If HA asks for a `step_id` the config does not provide, or aborts, the flow is aborted
  (`DELETE .../flow/{id}`) and HA's reason is surfaced.
- Any change to inputs forces replacement. Drift is detected only as "config entry is gone".
- Typed `homeassistant_esphome` and `homeassistant_mqtt` come later. They use reconfigure flows
  where the integration supports them. Hue and Z-Wave are not planned.
- Write-only attributes (OpenTofu/Terraform ≥ 1.11) are not used yet. Revisit once the minimum
  version allows it.
- `data.homeassistant_integration` reads a config entry by domain and optional title.

## Consequences

- Secrets live in state (marked sensitive), so state must be protected.
- Recreation is the update mechanism for the generic resource. Users see entities briefly
  disappear and come back.
