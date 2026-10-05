---
status: done
depends_on: [016, 017, 018]
adrs: [0023, 0006]
spec: [automations.md, overview.md]
---

# Stored baseline for automations, scripts, scenes

From issue #36: old keys that HA renames on save cause a diff that apply never resolves.

## Scope
- In the shared config-item resource: after create and update, read the config back (with `id`
  stripped, as on Read), and store a hash of its normalised JSON in private state.
- Read: if the read-back hash equals the stored baseline, keep the prior `config`. Otherwise,
  fall back to semantic equality (ADR-0006).
- Without a baseline (state from v0.1.0, or after import), only semantic equality applies. The
  next apply records the baseline.
- Docs on `homeassistant_automation` and `homeassistant_script`: HA renames old keys on save, and
  the provider keeps your spelling.
- `CHANGELOG.md` entry under `Unreleased` (Fixed), referencing #36.

## Acceptance criteria
- [x] Acceptance test: the reproduction from #36 (top-level `trigger`/`condition`/`action`, and
      `service` in a nested step such as `choose[].sequence`). The plan after apply is empty.
- [x] Acceptance test: the same for a script with `service` in its `sequence`.
- [x] Drift is still detected: a config changed through the REST editor between steps shows as a
      diff.
- [x] Unit test: a state without a baseline falls back to semantic equality.
- [x] The PR body contains "Closes #36" only if ticket 032 is already merged; otherwise "Part of
      #36".
