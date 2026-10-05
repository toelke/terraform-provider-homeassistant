# Tickets

Each ticket is one vertical slice. A slice is done when it has code, unit tests, an acceptance
test where it applies, generated docs, and an example. The numbers give a rough order; the
`depends_on` frontmatter gives the real order; `scripts/tickets.py` shows it.

Releases cut whatever is merged (ADR-0019): `v0.0.x` pre-releases at any time, `v0.1.0` once
ticket 021's dependencies are done. The announcement (023) follows v0.1.0. Later releases come
when there is something worth releasing.

Status values: `todo`, `doing`, `done`.

A ticket inserted later gets a letter suffix (`002b`), so existing numbers never change. It sorts
right after its base number.

Definition of done, for every ticket:

- pre-commit passes.
- `go test -short ./...` is green.
- Acceptance tests are green against both HA versions in the CI matrix.
- `docs/` is regenerated.
- The `spec/` file is updated if behaviour deviated from it.
- `CHANGELOG.md` has an entry under `Unreleased` for every user-visible change.
