---
status: done
depends_on: [016, 017, 018, 031]
issues: [41]
adrs: [0008, 0024, 0011]
spec: [automations.md, overview.md]
---

# Config items: collision check, `timeouts`, cheaper entity lookup

From issue #41 (findings 5, 7, 8, 16, and the config-item part of 21).

## Scope
- Create of an automation, script, or scene fails if an item with its `id` exists (ADR-0024).
- `timeouts = { create }` on the three resources, default 60s. Create polls for the entity with
  the backoff of `waitFor`; the client looks up once, without its own polling loop.
- Read: a script's `entity_id` is `script.<id>`, without a request. For automations and scenes,
  the entity in state is checked first with `GET /api/states/<entity_id>`; all states are read
  only if it is gone or no longer belongs to the item.
- HA rejecting a `config` (REST 400 for config items, a WS error for dashboards) is attached to
  the `config` attribute.
- `CHANGELOG.md` entries under `Unreleased`.

## Acceptance criteria
- [x] Acceptance test: an automation that already exists with the `id` is not overwritten, and
      the error names `tofu import`.
- [x] Acceptance test: an automation with `timeouts.create` set applies.
- [x] Unit tests of the config-item resource: Create waits for the entity; Create without an
      entity keeps a state with a null `entity_id` and an error; Create fails on an existing ID;
      a rejected config is on `config`; Read keeps `entity_id` while the entity is missing;
      Read of a script reads no states.
- [x] `spec/automations.md` names the `timeouts` attribute and the collision check.
