# 3. Support the six most recent Home Assistant releases

Date: 2026-10-03 · Status: Accepted

## Context

HA ships monthly and keeps adding API fields: area temperature/humidity entities, the plural
`triggers`/`actions` syntax (2024.10), and `action:` replacing `service:` (2024.8). Old syntax is
still accepted, and HA has no plans to remove it.

## Decision

- Support a rolling window of the six most recent monthly HA releases.
- CI runs acceptance tests against the oldest and the newest release in the window.
- The provider does not check HA versions at runtime and has no compatibility branches for
  releases outside the window.
- Dynamic config (ADR-0005) is passed through as written. The provider never rewrites the user's
  syntax. Docs and examples use the current syntax.

## Consequences

- Fields that exist in every release in the window are used unconditionally.
- Running against an older HA may fail with HA's own validation errors. That is acceptable, and
  the README says so.
- The CI matrix must be bumped regularly (see the tooling ticket).
