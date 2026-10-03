# 2. Use terraform-plugin-framework

Date: 2026-10-03 · Status: Accepted

## Context

The original spec mixed SDKv2 idioms (`d.SetId("")`) with plugin-framework ones
(`ImportStatePassthroughID`, `ModifyPlan`). Several later decisions need features that only the
framework has: dynamic attributes (ADR-0005), custom types with semantic equality (ADR-0006), and
provider-defined functions (ADR-0007).

## Decision

Use `github.com/hashicorp/terraform-plugin-framework` exclusively, with protocol v6. Do not use
SDKv2 or the mux server.

## Consequences

- A resource that has disappeared is reported with `resp.State.RemoveResource(ctx)`, not
  `SetId("")`.
- The minimum CLI version is set by protocol v6 plus function support (see ADR-0016).
