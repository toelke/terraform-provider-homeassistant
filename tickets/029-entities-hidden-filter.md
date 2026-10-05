---
status: todo
depends_on: [008]
adrs: [0022]
spec: [data-sources.md]
---

# data.homeassistant_entities: `hidden` filter and flag

From issue #30. Lookups by area and name fail on stale hidden duplicates.

## Scope
- New optional `hidden` (bool) filter with three states: `false`, `true`, or unset (no filter).
- Each entry in `entities` gets `hidden` (bool).
- Both come from one `is_hidden_entity` template render per read. No WebSocket.
- Docs: the `hidden` filter, a note that there is no `disabled` filter and why, and an example
  of the issue's "exactly one match" lookup with `hidden = false`.
- `CHANGELOG.md` entry under `Unreleased`, referencing #30.

## Acceptance criteria
- [ ] Acceptance test: two entities with the same friendly name in one area, one hidden by the
      user (`config/entity_registry/update` with `hidden_by: user`). `hidden = false` returns
      only the visible one, `hidden = true` only the hidden one, unset returns both.
- [ ] An entity without a registry entry (e.g. one created by `POST /api/states`) counts as not
      hidden.
- [ ] `entities[...].hidden` is correct for both entities.
- [ ] The PR body contains "Closes #30" only if ticket 030 is already merged; otherwise "Part of
      #30".
