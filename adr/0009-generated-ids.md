# 9. Generated IDs: computed, collision-checked, optional `id` via create-then-rename

Date: 2026-10-03 · Status: Accepted

## Context

For floors, areas, labels, and input helpers, HA assigns the ID as `slugify(name)`. On a clash,
it appends `_2`, `_3`, …. Create calls accept no ID. A dashboard's `dashboard_id` is
`slugify(url_path)`. Renaming an object later keeps its original ID.

## Decision

- `id` is a computed attribute.
- After create, the provider compares the returned ID with the expected one. A mismatch is a hard
  error: the created object is deleted again, and the message lists the ways out (import the
  existing object, rename or remove it, or choose another name).
- `id` may optionally be set by the user. The provider then validates it as a slug, creates the
  object with `name = id`, and immediately updates it to the real `name` (**create-then-rename**).
  The same collision check applies.
- Changing `name` is an in-place update. Changing `id` forces replacement.

## Consequences

- IDs are predictable, and collisions surface at apply time instead of silently producing `_2`.
- With create-then-rename, the `id`-as-name is briefly visible in HA, and creation takes two calls.
- Renaming a display name (e.g. "Bedroom" → "Master Bedroom") does not change references, as long
  as the user pinned the `id` or HA's original slug is kept in state.
