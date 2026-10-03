---
status: done
depends_on: [001]
adrs: [0017]
spec: []
---

# Tooling: pre-commit, lint, docs generation, CI

## Scope
- `.pre-commit-config.yaml`: gofumpt, `go mod tidy` with a diff check, golangci-lint,
  `go test -short ./...`, and a tfplugindocs check.
- `tfplugindocs` setup with `examples/` and `templates/`. `docs/` is generated.
- GitHub Actions: lint, unit tests, and a docs-freshness check on every PR.
- Dependabot or Renovate for Go modules and actions.

## Acceptance criteria
- [x] `pre-commit run --all-files` passes on a clean checkout.
- [x] CI fails when `docs/` is stale or `go.mod` is untidy.
