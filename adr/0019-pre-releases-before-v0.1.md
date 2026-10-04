# 19. `v0.0.x` pre-releases before `v0.1.0`

Date: 2026-10-04 · Status: Accepted · Supersedes the versioning point of ADR-0018

## Context

ADR-0018 says "Semver, starting at `v0.1.0`". The plan in `tickets/` sets up the release pipeline
early (ticket 006), so the provider can be installed from the registries and the signing checked
long before the v0.1 feature set is complete (ticket 021). The agent that built the pipeline
raised the conflict in PR #10, and the maintainer decided it.

## Decision

- Releases start at `v0.0.1`. Any merged ticket may be released as a `v0.0.x` pre-release.
- `v0.1.0` is tagged when the v0.1 milestone in `tickets/README.md` is complete.
- Everything else in ADR-0018 stands: semver, signed releases, and breaking changes allowed in
  minor versions before v1.0 and noted in `CHANGELOG.md`.

## Consequences

- `v0.0.x` releases may contain any subset of the planned features. `CHANGELOG.md` says what is
  in each one.
- Every user-visible change needs a `CHANGELOG.md` entry when it merges, not when it is
  released. The definition of done in `tickets/README.md` requires this.
