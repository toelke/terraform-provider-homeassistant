---
status: done
depends_on: [006, 012, 013, 014, 020]
adrs: [0018]
spec: []
---

# Release v0.1.0

v0.1.0 releases everything merged by then (ADR-0019), including tickets beyond 020 that are
already done, such as dashboards, integrations, and builder functions.

## Scope
- In a PR: move the `Unreleased` entries in `CHANGELOG.md` to `0.1.0`, and replace the README's
  "early development" status with a feature list and an install snippet.
- After the PR is merged, the human tags `v0.1.0`.

## Acceptance criteria
- [ ] `v0.1.0` is installable from both registries.
- [ ] The README describes what the provider supports, not "early development".
