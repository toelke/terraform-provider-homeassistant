# 20. Builder function signatures: required arguments plus one `options` object

Date: 2026-10-04 · Status: Accepted

## Context

Builder functions (ADR-0007) are public API that ticket 027 and every later function will follow,
so changing their shape later breaks users. Provider-defined functions have no optional
parameters: a function either takes every argument it declares, or ends in one variadic
parameter. The agent that built the card functions (PR #16) also reported that OpenTofu 1.11
fails a call when a top-level argument is `null`: it arrives as an empty dynamic value. The
maintainer chose among the options below on PR #16.

Options considered:

1. Required arguments positionally, plus one optional trailing `options` object.
2. Every key as a positional parameter. Every call would spell out every argument, and each new
   HA key would change the signature.
3. Required arguments only, with extras added through HCL `merge()`. `merge()` silently
   overwrites keys and keeps nulls.

## Decision

Option 1, for every builder function:

- **Required** values are positional parameters. Nothing else is positional.
- Everything **optional** goes in one trailing `options` object: a variadic parameter with at
  most one element, merged into the result.
- Keys in `options` with a `null` value are left out, so `name = var.maybe_name` works without a
  conditional.
- Setting a key in `options` that a positional argument already sets is an error at plan time.
- No positional argument is nullable, and there are no sentinel values such as `""` for
  "absent". A value that can be absent is optional, so it belongs in `options`. For example:
  `grid_section(cards)` or `grid_section(cards, { heading = "Lights" })`.

## Consequences

- Common calls stay short, and any card or trigger key HA adds later works through `options`
  without a provider release.
- The OpenTofu null-argument bug can't affect users, because no positional argument accepts null.
- Typed checks apply only to positional arguments. Keys in `options` are validated by HA when the
  config is saved.
