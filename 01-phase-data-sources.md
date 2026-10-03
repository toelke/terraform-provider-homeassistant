# Phase 0 — REST Data Sources

**Goal:** Query live HA state. No resources yet — purely read-only. This phase is the
foundation everything else builds on: automations, dashboards, and integrations all
need to look up entity IDs, area IDs, and instance metadata at plan time.

**Transport:** REST only. No WebSocket needed in this phase.

---

## Data Sources

### `data.homeassistant_entities`

Calls `GET /api/states`, then filters the response client-side. When `area` or `label`
is set, it first calls `POST /api/template` to resolve the area/label to a list of
entity IDs using HA's Jinja2 engine (which has access to the full registry), then
intersects with the state list.

```hcl
# All lights in a named area
data "homeassistant_entities" "bedroom_lights" {
  domain = "light"
  area   = "bedroom"
}

# All motion sensors regardless of area
data "homeassistant_entities" "motion_sensors" {
  domain       = "binary_sensor"
  device_class = "motion"
}

# Lights that are currently on
data "homeassistant_entities" "lights_on" {
  domain = "light"
  state  = "on"
}

# All climate entities with a label
data "homeassistant_entities" "managed_climate" {
  domain = "climate"
  label  = "managed"
}
```

**Arguments:**

| Name | Type | Description |
|---|---|---|
| `domain` | string | Entity domain prefix, e.g. `"light"` |
| `area` | string | Area name or area_id |
| `label` | string | Label name |
| `device_class` | string | e.g. `"temperature"`, `"motion"` |
| `state` | string | Match exact current state value |
| `name_pattern` | string | Glob against `friendly_name`, e.g. `"Bedroom *"` |

**Computed:**

| Name | Type | Description |
|---|---|---|
| `entity_ids` | `list(string)` | e.g. `["light.bedroom_ceiling", "light.bedside"]` |
| `entities` | `map(object(...))` | Keyed by entity_id, value has `state`, `friendly_name`, `attributes`, `last_changed` |

**Implementation notes:**

```go
// internal/datasources/entities.go

// 1. GET /api/states → []StateObject
// 2. If area set: POST /api/template {"template": "{{ area_entities('bedroom') | tojson }}"}
//    → parse JSON list, intersect with step 1
// 3. If label set: POST /api/template {"template": "{{ label_entities('managed') | tojson }}"}
// 4. Filter remaining list by domain, device_class (check attributes["device_class"]),
//    state, name_pattern (glob match on attributes["friendly_name"])
// 5. Write entity_ids and entities to state

type StateObject struct {
    EntityID    string                 `json:"entity_id"`
    State       string                 `json:"state"`
    Attributes  map[string]interface{} `json:"attributes"`
    LastChanged string                 `json:"last_changed"`
    LastUpdated string                 `json:"last_updated"`
}
```

---

### `data.homeassistant_entity`

Calls `GET /api/states/<entity_id>`. Returns 404 if missing — surface as an error with
a suggestion to check the entity_id.

```hcl
data "homeassistant_entity" "outdoor_temp" {
  entity_id = "sensor.outdoor_temperature"
}

output "current_temp" {
  value = "${data.homeassistant_entity.outdoor_temp.state} ${data.homeassistant_entity.outdoor_temp.unit_of_measurement}"
}
```

**Arguments:**

| Name | Type | Description |
|---|---|---|
| `entity_id` | string | e.g. `"sensor.outdoor_temperature"` |

**Computed:**

| Name | Type | Description |
|---|---|---|
| `state` | string | Current state value |
| `friendly_name` | string | From `attributes.friendly_name` |
| `device_class` | string | From `attributes.device_class` |
| `unit_of_measurement` | string | From `attributes.unit_of_measurement` |
| `attributes` | `map(string)` | All raw attributes (values coerced to string) |
| `last_changed` | string | ISO 8601 |
| `last_updated` | string | ISO 8601 |

---

### `data.homeassistant_template`

Calls `POST /api/template` with `{"template": "<jinja2>"}`. Returns the rendered string.

This is the escape hatch for everything the typed data sources don't cover: area IDs,
floor structure, group members, calculated values, complex filtering. Return JSON from
the template and use `jsondecode()` in Terraform for structured data.

```hcl
# Get all area names as a list
data "homeassistant_template" "areas" {
  template = "{{ areas() | tojson }}"
}

locals {
  area_names = jsondecode(data.homeassistant_template.areas.result)
}

# Get entities in an area filtered by domain
data "homeassistant_template" "living_room_lights" {
  template = <<-JINJA
    {{ area_entities('living_room')
       | select('match', 'light\\.')
       | list
       | tojson }}
  JINJA
}

locals {
  living_room_lights = jsondecode(data.homeassistant_template.living_room_lights.result)
}

# Compute something from entity state
data "homeassistant_template" "avg_temp" {
  template = <<-JINJA
    {{ (states('sensor.bedroom_temp') | float
        + states('sensor.living_room_temp') | float) / 2 }}
  JINJA
}
```

**Arguments:**

| Name | Type | Description |
|---|---|---|
| `template` | string | Jinja2 template (same syntax as HA automations) |

**Computed:**

| Name | Type | Description |
|---|---|---|
| `result` | string | Rendered output |

**Implementation notes:** The template API returns plain text, not JSON. If HA renders
an error (e.g. unknown variable), it returns the error string with HTTP 200 — detect
this by checking if the result starts with `"Error rendering template"` and surface as
a diagnostic error.

---

### `data.homeassistant_config`

Calls `GET /api/config`. Useful for conditional logic based on HA version, unit system,
or loaded components.

```hcl
data "homeassistant_config" "ha" {}

# Only create a celsius-specific resource if the HA instance uses metric
locals {
  use_metric = data.homeassistant_config.ha.unit_system.temperature == "°C"
}

output "ha_version" {
  value = data.homeassistant_config.ha.version
}
```

**Computed:**

| Name | Type |
|---|---|
| `version` | string |
| `location_name` | string |
| `latitude` | number |
| `longitude` | number |
| `elevation` | number |
| `time_zone` | string |
| `unit_system` | `object({ temperature, length, mass, volume })` |
| `components` | `list(string)` |

---

## Full Phase 0 Example

```hcl
provider "homeassistant" {
  url   = "http://homeassistant.local:8123"
  token = var.ha_token
}

# ── Area discovery ──────────────────────────────────────────────────────────

data "homeassistant_template" "all_areas" {
  template = "{{ areas() | tojson }}"
}

locals {
  rooms = jsondecode(data.homeassistant_template.all_areas.result)
}

# ── Per-room entity discovery ───────────────────────────────────────────────

data "homeassistant_entities" "room_lights" {
  for_each = toset(local.rooms)
  domain   = "light"
  area     = each.key
}

data "homeassistant_entities" "room_sensors" {
  for_each     = toset(local.rooms)
  domain       = "sensor"
  area         = each.key
}

data "homeassistant_entities" "all_cameras" {
  domain = "camera"
}

# ── Outputs ─────────────────────────────────────────────────────────────────

output "rooms" {
  value = local.rooms
}

output "bedroom_lights" {
  value = data.homeassistant_entities.room_lights["bedroom"].entity_ids
}

output "all_camera_ids" {
  value = data.homeassistant_entities.all_cameras.entity_ids
}
```

---

## Implementation Checklist

- [ ] `RESTClient` struct with `baseURL`, `token`, `http.Client` (with timeout)
- [ ] Helper methods: `Get(path)`, `Post(path, body)` — both return `([]byte, error)`
- [ ] 401 → hard error with "check your token"
- [ ] 404 → return `nil, nil` (caller decides if that's an error)
- [ ] `data.homeassistant_entities`: implement all filters; make area/label use template
      API rather than trying to parse `.storage` files
- [ ] `data.homeassistant_template`: detect HA template errors in the 200 response body
- [ ] All data sources: use `datasource.Schema` with `Computed: true` on output fields
- [ ] Write acceptance tests against real HA (or docker): check that each filter
      actually narrows the list correctly
