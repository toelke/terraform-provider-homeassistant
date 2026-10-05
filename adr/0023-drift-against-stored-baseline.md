# 23. Drift is measured against what Home Assistant stored after our last write

Date: 2026-10-05 · Status: Accepted · Extends ADR-0006

## Context

Issue #36: automations written with the old key names show a diff on every plan, and apply never
resolves it. HA's save handler validates the posted config, and the validation renames old keys
in place in the very dict that is then written to `automations.yaml` (checked in HA 2026.9.4):

- `cv.renamed` turns `trigger`, `condition`, and `action` at the top level into their plurals;
- `_backward_compat_service_schema` turns `service` into `action` in every action step it
  validates, including nested ones.

`platform` in triggers is not rewritten. Scripts go through the same validation, so `service` in
their `sequence` is renamed too. Semantic equality (ADR-0006) only normalises representation
(number format, key order), so the renamed config never equals the user's.

The maintainer chose among these options in the issue's grilling session:

1. Reimplement HA's renaming in the semantic-equality rules. This copies HA internals: every
   mistake, or any future HA migration, creates new phantom diffs.
2. Record what HA stored right after our own write, and compare later reads against that.
3. Only warn about old keys at plan time.

## Decision

Option 2, for every resource with a dynamic `config`: automation, script, scene, and dashboard.

- After every create and update, the provider reads the config back from HA and records a hash
  of its normalised JSON in the resource's **private state**. This is the **stored baseline**.
- On refresh, the config read from HA is compared with the baseline. If they're equal, nothing
  changed outside Tofu, so state keeps the user's `config` exactly as written. Otherwise, the
  semantic-equality rules of ADR-0006 apply as before, and a real difference is drift.
- A state without a baseline (from v0.1.0, or right after import) falls back to ADR-0006. The
  first apply records the baseline. Taking HA's current value as the baseline silently on the
  first refresh was rejected, because it would hide drift that happened before the upgrade.
- There is no plan-time warning about old keys. They are valid HA syntax, and with the baseline
  they no longer cause diffs. The docs mention that HA renames them when it saves.

## Consequences

- The provider needs no knowledge of HA's rename rules, and keeps working when HA adds new
  migrations.
- Each create or update costs one extra read.
- Users upgrading from v0.1.0 with old keys see the diff once more, until their next apply.
- A user who switches spelling, e.g. from `service` to `action`, sees one diff and one apply,
  after which the plan is empty again.
