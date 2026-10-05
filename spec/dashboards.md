# Dashboards

The WebSocket `lovelace/*` API, storage-mode dashboards only. YAML-mode dashboards and the default
"Overview" dashboard are out of scope.

## `homeassistant_dashboard`

```hcl
locals {
  quick_actions = {
    type  = "grid"
    cards = [
      { type = "heading", heading = "Quick actions" },
      { type = "button", entity = homeassistant_script.good_morning.entity_id },
    ]
  }
}

resource "homeassistant_dashboard" "home" {
  url_path = "my-home"
  title    = "My Home"
  icon     = "mdi:home-variant"
  config = {
    views = [{
      type     = "sections"
      title    = "Overview"
      sections = [local.quick_actions]
    }]
  }
}

# The same section, reused on a second dashboard: edit the local once, both dashboards change.
resource "homeassistant_dashboard" "wall_tablet" {
  url_path        = "wall-tablet"
  title           = "Wall Tablet"
  show_in_sidebar = false
  config = {
    views = [{
      type     = "sections"
      title    = "Hallway"
      sections = [
        local.quick_actions,
        { type = "grid", cards = [{ type = "tile", entity = "lock.front_door" }] },
      ]
    }]
  }
}
```

| Argument | Type | |
|---|---|---|
| `url_path` | string | required, must contain `-`, forces replacement |
| `title` | string | required |
| `icon` | string | optional |
| `show_in_sidebar` | bool | default `true` |
| `require_admin` | bool | default `false` |
| `config` | dynamic (stored baseline, ADR-0023; semantic equality) | required; the full Lovelace config (`views`, …) |

| Computed | |
|---|---|
| `id` | `url_path` (also the import ID) |
| `dashboard_id` | HA's internal ID, `slugify(url_path)`; looked up via `lovelace/dashboards/list` |

| Operation | Calls |
|---|---|
| Create | `lovelace/dashboards/create` (metadata) → `lovelace/config/save` (`url_path`, `config`) → `lovelace/config`, stored as the baseline (ADR-0023); if the save fails, the dashboard is deleted again |
| Read | `lovelace/dashboards/list` (find by `url_path`; missing → remove from state; not `mode: storage` → error) + `lovelace/config` (`force: true`; `config_not_found` → `config` null, so the next apply saves it). A config equal to the baseline keeps the prior `config` |
| Update | `lovelace/dashboards/update` (by `dashboard_id`) if metadata changed, with all four fields (an unset `icon` is sent as null, which clears it); `lovelace/config/save` if config changed, then `lovelace/config` for a new baseline |
| Delete | `lovelace/dashboards/delete` (by `dashboard_id`; `not_found` counts as deleted) |

## Reuse (ADR-0007)

There is no fragment resource. Sections and cards are reused through `locals`, module outputs,
and `for` expressions. The example above shows one section local used in two dashboards. The
docs must also show a section generated per area with `for` over `homeassistant_entities`:

```hcl
data "homeassistant_entities" "lights" {
  for_each = toset(["kitchen", "bedroom"])
  domain   = "light"
  area     = each.key
}

locals {
  light_sections = {
    for area, d in data.homeassistant_entities.lights : area => {
      type  = "grid"
      cards = concat(
        [{ type = "heading", heading = title(area) }],
        [for e in d.entity_ids : { type = "tile", entity = e }],
      )
    }
  }
}

# One view per area: its lights section plus the shared quick actions.
#   views = [for area, s in local.light_sections : {
#     type = "sections", title = title(area), sections = [s, local.quick_actions]
#   }]
```

Card builder functions ([functions.md](functions.md)) build the same objects with less typing.
