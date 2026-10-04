# Input helpers

These are typed resources (ADR-0013) over the WebSocket collection API:
`<domain>/list|create|update|delete`. Update and delete take `<domain>_id`. IDs are generated
from `name` (ADR-0009, including the optional `id` via create-then-rename and the collision
check).

Common arguments: `name` (required), `id` (optional, computed), `icon` (optional).
Common computed: `entity_id` = `<domain>.<id>`.
Update sends the whole helper, because HA replaces every field on `<domain>/update`. An optional
attribute removed from config is therefore cleared in HA. Attributes that HA fills with a default
have the same default in the schema, so removing them resets them to it: `input_number` `step`
(`1`) and `mode` (`slider`); `input_text` `min` (`0`), `max` (`100`) and `mode` (`text`);
`input_datetime` `has_date` and `has_time` (`false`). The plan-time checks apply these defaults to
unset attributes too.
Read: `<domain>/list`, find by ID; if missing → remove from state. Import: by ID.

| Resource | Extra arguments | Plan-time validation |
|---|---|---|
| `homeassistant_input_boolean` | `initial` (bool) | — |
| `homeassistant_input_number` | `min`, `max` (required), `step`, `initial`, `mode` (`slider`/`box`), `unit_of_measurement` | `min < max`; `initial` within the range; `step > 0` |
| `homeassistant_input_text` | `min`, `max`, `initial`, `pattern`, `mode` (`text`/`password`) | `0 ≤ min ≤ max ≤ 255`; `initial` is `min` to `max` characters long; `pattern` compiles |
| `homeassistant_input_select` | `options` (list, required), `initial` | options non-empty and unique; `initial ∈ options` |
| `homeassistant_input_datetime` | `has_date`, `has_time`, `initial` | at least one of `has_date`/`has_time`; `initial` format matches |
| `homeassistant_input_button` | — | — |

All six share one generic implementation, parameterised by domain and a field mapping.

`pattern` is a Python regular expression, because HA matches it with `re`. Go's `regexp` (RE2)
rejects valid Python patterns such as lookarounds, so the check compiles it with `regexp2`
instead, after rewriting `(?P=name)` to `\k<name>`. The `input_datetime` `initial` format is
`YYYY-MM-DD` with only a date, `HH:MM[:SS]` with only a time, and `YYYY-MM-DD HH:MM[:SS]` (or
with `T` as the separator) with both.
