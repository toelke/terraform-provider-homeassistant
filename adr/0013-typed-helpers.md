# 13. Input helpers are typed resources

Date: 2026-10-03 · Status: Accepted

## Context

Unlike automations, the helpers (`input_boolean`, `input_number`, `input_text`, `input_select`,
`input_datetime`, `input_button`) each have a small, stable schema. They are managed over the
WebSocket collection API, and their IDs are generated (ADR-0009).

## Decision

- Each helper is its own resource with typed attributes and plan-time validation, e.g.
  `min < max`, `initial` must be one of `options`, at least one of `has_date`/`has_time`.
- All six share one generic implementation over the `<domain>/list|create|update|delete`
  commands.

## Consequences

Good plan-time errors for helpers. A new helper field means a provider change, which is
acceptable for these small schemas.
