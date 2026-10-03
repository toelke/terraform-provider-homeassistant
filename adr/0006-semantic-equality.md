# 6. Dynamic config uses a custom type with semantic equality

Date: 2026-10-03 · Status: Accepted

## Context

HA stores automation, script, and scene config as YAML and returns it as JSON. Lovelace config is
returned as JSON. A value read back can differ from the value sent without meaning anything
different: number representation (`5` vs `5.0`), key order, and possibly omitted empty
collections. Comparing the values as raw text would produce a diff on every plan.

## Decision

Implement a custom framework type for dynamic config that implements semantic equality:

- Both sides are converted to JSON, normalised (sorted keys, canonical numbers), and compared.
- If the value read from HA is semantically equal to the prior state, the prior state value is
  kept, so the user's exact HCL value survives and the plan is empty.
- Normalisation rules live in one place, with a table-driven unit test per known HA quirk.

## Consequences

- Each newly discovered HA normalisation quirk becomes a test case plus a rule.
- A real change made in the HA UI still shows as drift.
