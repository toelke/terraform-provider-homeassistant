---
status: todo
depends_on: [002, 005]
adrs: [0018]
spec: []
---

# Release pipeline and first pre-release

## Scope
- GoReleaser config (registry manifest, GPG-signed `SHA256SUMS`), and a release workflow on `v*`
  tags. The GPG key is stored as a repo secret.
- Register the provider on the OpenTofu registry and the Terraform registry.
- Start `CHANGELOG.md`.
- Tag `v0.0.1`, containing only `data.homeassistant_config`.

From here on, any merged ticket may be released as a `v0.0.x` pre-release.

## Acceptance criteria
- [ ] `v0.0.1` is installable as `toelke/homeassistant` from both registries, and
      `tofu init` verifies the signature.
- [ ] The README states the pre-1.0 stability policy.
