# 22. Entity lookups filter on registry data through template functions

Date: 2026-10-05 · Status: Accepted

## Context

Issue #30: `homeassistant_entities` can't tell apart entities that share a friendly name when one
of them is hidden. Hidden entities are usually stale leftovers, such as an old MQTT discovery
entity, so a lookup with "exactly one match" fails. The hidden flag lives in the entity
registry, not in `/api/states`. The maintainer decided the options below in the issue's grilling
session.

HA's template engine already reads the registry: `is_hidden_entity(entity_id)` is true when
`hidden_by` is set, by the user or by an integration, and false for entities without a registry
entry. `device_entities(device_id)` lists a device's entities. Both exist in every HA release of
the support window (checked in 2026.4.4).

## Decision

- **Filters on registry data use template functions**, like `area` and `label` (ADR-0015), so
  `homeassistant_entities` stays REST-only and never opens a WebSocket (ADR-0004).
- **`hidden`** (bool) has three states: `false` gives only entities that aren't hidden, `true`
  gives only hidden ones, and unset applies no filter. Unset keeps today's behaviour. Excluding
  hidden entities by default was rejected, because it would silently change existing lookups.
- **Each entry in `entities` gets a `hidden` flag**, so callers can also filter in HCL.
- **`device_id`** filters by device ID only. Resolving device names belongs to
  `data.homeassistant_device`, which errors on ambiguous names. The template function
  `device_id(name)` would silently pick one device, and duplicate device names are exactly the
  situation that motivated this.
- **There is no `disabled` filter.** HA removes disabled entities from `/api/states`, so the
  filter would only matter for the few seconds before that happens. The docs say so.

## Consequences

- One extra template render per read, to find the hidden entities among the states.
- An entity that becomes hidden changes the result on the next refresh, like any registry change.
- Device names are resolved with a separate `data.homeassistant_device`, so a configuration has
  two data sources where one might have been enough.
