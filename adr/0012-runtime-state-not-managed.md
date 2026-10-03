# 12. Runtime state, including automation enabled/disabled, is not managed

Date: 2026-10-03 · Status: Accepted

## Context

Whether an automation is enabled is entity state (`automation.turn_on`/`turn_off`), and it is
restored across restarts. The config has only `initial_state`, which forces the value on every
start or reload. The original spec had an `enabled` attribute.

## Decision

The provider has no `enabled` attribute and never calls services to change runtime state. Users
who want an automation disabled as code put `initial_state: false` into its dynamic `config`. The
docs show this.

## Consequences

- Toggling an automation in the UI never shows up as Tofu drift.
- `initial_state` overrides UI toggles on every restart or reload, by HA design.
