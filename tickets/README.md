# Tickets

Each ticket is one vertical slice. A slice is done when it has code, unit tests, an acceptance
test where it applies, generated docs, and an example. The numbers give a rough order; the
`depends_on` frontmatter gives the real order. Milestones are a plan, not a decision, and can be
reshuffled at any time.

| Milestone | Tickets |
|---|---|
| Foundation | 001–005, 006 (release pipeline; `v0.0.x` pre-releases from here on) |
| v0.1 | 007–020, 021 (tag) |
| v0.2 dashboards + announcement | 022, 023 (tag, demo, Reddit post) |
| v0.3 integrations | 024–026 |
| Builder functions | 027–028 |

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
