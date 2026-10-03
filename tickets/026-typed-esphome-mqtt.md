---
status: todo
depends_on: [025]
adrs: [0014]
spec: [integrations.md]
---

# Typed homeassistant_esphome and homeassistant_mqtt

## Scope
Read the step schemas and reconfigure support from each integration's `config_flow.py` for
the oldest supported HA release, and write them into `spec/integrations.md` first.

## Acceptance criteria
- [ ] Unit tests of the step data built from the typed attributes.
- [ ] Acceptance tests are only feasible for MQTT (with a mosquitto container next to HA).
      ESPHome is covered by unit tests plus a documented manual test.
