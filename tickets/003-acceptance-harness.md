---
status: done
depends_on: [001]
adrs: [0003, 0017]
spec: []
---

# Acceptance test harness: HA in testcontainers with headless onboarding

## Scope
- `internal/acctest`: start `ghcr.io/home-assistant/home-assistant:<tag>` with
  testcontainers-go and wait for `/api/onboarding`.
- Onboard headlessly:
  1. `POST /api/onboarding/users` (no auth) → `auth_code`.
  2. `POST /auth/token` (`grant_type=authorization_code`, same `client_id`) → access token.
  3. Finish the `core_config`, `analytics`, and `integration` onboarding steps.
  4. Mint a long-lived token via WS `auth/long_lived_access_token`.
- Share one container per `go test` package. Expose the URL and token to
  `terraform-plugin-testing`. Tests run with the OpenTofu binary (`TF_ACC_TERRAFORM_PATH`,
  `TF_ACC_PROVIDER_NAMESPACE`).
- CI job with a matrix of HA tags: the oldest and newest of the support window. Document how to
  bump them.

## Acceptance criteria
- [x] A trivial acceptance test passes locally with Docker and in CI. `homeassistant_config`
      only arrives in 005, which depends on this ticket, so the trivial test configures the
      provider from the harness's environment and checks the minted token against
      `GET /api/config`. 005 adds the read through the provider.
- [x] Without `TF_ACC`, acceptance tests are skipped.
