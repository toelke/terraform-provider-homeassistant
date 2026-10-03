# 18. Distribution: `toelke/homeassistant`, MPL-2.0, GoReleaser

Date: 2026-10-03 · Status: Accepted

## Context

The provider is developed against the maintainer's own HA, and published as soon as it is useful.
The registries require a repository named `terraform-provider-<name>`, GPG-signed checksums, and a
licence.

## Decision

- Provider source address: `toelke/homeassistant`.
- Repository: `github.com/toelke/terraform-provider-homeassistant`.
  The Go module path is the same.
- Licence: MPL-2.0.
- Releases: GoReleaser on `v*` tags, with GPG-signed `SHA256SUMS`, using the registry manifest
  format. Semver, starting at `v0.1.0`.

## Consequences

Before v1.0, breaking changes are allowed in minor versions and are noted in `CHANGELOG.md`.
