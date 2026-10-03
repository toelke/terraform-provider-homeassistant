# Input helpers

These are typed resources (ADR-0013) over the WebSocket collection API:
`<domain>/list|create|update|delete`. Update and delete take `<domain>_id`. IDs are generated
from `name` (ADR-0009, including the optional `id` via create-then-rename and the collision
check).

Common arguments: `name` (required), `id` (optional, computed), `icon` (optional).
Common computed: `entity_id` = `<domain>.<id>`.
Read: `<domain>/list`, find by ID; if missing → remove from state. Import: by ID.

| Resource | Extra arguments | Plan-time validation |
|---|---|---|
| `homeassistant_input_boolean` | `initial` (bool) | — |
| `homeassistant_input_number` | `min`, `max` (required), `step`, `initial`, `mode` (`slider`/`box`), `unit_of_measurement` | `min < max`; `initial` within the range; `step > 0` |
| `homeassistant_input_text` | `min`, `max`, `initial`, `pattern`, `mode` (`text`/`password`) | `0 ≤ min ≤ max ≤ 255`; `pattern` compiles |
| `homeassistant_input_select` | `options` (list, required), `initial` | options non-empty and unique; `initial ∈ options` |
| `homeassistant_input_datetime` | `has_date`, `has_time`, `initial` | at least one of `has_date`/`has_time`; `initial` format matches |
| `homeassistant_input_button` | — | — |

All six share one generic implementation, parameterised by domain and a field mapping.
