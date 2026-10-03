---
status: todo
depends_on: [001]
adrs: [0005, 0006]
spec: [overview.md]
---

# Dynamic-config custom type with semantic equality

## Scope
`internal/dyntype`: a custom framework type wrapping a dynamic value. It provides JSON
conversion, normalisation, and `SemanticEquals`, so the prior value is kept when the read-back
value means the same thing.

## Acceptance criteria
- [ ] Table-driven unit tests: key order, `5` vs `5.0`, nested lists and objects, and
      null-versus-absent where HA is known to drop it.
- [ ] A real change is not considered equal.
