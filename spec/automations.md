# Automations, scripts, scenes

These use REST config endpoints. Each resource has a user-chosen `id` (ADR-0008) and one dynamic
`config` (ADR-0005), compared against the stored baseline (ADR-0023) and by semantic equality
(ADR-0006).

## Common behaviour

| Operation | Call |
|---|---|
| Create | `GET /api/config/<domain>/config/<id>` must be 404 (ADR-0024), then as Update |
| Update | `POST /api/config/<domain>/config/<id>`, body = `config` plus `"id": <id>` (automation, scene) |
| Read | `GET /api/config/<domain>/config/<id>`; 404 → remove from state |
| Delete | `DELETE /api/config/<domain>/config/<id>` |
| Import | by `id` |

- POST replaces an existing item with the same ID. So Create first reads the item, and if it
  exists, fails with an error on `id` that suggests `tofu import` or another `id` (ADR-0024).
- HA reloads asynchronously after a write. Create then polls with backoff until the entity
  appears, which sets `entity_id`. If it has not appeared within `timeouts.create` (default 60s),
  the apply fails and the resource is tainted. Update does not wait: the entity ID never changes
  once the entity exists.
- Read finds the entity cheaply: a script's is `script.<id>`, without a request. For automations
  and scenes, Read checks the entity in state with `GET /api/states/<entity_id>`, and reads all of
  `GET /api/states` only if it is gone or its `attributes.id` no longer matches (after an import,
  or when the user renamed the entity). If no entity is found, Read keeps `entity_id`.
- After a create or update, the provider reads the config back and stores a hash of it as the
  stored baseline in private state. On Read, a config equal to the baseline keeps the prior
  state value. Otherwise it is compared by semantic equality, and a real difference is drift.
  Without a baseline (state from v0.1.0, or after import), only semantic equality applies.
- HA renames old keys when it saves (`trigger`, `condition`, `action` at the top level become
  plurals, and `service` becomes `action` in action steps, also in scripts). The baseline keeps
  this from showing as a diff. The docs mention it, and there is no warning.
- A 400 response carries HA's validation message, which becomes a diagnostic on `config`.
- `config` must be an object and must not contain an `id` key (checked at plan time); the
  provider owns it. YAML copied from the HA UI needs its `id:` line removed. On Read, the
  provider strips `id` from the returned object before comparing.

| Argument | Type | |
|---|---|---|
| `id` | string | required, `^[a-z0-9_]+$`, forces replacement |
| `config` | dynamic (semantic equality) | required |
| `timeouts` | object | optional; `create`, default 60s, bounds the wait for the entity |

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
key `id`, so its entity is `script.<id>`. Create waits for `GET /api/states/script.<id>` to
find it.

## `homeassistant_scene`

`config` is the scene body (`name`, `entities`, `icon`, …). `entities` maps entity IDs to target
state objects. Like an automation's, the scene's entity ID derives from its `name` when it is first
created, so `entity_id` is the `scene.*` state whose `attributes.id == id`.

## Docs requirements

- Examples use the current syntax (`triggers`/`trigger:`, `actions`/`action:`), as required by
  ADR-0003.
- Show the `yamldecode(file(...))` pattern, and reusing actions through `locals`.
