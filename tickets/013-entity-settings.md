---
status: todo
depends_on: [011]
adrs: [0010]
spec: [registries.md]
---

# homeassistant_entity_settings

## Acceptance criteria
- [ ] Field-granular: an attribute changed in the UI that is *not* in config causes no diff
      (acceptance test changes it via WS between steps).
- [ ] Create waits for the entity to appear (unit test with a fake that appears after N polls)
      and times out with a helpful error.
- [ ] Destroy resets only the configured fields. The entity still exists afterwards.
- [ ] `hidden` and `disabled` map to and from `"user"` and `null`.
- [ ] Import by `entity_id`.
