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
| `components` | list(string), sorted (HA builds it from a set, so its order is not stable) |

## `homeassistant_entity`

`GET /api/states/<entity_id>`. A 404 is an error, with a hint to check the ID.

| Argument | Type | |
|---|---|---|
| `entity_id` | string | required; validated at plan time as `<domain>.<object_id>` (lowercase letters, digits, `_`) |

| Computed | Type | Source |
|---|---|---|
| `state` | string | |
| `friendly_name` | string | `attributes.friendly_name` |
| `device_class` | string | `attributes.device_class` |
| `unit_of_measurement` | string | `attributes.unit_of_measurement` |
| `attributes` | dynamic | all attributes, keeping JSON types (ADR-0015) |
| `last_changed`, `last_updated` | string | ISO 8601 |

`friendly_name`, `device_class`, and `unit_of_measurement` are null when the attribute is
missing or not a string.

## `homeassistant_entities`

```hcl
data "homeassistant_entities" "bedroom_lights" {
  domain = "light"
  area   = "bedroom"          # area id or name
}
```

Every filter is optional, and an entity must match all that are set.

| Argument | Type | Notes |
|---|---|---|
| `domain` | string | the part of the entity ID before the dot |
| `area` | string | area ID or name, resolved through `area_entities()`; an unknown area matches nothing |
| `label` | string | label ID or name, resolved through `label_entities()`; an unknown label matches nothing |
| `device_class` | string | equals `attributes.device_class` |
| `name_pattern` | string | glob against `friendly_name`: `*` is any characters, `?` one character, everything else is literal and case-sensitive. Entities without a `friendly_name` never match |

There is no `state` filter (ADR-0015).

| Computed | Type |
|---|---|
| `entity_ids` | list(string), sorted |
| `entities` | dynamic: an object keyed by entity ID, each value `{ state, friendly_name, attributes, last_changed }` |

`entities` is dynamic rather than `map(object)`, because terraform-plugin-framework does not
allow a dynamic value (`attributes`, ADR-0015) inside a map. HCL reads it the same way:
`entities["light.x"].state`, and `for` expressions work on it. `friendly_name` is null when the
entity has none.

**Algorithm:**

1. `GET /api/states`.
2. If `area` or `label` is set, render `{{ area_entities("<x>") | tojson }}` or
   `{{ label_entities("<x>") | tojson }}` through the template API, JSON-decode the result, and
   intersect it with the states. The value is written as a Jinja string literal in which
   `"`, `\`, and every character outside printable ASCII become a `\uXXXX` or `\UXXXXXXXX`
   escape, so it can't end the literal.
3. Apply the remaining filters.
4. Sort the result by entity ID.

## `homeassistant_template`

`POST /api/template` with `{"template": "..."}`. It returns the rendered text in `result`. The
HTTP status is checked, and so is a 200 body starting with `Error rendering template`. Either
becomes an error diagnostic. HA answers a broken template with HTTP 400, so a 400 or such a 200
body is reported on `template`; any other failure is reported like every other REST error.

```hcl
data "homeassistant_template" "areas" {
  template = "{{ areas() | tojson }}"
}
locals {
  area_ids = jsondecode(data.homeassistant_template.areas.result)
}
```
