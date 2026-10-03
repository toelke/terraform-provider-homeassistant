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
  `grid_section(cards, heading)`

Unit tests compare each function's output with a golden JSON snapshot. Each function has an
example in `examples/functions/`.
