---
status: done
depends_on: [022, 031]
adrs: [0023, 0006]
spec: [dashboards.md, overview.md]
---

# Stored baseline for dashboards

ADR-0023 applies to every dynamic `config`. Reuse the baseline helper from ticket 031.

## Scope
- After create, and after an update that saves the config, read it back with `lovelace/config`
  and store the baseline hash in private state.
- Read: equal to the baseline keeps the prior `config`; otherwise semantic equality.
- `CHANGELOG.md` entry under `Unreleased` only if a user-visible diff disappears. Otherwise none.

## Acceptance criteria
- [x] The existing dashboard acceptance tests still have empty second plans.
- [x] Drift is still detected: a config saved through `lovelace/config/save` between steps shows
      as a diff.
- [x] Unit test: a state without a baseline falls back to semantic equality.
- [x] The PR body contains "Closes #36" only if ticket 031 is already merged; otherwise "Part of
      #36".
