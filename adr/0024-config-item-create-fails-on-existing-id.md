# 24. Creating an automation, script, or scene fails if its `id` is taken

Date: 2026-10-05 · Status: Accepted · Extends ADR-0008

## Context

Issue #41, finding 5: HA's config editor endpoint, `POST /api/config/<domain>/config/<id>`,
creates an item or replaces the one with that ID. The docs tell users to copy YAML out of the
editor and remove its `id:` line, so they naturally reuse that ID as the resource's `id`. Then
`tofu apply` silently replaces the item made in the UI, and a later `tofu destroy` deletes it.
Resources with generated IDs fail on a collision and point to `tofu import` (ADR-0009), so the
two kinds of ID behaved differently.

The options put to the maintainer in the issue:

1. Fail: in Create, read the item first, and if it exists, fail with an error that suggests
   importing it or choosing another `id`.
2. Take the existing item over, and document that.

The maintainer accepted the recommendation, option 1.

## Decision

Create of `homeassistant_automation`, `homeassistant_script`, and `homeassistant_scene` first
reads `GET /api/config/<domain>/config/<id>`. If the item exists, the create fails with an error
on `id` that names the `tofu import` command, and nothing is written. Update still writes
unconditionally: the item is the resource's own.

## Consequences

- An item made in the UI is never replaced without an import, as for generated IDs.
- Each create costs one extra read.
- The check and the save are two requests, so an item created in between is still replaced.
  That window is too small to matter in practice.
- `create_before_destroy` cannot work for a change that keeps the `id`; it couldn't before
  either, because the destroy then deleted the new item.
