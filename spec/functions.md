# Builder functions (later)

Provider-defined functions (ADR-0007), requiring OpenTofu or Terraform ≥ 1.8. They are pure and
typed, and each returns a plain object that can be placed anywhere inside a dynamic `config`.
They are always optional: everything they build can also be written by hand.

Candidates, prioritised by how much they reduce verbosity:

- **Triggers:** `state_trigger(entity_id, to, from, for)`, `time_trigger(at)`,
  `numeric_state_trigger(...)`
- **Conditions:** `state_condition(...)`, `time_condition(...)`
- **Actions:** `action(name, target, data)`, `delay(duration)`
- **Cards:** `tile_card(entity, ...)`, `entities_card(entities, ...)`, `heading_card(text)`,
  `grid_section(cards, ...)`

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

## Cards (shipped)

| Function | Result |
|---|---|
| `tile_card(entity, options...)` | `{ type = "tile", entity = entity, ... }` |
| `entities_card(entities, options...)` | `{ type = "entities", entities = entities, ... }`; each row is an entity ID or an object |
| `heading_card(heading, options...)` | `{ type = "heading", heading = heading, ... }` |
| `grid_section(cards, options...)` | `{ type = "grid", cards = [...], ... }`; `options.heading`, if set, becomes a heading card in front of `cards` |

Unit tests compare each function's output with a golden JSON snapshot. Each function has an
example in `examples/functions/`.
