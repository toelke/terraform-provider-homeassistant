# 16. Support OpenTofu and Terraform ≥ 1.8; OpenTofu first

Date: 2026-10-03 · Status: Accepted

## Context

OpenTofu and Terraform speak the same plugin protocol. Provider-defined functions (ADR-0007)
require version 1.8 in both.

## Decision

- Minimum version: OpenTofu 1.8 or Terraform 1.8.
- Docs and examples use `tofu`. Acceptance tests run with the OpenTofu binary.
- Publish to both the OpenTofu and Terraform registries.

## Consequences

Terraform-only behaviour differences are found by users, not by CI. This is accepted.
