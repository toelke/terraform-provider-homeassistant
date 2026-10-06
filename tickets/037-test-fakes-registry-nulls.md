---
status: done
depends_on: [009, 010, 011]
issues: [44]
adrs: [0017, 0009]
spec: [registries.md]
---

# Shared test fakes, `acctest.Eventually`, one create rule for registries

From issue #44: finding 22, and the consistency part of finding 14. Finding 21 was done by
tickets 033 and 034.

## Scope
- `client`: one scripted fake, `fakeWS` (command type → handler), and `fakeStore`, an in-memory
  registry or helper collection built on it. They replace the per-file fakes of the area, floor,
  device, label, entity, and helper tests.
- `acctest.Eventually(t, timeout, what, cond)` replaces the hand-written polling loops in the
  acceptance tests.
- `resources/helper_test.go` has one schema-type helper, not two.
- Floor, area, and label create leave unset attributes out (`withoutNulls`). Only area did
  before; floor and label sent `null`.
- No user-visible change, so no `CHANGELOG.md` entry.

## Acceptance criteria
- [x] No acceptance test has its own polling loop.
- [x] `go test -short ./...` and the acceptance tests are green.
- [x] `spec/registries.md` says create leaves unset attributes out.

## Notes
- The sleep in `TestWSContextCancel` (finding 22) was already replaced with a channel.
- `client/dashboard_test.go` keeps `fakeCommander`: its tests check the order of commands, and
  `fakeWS` answers by type only.
- The generic `registryResource` from finding 14 is not done. The issue makes it conditional on
  more registry-backed resources being planned, and none are.
