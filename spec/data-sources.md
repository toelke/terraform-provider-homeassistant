# Data sources (REST)

Read-only lookups of live HA data. They use REST only, so they never open a WebSocket.

## `homeassistant_config`

`GET /api/config`.

| Computed | Type |
|---|---|
| `version` | string |
| `location_name` | string |
| `latitude`, `longitude`, `elevation` | number |
| `time_zone` | string |
| `unit_system` | object `{ temperature, length, mass, volume }` |
| `components` | list(string) |

## `homeassistant_entity`

`GET /api/states/<entity_id>`. A 404 is an error, with a hint to check the ID.

| Argument | Type | |
|---|---|---|
| `entity_id` | string | required |

| Computed | Type | Source |
|---|---|---|
| `state` | string | |
| `friendly_name` | string | `attributes.friendly_name` |
| `device_class` | string | `attributes.device_class` |
| `unit_of_measurement` | string | `attributes.unit_of_measurement` |
| `attributes` | dynamic | all attributes, keeping JSON types (ADR-0015) |
| `last_changed`, `last_updated` | string | ISO 8601 |

## `homeassistant_entities`

```hcl
data "homeassistant_entities" "bedroom_lights" {
  domain = "light"
  area   = "bedroom"          # area id or name
}
```

| Argument | Type | Notes |
|---|---|---|
| `domain` | string | entity ID prefix |
| `area` | string | area ID or name, resolved through `area_entities()` |
| `label` | string | label ID or name, resolved through `label_entities()` |
| `device_class` | string | matches `attributes.device_class` |
| `name_pattern` | string | glob against `friendly_name` |

There is no `state` filter (ADR-0015).

| Computed | Type |
|---|---|
| `entity_ids` | list(string), sorted |
| `entities` | map(object `{ state, friendly_name, attributes (dynamic), last_changed }`) keyed by entity ID |

**Algorithm:**

1. `GET /api/states`.
2. If `area` or `label` is set, render `{{ area_entities('<x>') | tojson }}` or
   `{{ label_entities('<x>') | tojson }}` through the template API, JSON-decode the result, and
   intersect it with the states. Quote the value safely inside the template.
3. Apply the remaining filters.
4. Sort the result.

## `homeassistant_template`

`POST /api/template` with `{"template": "..."}`. It returns the rendered text in `result`. The
HTTP status is checked, and so is a 200 body starting with `Error rendering template`. Either
becomes an error diagnostic.

```hcl
data "homeassistant_template" "areas" {
  template = "{{ areas() | tojson }}"
}
locals {
  area_ids = jsondecode(data.homeassistant_template.areas.result)
}
```
