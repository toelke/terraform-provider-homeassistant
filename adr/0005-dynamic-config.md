# 5. Automations, scripts, scenes, and dashboards take a dynamic `config`

Date: 2026-10-03 · Status: Accepted

## Context

The original spec proposed a typed HCL DSL for triggers, conditions, and actions, plus typed card
blocks for dashboards, each with `raw` YAML escape hatches. This is the largest and most fragile
part of that spec:

- HA has dozens of trigger, condition, action, and card types, and recursive structures
  (`choose`, `if`, `repeat`, `parallel`, stacks).
- Drift detection would need a reverse parser.
- The `raw` blocks were declared as accepting drift.
- New HA features would need provider releases.

## Decision

- `homeassistant_automation`, `homeassistant_script`, and `homeassistant_scene` have a required
  `id` and **one** dynamic attribute `config`, holding the object exactly as HA stores it (alias,
  description, mode, triggers, conditions, actions, …).
- `homeassistant_dashboard` has typed metadata attributes and one dynamic `config` (the whole
  Lovelace config).
- Users write `config` as native HCL objects or as `yamldecode(file("x.yaml"))`. YAML copied from
  the HA UI works unchanged.
- Validation of `config` contents is left to HA. HA's error message is surfaced as a diagnostic.
- Typed ergonomics may be added later as **builder functions** (ADR-0007). They are optional and
  never required.

## Consequences

- Every HA feature is usable from day one, and there is no reverse parser.
- Drift detection is exact, given semantic equality (ADR-0006).
- Typos in keys are found at apply time (by HA), not at plan time.
