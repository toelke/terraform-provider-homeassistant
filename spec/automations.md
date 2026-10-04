# Automations, scripts, scenes

These use REST config endpoints. Each resource has a user-chosen `id` (ADR-0008) and one dynamic
`config` (ADR-0005), compared by semantic equality (ADR-0006).

## Common behaviour

| Operation | Call |
|---|---|
| Create / Update | `POST /api/config/<domain>/config/<id>`, body = `config` plus `"id": <id>` (automation, scene) |
| Read | `GET /api/config/<domain>/config/<id>`; 404 → remove from state |
| Delete | `DELETE /api/config/<domain>/config/<id>` |
| Import | by `id` |

- HA reloads asynchronously after a write. Create then polls `GET /api/states` until the entity
  appears, which sets `entity_id`. If it has not appeared after a minute, the apply fails and the
  resource is tainted. Update does not wait: the entity ID never changes once the entity exists.
- A 400 response carries HA's validation message, which becomes the diagnostic.
- `config` must be an object and must not contain an `id` key (checked at plan time); the
  provider owns it. YAML copied from the HA UI needs its `id:` line removed. On Read, the
  provider strips `id` from the returned object before comparing.

| Argument | Type | |
|---|---|---|
| `id` | string | required, `^[a-z0-9_]+$`, forces replacement |
| `config` | dynamic (semantic equality) | required |

| Computed | How it is found |
|---|---|
| `entity_id` | automation and scene: the state whose `attributes.id == id`; script: `script.<id>` |

## `homeassistant_automation`

```hcl
resource "homeassistant_automation" "motion_bedroom" {
  id = "motion_bedroom"
  config = {
    alias = "Bedroom — lights on motion"
    mode  = "restart"
    triggers = [{
      trigger   = "state"
      entity_id = "binary_sensor.bedroom_motion"
      to        = "on"
    }]
    conditions = [{
      condition = "time"
      after     = "22:00:00"
      before    = "08:00:00"
    }]
    actions = [{
      action = "light.turn_on"
      target = { area_id = homeassistant_area.bedroom.id }
      data   = { brightness_pct = 20 }
    }]
  }
}

# or from a file copied out of the HA UI
resource "homeassistant_automation" "away" {
  id     = "away_mode"
  config = yamldecode(file("${path.module}/automations/away.yaml"))
}
```

Enabled or disabled is runtime state (ADR-0012). To disable an automation as code, set
`initial_state = false` inside `config`.

## `homeassistant_script`

`config` is the script body (`alias`, `mode`, `sequence`, `fields`, …). HA stores it under the
key `id`, so its entity is `script.<id>`. Create waits for that entity ID to appear in
`GET /api/states`.

## `homeassistant_scene`

`config` is the scene body (`name`, `entities`, `icon`, …). `entities` maps entity IDs to target
state objects.

## Docs requirements

- Examples use the current syntax (`triggers`/`trigger:`, `actions`/`action:`), as required by
  ADR-0003.
- Show the `yamldecode(file(...))` pattern, and reusing actions through `locals`.
