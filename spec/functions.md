# Builder functions

Provider-defined functions (ADR-0007), requiring OpenTofu or Terraform ≥ 1.8. They are pure and
typed, and each returns a plain object that can be placed anywhere inside a dynamic `config`.
They are always optional: everything they build can also be written by hand.

They cover two areas, each listed below: triggers, conditions, and actions for automation and
script `config`, and cards and sections for dashboard `config`.

## Signature convention

Per ADR-0020, required arguments are positional, and nothing else is. Every optional key goes in
one optional trailing `options` object (a variadic parameter of at most one element), which is
merged into the result: keys with a null value are left out, and setting a key that a positional
argument already sets is an error. No positional argument is nullable, and no sentinel value
stands for "absent": a value that can be absent goes in `options`.

```hcl
provider::homeassistant::tile_card("light.kitchen")                     # { type = "tile", entity = "light.kitchen" }
provider::homeassistant::tile_card("light.kitchen", { name = "Kitchen" })
```

## Triggers, conditions, actions

Results use the current HA syntax (`trigger`, `condition`, `action` keys), which every supported
HA version accepts.

| Function | Result |
|---|---|
| `state_trigger(entity_id, options...)` | `{ trigger = "state", entity_id = entity_id, ... }` |
| `numeric_state_trigger(entity_id, options...)` | `{ trigger = "numeric_state", entity_id = entity_id, ... }`; `options` must set `above` or `below` |
| `time_trigger(at, options...)` | `{ trigger = "time", at = at, ... }`; `at` is a string, an object, or a list of these |
| `state_condition(entity_id, state, options...)` | `{ condition = "state", entity_id = entity_id, state = state, ... }` |
| `time_condition(options...)` | `{ condition = "time", ... }`; `options` must set `after`, `before`, or `weekday` |
| `action(name, options...)` | `{ action = name, ... }`; `target` and `data` go in `options` |
| `delay(duration, options...)` | `{ delay = duration, ... }`; `duration` is a string, a number of seconds, or an object |

`entity_id` and `state` take a string or a non-empty list of strings. The two "must set" rules
are checked at plan time because HA rejects such a trigger or condition anyway; other keys in
`options` are left to HA (ADR-0020).

`for` is a key of several triggers and conditions, but HCL reads `{ for` as the start of a `for`
expression. It is written as a quoted key, `{ "for" = "00:05:00" }`; the docs say so.

## Cards

| Function | Result |
|---|---|
| `tile_card(entity, options...)` | `{ type = "tile", entity = entity, ... }` |
| `entities_card(entities, options...)` | `{ type = "entities", entities = entities, ... }`; each row is an entity ID or an object |
| `heading_card(heading, options...)` | `{ type = "heading", heading = heading, ... }` |
| `grid_section(cards, options...)` | `{ type = "grid", cards = [...], ... }`; `options.heading`, if set, becomes a heading card in front of `cards` |

Unit tests compare each function's output with a golden JSON snapshot. Each function has an
example in `examples/functions/`.
