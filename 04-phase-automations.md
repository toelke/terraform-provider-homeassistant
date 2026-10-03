# Phase 3 — Automations, Scripts, and Helpers

**Goal:** Manage HA's logic layer declaratively. Automations respond to events, scripts
define reusable action sequences, and helpers provide stateful variables that tie them
together. Together these are the "business logic" of a smart home.

**Transport:** REST only, via the undocumented but stable
`/api/config/<domain>/config/<id>` endpoints. These are the same endpoints the HA
frontend uses when you save an automation in the UI editor.

---

## ID Convention

For all resources in this phase, **you supply the ID** as part of the resource
configuration. This is the path segment used in the REST URL and stored as the TF
resource ID. Choose short, stable, lowercase slugs:

```hcl
resource "homeassistant_automation" "bedroom_lights_on_motion" {
  id = "bedroom_lights_on_motion"   # → POST /api/config/automation/config/bedroom_lights_on_motion
  # ...
}
```

If `id` is omitted, the provider generates a deterministic ID from the resource address
(e.g. SHA1 of `module.path/resource_type.resource_name`), ensuring stable IDs across
`tofu apply` runs without user management.

---

## REST Operation Pattern

```
Create/Update:  POST /api/config/automation/config/<id>   body: full object (incl. id)
Read:           GET  /api/config/automation/config/<id>
Delete:         DELETE /api/config/automation/config/<id>
Reload:         POST /api/services/automation/reload      (call after any write)
List (ids):     GET  /api/states  → filter domain "automation.*", read attributes.id
```

The same pattern applies for `script`, `scene`, `input_boolean`, `input_number`,
`input_text`, `input_select`, `input_datetime`, `input_button`.

---

## `homeassistant_automation`

Automations are the most complex resource in this phase — they have three sections
(`trigger`, `condition`, `action`) each with many sub-types. The resource exposes a
**typed DSL** for each, with a `raw` escape hatch that accepts a YAML/JSON string
merged in verbatim.

### Top-level schema

```hcl
resource "homeassistant_automation" "example" {
  id          = "my_automation"
  alias       = "Human readable name"
  description = "Optional longer description"
  mode        = "single"   # single | restart | queued | parallel
  max         = 10         # only for queued/parallel mode

  # trigger blocks (one or more)
  # condition blocks (optional)
  # action blocks (one or more)
}
```

**Arguments:**

| Name | Type | Default | Description |
|---|---|---|---|
| `id` | string | auto | REST path ID |
| `alias` | string | yes | Display name in the UI |
| `description` | string | no | |
| `mode` | string | `"single"` | `single`, `restart`, `queued`, `parallel` |
| `max` | number | `10` | Max concurrent runs (queued/parallel) |
| `enabled` | bool | `true` | Whether the automation is enabled |

---

### Trigger blocks

```hcl
# State trigger
trigger {
  platform = "state"
  entity_id = "binary_sensor.motion_bedroom"
  from      = "off"
  to        = "on"
  for       = "00:00:05"  # optional: must be in this state for duration
}

# Time trigger
trigger {
  platform = "time"
  at       = "07:00:00"
}

# Sun trigger
trigger {
  platform = "sun"
  event    = "sunset"
  offset   = "-00:30:00"
}

# Numeric state trigger
trigger {
  platform    = "numeric_state"
  entity_id   = "sensor.bedroom_temperature"
  above       = 25
  below       = null
}

# Time pattern trigger
trigger {
  platform = "time_pattern"
  minutes  = "/15"   # every 15 minutes
}

# MQTT trigger
trigger {
  platform = "mqtt"
  topic    = "home/doorbell"
  payload  = "pressed"
}

# Webhook trigger
trigger {
  platform    = "webhook"
  webhook_id  = "my-secret-webhook-id"
  allowed_methods = ["POST"]
}

# Template trigger
trigger {
  platform = "template"
  value_template = "{{ states('sensor.power') | float > 3000 }}"
  for            = "00:05:00"
}

# Device trigger (typed, for known device triggers)
trigger {
  platform   = "device"
  device_id  = homeassistant_esphome.bedroom_sensor.device_id
  domain     = "binary_sensor"
  entity_id  = "binary_sensor.bedroom_motion"
  type       = "turned_on"
}

# Escape hatch
trigger {
  raw = <<-YAML
    platform: homeassistant
    event: start
  YAML
}
```

---

### Condition blocks

```hcl
# State condition
condition {
  type      = "state"
  entity_id = "input_boolean.guest_mode"
  state     = "off"
}

# Numeric state condition
condition {
  type      = "numeric_state"
  entity_id = "sensor.outdoor_temperature"
  above     = 5
}

# Time condition
condition {
  type   = "time"
  after  = "07:00:00"
  before = "23:00:00"
  weekday = ["mon", "tue", "wed", "thu", "fri"]
}

# Template condition
condition {
  type              = "template"
  value_template    = "{{ is_state('person.alice', 'home') }}"
}

# AND / OR / NOT nesting
condition {
  type = "or"
  conditions = [
    { type = "state", entity_id = "person.alice", state = "home" },
    { type = "state", entity_id = "person.bob",   state = "home" },
  ]
}

# Escape hatch
condition {
  raw = <<-YAML
    condition: zone
    entity_id: device_tracker.phone
    zone: zone.work
  YAML
}
```

---

### Action blocks

```hcl
# Call a service
action {
  type    = "service"
  service = "light.turn_on"
  target = {
    entity_id = ["light.bedroom_ceiling", "light.bedside"]
  }
  data = {
    brightness_pct = 80
    color_temp_kelvin = 3000
  }
}

# Turn on via area
action {
  type    = "service"
  service = "light.turn_on"
  target = {
    area_id = homeassistant_area.first["bedroom"].id
  }
}

# Delay
action {
  type  = "delay"
  delay = "00:00:30"
}

# Wait for state
action {
  type          = "wait_for_trigger"
  timeout       = "00:05:00"
  continue_on_timeout = false
  trigger {
    platform  = "state"
    entity_id = "binary_sensor.motion_bedroom"
    to        = "off"
  }
}

# Conditional action
action {
  type = "choose"
  choose {
    conditions = [{ type = "state", entity_id = "sun.sun", state = "below_horizon" }]
    sequence {
      action {
        type    = "service"
        service = "light.turn_on"
        target  = { entity_id = "light.bedroom_ceiling" }
        data    = { brightness_pct = 10 }
      }
    }
  }
  default {
    action {
      type    = "service"
      service = "light.turn_on"
      target  = { entity_id = "light.bedroom_ceiling" }
      data    = { brightness_pct = 80 }
    }
  }
}

# Call a script
action {
  type   = "service"
  service = "script.good_morning"
}

# Notify
action {
  type    = "service"
  service = "notify.mobile_app_alice_phone"
  data = {
    title   = "Motion detected"
    message = "Someone is in the bedroom"
  }
}

# Escape hatch: verbatim YAML merged into the action sequence
action {
  raw = <<-YAML
    - service: tts.google_translate_say
      data:
        entity_id: media_player.living_room
        message: "Good morning!"
  YAML
}
```

---

### Full automation example

```hcl
resource "homeassistant_automation" "bedroom_morning_routine" {
  id          = "bedroom_morning_routine"
  alias       = "Bedroom — Morning Routine"
  description = "Gradually wake up the bedroom at the scheduled time"
  mode        = "single"

  trigger {
    platform = "time"
    at       = "{{ states('input_datetime.wake_time') }}"
  }

  condition {
    type      = "state"
    entity_id = homeassistant_input_boolean.guest_mode.entity_id
    state     = "off"
  }

  condition {
    type    = "time"
    weekday = ["mon", "tue", "wed", "thu", "fri"]
  }

  action {
    type    = "service"
    service = "light.turn_on"
    target  = { entity_id = data.homeassistant_entities.bedroom_lights.entity_ids }
    data    = { brightness_pct = 1, color_temp_kelvin = 2700 }
  }

  action {
    type  = "delay"
    delay = "00:10:00"
  }

  action {
    type    = "service"
    service = "light.turn_on"
    target  = { entity_id = data.homeassistant_entities.bedroom_lights.entity_ids }
    data    = { brightness_pct = 80, color_temp_kelvin = 4000, transition = 600 }
  }
}
```

---

## `homeassistant_script`

Scripts are reusable action sequences. They use the same action blocks as automations
but have no triggers or conditions — they're called by automations or via the UI.

```hcl
resource "homeassistant_script" "good_morning" {
  id    = "good_morning"
  alias = "Good Morning"
  mode  = "single"

  action {
    type    = "service"
    service = "light.turn_on"
    target  = { area_id = homeassistant_area.first["bedroom"].id }
    data    = { brightness_pct = 30, color_temp_kelvin = 2700 }
  }

  action {
    type  = "delay"
    delay = "00:05:00"
  }

  action {
    type    = "service"
    service = "media_player.play_media"
    target  = { entity_id = "media_player.bedroom_speaker" }
    data = {
      media_content_id   = "Good morning playlist"
      media_content_type = "playlist"
    }
  }
}
```

---

## `homeassistant_scene`

Scenes capture a snapshot of entity states to be restored on demand.

```hcl
resource "homeassistant_scene" "movie_mode" {
  id    = "movie_mode"
  alias = "Movie Mode"

  entity {
    entity_id = "light.living_room_main"
    state     = "on"
    attributes = {
      brightness    = 30
      color_temp_kelvin = 2700
    }
  }

  entity {
    entity_id = "switch.projector"
    state     = "on"
  }

  entity {
    entity_id = "cover.living_room_blind"
    state     = "closed"
  }
}
```

---

## Helper Resources

Helpers are stateful variables that automations and scripts can read and write. All
helpers follow the same REST pattern: `POST /api/config/<domain>/config/<id>`.

### `homeassistant_input_boolean`

```hcl
resource "homeassistant_input_boolean" "guest_mode" {
  id      = "guest_mode"
  name    = "Guest Mode"
  icon    = "mdi:account-group"
  initial = false
}

# Reference in automation:
# entity_id = homeassistant_input_boolean.guest_mode.entity_id
# → "input_boolean.guest_mode"
```

**Arguments:** `id`, `name`, `icon`, `initial` (bool).
**Computed:** `entity_id` → `"input_boolean.<id>"`.

---

### `homeassistant_input_number`

```hcl
resource "homeassistant_input_number" "target_temperature" {
  id   = "target_temperature"
  name = "Target Temperature"
  min  = 15
  max  = 30
  step = 0.5
  unit = "°C"
  mode = "slider"   # slider | box
  icon = "mdi:thermometer"
  initial = 21
}
```

**Arguments:** `id`, `name`, `min`, `max`, `step`, `unit`, `mode`, `icon`, `initial`.

---

### `homeassistant_input_text`

```hcl
resource "homeassistant_input_text" "alarm_code" {
  id        = "alarm_code"
  name      = "Alarm Code"
  min       = 4
  max       = 6
  pattern   = "[0-9]+"
  mode      = "password"   # text | password
  icon      = "mdi:lock"
  initial   = ""
}
```

**Arguments:** `id`, `name`, `min`, `max`, `pattern`, `mode`, `icon`, `initial`.

---

### `homeassistant_input_select`

```hcl
resource "homeassistant_input_select" "house_mode" {
  id      = "house_mode"
  name    = "House Mode"
  options = ["Home", "Away", "Guest", "Vacation", "Night"]
  initial = "Home"
  icon    = "mdi:home-variant"
}
```

**Arguments:** `id`, `name`, `options` (list), `initial`, `icon`.

---

### `homeassistant_input_datetime`

```hcl
resource "homeassistant_input_datetime" "wake_time" {
  id        = "wake_time"
  name      = "Wake Time"
  has_date  = false
  has_time  = true
  initial   = "07:00:00"
  icon      = "mdi:alarm"
}
```

**Arguments:** `id`, `name`, `has_date`, `has_time`, `initial`, `icon`.

---

### `homeassistant_input_button`

```hcl
resource "homeassistant_input_button" "doorbell" {
  id   = "doorbell"
  name = "Doorbell"
  icon = "mdi:doorbell"
}
```

**Arguments:** `id`, `name`, `icon`.

---

## Full Phase 3 Example

```hcl
# ── Helpers ──────────────────────────────────────────────────────────────────

resource "homeassistant_input_boolean" "guest_mode" {
  id      = "guest_mode"
  name    = "Guest Mode"
  icon    = "mdi:account-group"
  initial = false
}

resource "homeassistant_input_select" "house_mode" {
  id      = "house_mode"
  name    = "House Mode"
  options = ["Home", "Away", "Night", "Guest", "Vacation"]
  initial = "Home"
}

resource "homeassistant_input_datetime" "wake_time" {
  id       = "wake_time"
  name     = "Wake Time"
  has_time = true
  initial  = "07:00:00"
}

# ── Scripts ───────────────────────────────────────────────────────────────────

resource "homeassistant_script" "good_morning" {
  id    = "good_morning"
  alias = "Good Morning"

  action {
    type    = "service"
    service = "input_select.select_option"
    target  = { entity_id = homeassistant_input_select.house_mode.entity_id }
    data    = { option = "Home" }
  }

  action {
    type    = "service"
    service = "light.turn_on"
    target  = { area_id = homeassistant_area.first["bedroom"].id }
    data    = { brightness_pct = 50, color_temp_kelvin = 3000, transition = 30 }
  }
}

resource "homeassistant_script" "goodnight" {
  id    = "goodnight"
  alias = "Goodnight"

  action {
    type    = "service"
    service = "input_select.select_option"
    target  = { entity_id = homeassistant_input_select.house_mode.entity_id }
    data    = { option = "Night" }
  }

  action {
    type    = "service"
    service = "light.turn_off"
    target  = { area_id = "all" }
  }

  action {
    type    = "service"
    service = "lock.lock"
    target  = { label_id = homeassistant_label.managed.id }
  }
}

# ── Automations ───────────────────────────────────────────────────────────────

resource "homeassistant_automation" "wake_time_routine" {
  id    = "wake_time_routine"
  alias = "Wake Time — Morning Routine"
  mode  = "single"

  trigger {
    platform = "time"
    at       = "{{ states('${homeassistant_input_datetime.wake_time.entity_id}') }}"
  }

  condition {
    type      = "state"
    entity_id = homeassistant_input_select.house_mode.entity_id
    state     = "Home"
  }

  condition {
    type      = "state"
    entity_id = homeassistant_input_boolean.guest_mode.entity_id
    state     = "off"
  }

  action {
    type    = "service"
    service = "script.turn_on"
    target  = { entity_id = homeassistant_script.good_morning.entity_id }
  }
}

resource "homeassistant_automation" "motion_bedroom_lights" {
  id    = "motion_bedroom_lights"
  alias = "Bedroom — Lights on Motion"
  mode  = "restart"

  trigger {
    platform  = "state"
    entity_id = "binary_sensor.bedroom_motion"
    to        = "on"
  }

  condition {
    type   = "time"
    after  = "22:00:00"
    before = "08:00:00"
  }

  action {
    type    = "service"
    service = "light.turn_on"
    target  = { area_id = homeassistant_area.first["bedroom"].id }
    data    = { brightness_pct = 20, color_temp_kelvin = 2200 }
  }

  action {
    type  = "delay"
    delay = "00:05:00"
  }

  action {
    type = "wait_for_trigger"
    timeout = "00:30:00"
    continue_on_timeout = true
    trigger {
      platform  = "state"
      entity_id = "binary_sensor.bedroom_motion"
      to        = "off"
    }
  }

  action {
    type    = "service"
    service = "light.turn_off"
    target  = { area_id = homeassistant_area.first["bedroom"].id }
    data    = { transition = 5 }
  }
}

resource "homeassistant_automation" "away_mode" {
  id    = "away_mode"
  alias = "House Mode — Away"
  mode  = "single"

  trigger {
    platform  = "state"
    entity_id = homeassistant_input_select.house_mode.entity_id
    to        = "Away"
  }

  action {
    type    = "service"
    service = "light.turn_off"
    target  = { area_id = "all" }
  }

  action {
    type    = "service"
    service = "climate.set_hvac_mode"
    target  = { area_id = "all" }
    data    = { hvac_mode = "away" }
  }

  action {
    type    = "service"
    service = "notify.mobile_app_alice_phone"
    data    = { message = "Away mode activated. All lights off." }
  }
}
```

---

## YAML Serialisation

When the provider sends automation data to HA, it must serialise the typed blocks to
the exact JSON/YAML schema HA expects. Each trigger/condition/action block type maps to
a specific JSON object.

```go
// internal/resources/automation_types.go

type Trigger struct {
    Platform string `json:"platform"`
    EntityID string `json:"entity_id,omitempty"`
    From     string `json:"from,omitempty"`
    To       string `json:"to,omitempty"`
    For      string `json:"for,omitempty"`
    At       string `json:"at,omitempty"`
    // ... all trigger-type fields
    Raw      string `json:"-"` // handled separately: unmarshalled and merged
}

// SerializeTrigger converts a Trigger model to the map[string]interface{}
// that HA expects. Raw YAML is parsed and merged last (overrides typed fields).
func SerializeTrigger(t Trigger) (map[string]interface{}, error) { ... }
```

The `raw` field across all block types should be parsed as YAML and deep-merged into
the serialised output of the typed fields. This lets users override any field not
exposed by the typed schema without breaking everything else.

---

## Implementation Checklist

- [ ] `RESTClient.PostConfig(domain, id, body)` — POST and reload
- [ ] `RESTClient.GetConfig(domain, id)` — GET, return nil on 404
- [ ] `RESTClient.DeleteConfig(domain, id)` — DELETE and reload
- [ ] `RESTClient.Reload(domain)` — POST /api/services/<domain>/reload
- [ ] Trigger serialiser: state, time, sun, numeric_state, time_pattern, mqtt,
      webhook, template, device, raw
- [ ] Condition serialiser: state, numeric_state, time, template, and/or/not, raw
- [ ] Action serialiser: service, delay, wait_for_trigger, choose (with conditions +
      sequences), raw
- [ ] `homeassistant_automation`: full CRUD, computed `entity_id`
- [ ] `homeassistant_script`: full CRUD, computed `entity_id`
- [ ] `homeassistant_scene`: full CRUD, computed `entity_id`
- [ ] All `homeassistant_input_*`: full CRUD, computed `entity_id`
- [ ] Validate `mode` values at plan time, not apply time
- [ ] After every write, call the reload endpoint so HA picks up the new config without
      a restart
- [ ] On Read after import: deserialise HA's JSON back into the typed model (reverse
      of serialisation — needed for drift detection)
- [ ] Raw block round-trip: store raw YAML as-is in state; on Read, don't try to parse
      and re-serialize it (accept drift for raw blocks)
