---
status: done
depends_on: [035]
issues: [43]
adrs: [0002]
spec: []
---

# Validators from `terraform-plugin-framework-validators`

From issue #43, finding 13. Six hand-rolled validators repeat the same boilerplate, and
`int64Between` encodes "no maximum" as `max <= min`.

## Scope
- Add `github.com/hashicorp/terraform-plugin-framework-validators`, the issue's first proposal.
  It is HashiCorp's companion to the framework, like `terraform-plugin-framework-timeouts`.
- Replace `oneOfValidator` with `stringvalidator.OneOf`; `configItemIDValidator`,
  `urlPathValidator`, and `entityIDValidator` with `stringvalidator.RegexMatches`; and
  `int64Between` with `int64validator.Between` or `AtLeast`.
- Keep `slugValidator`, which depends on `client.Slugify`.

## Acceptance criteria
- [x] The existing validator unit tests pass, adapted where only the error text changed.
- [x] Every error still names what a valid value looks like, e.g. "must contain a hyphen".
- [x] `docs/` regenerated; a changed validator description shows up there.
- [x] `CHANGELOG.md` entry if an error message users see changes.
