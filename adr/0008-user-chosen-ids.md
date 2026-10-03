# 8. Automations, scripts, and scenes require a user-chosen `id`

Date: 2026-10-03 · Status: Accepted

## Context

The REST config endpoints take the ID as a path segment, and the caller chooses it. The original
spec proposed deriving a default ID from a hash of the resource address. The plugin protocol never
passes the resource address to the provider, so that is not possible.

## Decision

`id` is required on `homeassistant_automation`, `homeassistant_script`, and
`homeassistant_scene`. It is validated at plan time as `^[a-z0-9_]+$`. Changing it forces
replacement.

## Consequences

- IDs are readable and stable. They appear in URLs and, for scripts, in the entity ID.
- An automation's *entity ID* still derives from its alias, so the provider computes `entity_id`
  by looking up the state whose `attributes.id` matches.
