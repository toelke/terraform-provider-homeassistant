---
status: todo
depends_on: [022, 026]
issues: [41]
adrs: [0017]
spec: [dashboards.md, integrations.md]
---

# Unit tests: dashboard rollback, typed integration update

From issue #41, finding 21: the test gaps that ticket 033 doesn't cover. The config-flow abort
under a cancelled context, the WebSocket cases, and rollbacks under a cancelled context are
already tested.

## Scope
- Dashboard: Create rolls back (deletes the dashboard) when saving its config fails, also with a
  cancelled context; Read with `ErrNoDashboardConfig` leaves `config` null.
- `typedIntegration.Update`: reconfigures only when the flow inputs differ, runs the options flow
  only when the options differ. Cover ESPHome's update path with a fake flow, like
  `TestESPHomeAnswersEveryForm`.
- Unit tests only, against the existing fakes. No user-visible change, so no CHANGELOG entry.

## Acceptance criteria
- [ ] Each case above has a unit test.
