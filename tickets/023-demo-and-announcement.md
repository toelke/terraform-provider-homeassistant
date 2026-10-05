---
status: done
depends_on: [021]
adrs: [0005, 0007]
spec: [dashboards.md, automations.md]
---

# Demo and r/homeassistant announcement

## Scope
- Demonstrates the released `v0.1.0`; this ticket tags nothing.
- `examples/demo/`: a self-contained configuration that runs against a fresh HA container (reuse
  the `internal/acctest` onboarding, or a script with the same steps). It shows:
  1. Automations from `yamldecode(file(...))`. An automation is changed in the UI, `tofu plan`
     shows the drift, and `tofu apply` restores it.
  2. One dashboard view per area, generated with `for` over `homeassistant_entities`, with a
     shared quick-actions section reused in two dashboards.
- Record the terminal part with asciinema (`docs/demo.cast`, uploaded to asciinema.org).
  Capture the resulting HA dashboard as a short screen recording or screenshots, because
  asciinema cannot show the browser.
- Add a README section embedding the cast and the screenshots.
- Draft the Reddit post (title, 3–5 sentences, links, a known-limitations list), and hand it to
  the human. **The human posts it**; the agent does not.

## Acceptance criteria
- [x] `examples/demo/` applies cleanly against a fresh HA from the support window.
- [ ] The cast, the screenshots, and the README section are merged.
- [ ] The post draft is in the PR description for the human to review.
