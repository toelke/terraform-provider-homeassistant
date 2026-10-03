# Phase 4 — Dashboards

**Goal:** Manage Lovelace dashboards — their metadata, views, sections, and cards —
entirely via the WebSocket API. No file writing, no `config_path`.

**Transport:** WebSocket only. Uses the `lovelace/*` command family, the same commands
the HA frontend itself uses when you save a dashboard in the UI editor.

---

## Discovery: the `lovelace` WebSocket API

This exists and is fully functional. The repo
[tecnocrata/ha-openclaw-skill](https://github.com/tecnocrata/ha-openclaw-skill)
documents and exercises it with a working Python implementation.

```json
// List all dashboards
{ "id": 1, "type": "lovelace/dashboards/list" }

// Create a dashboard (metadata only)
{ "id": 2, "type": "lovelace/dashboards/create",
  "url_path": "home", "title": "My Home",
  "icon": "mdi:home-variant",
  "show_in_sidebar": true, "require_admin": false }

// Save (or update) the full view/section/card config
{ "id": 3, "type": "lovelace/config/save",
  "url_path": "home",
  "config": { "views": [ ... ] } }

// Read the current config
{ "id": 4, "type": "lovelace/config", "url_path": "home" }

// Delete
{ "id": 5, "type": "lovelace/dashboards/delete", "dashboard_id": "home" }
```

This removes every complication from the original file-based design:
- No `config_path` on the provider
- No filesystem access required
- `tofu destroy` cleanly deletes the dashboard
- `tofu plan` can diff against the live config read back from HA
- Works identically whether `tofu apply` runs locally, in CI, or in a container

The dashboard entry created by `lovelace/dashboards/create` registers immediately in
HA's sidebar — no manual `configuration.yaml` registration needed.

---

## Modern Dashboard Structure (HA 2026+)

The current standard layout is **sections-based**, not the old flat cards-per-view
layout. The structure has one extra level compared to pre-2024 dashboards:

```
dashboard
  └─ view  (type: sections)
       └─ section  (type: grid)
            └─ card
```

```yaml
views:
  - type: sections
    title: Living Room
    path: living-room
    icon: mdi:sofa
    sections:
      - type: grid
        cards:
          - type: heading
            heading: Lights
          - type: entities
            entities:
              - light.living_room_main
              - light.floor_lamp
          - type: heading
            heading: Climate
          - type: entity
            entity: climate.living_room
```

The older flat layout (`type: masonry` with direct `cards:`) still works and is
appropriate for panel/sidebar views. Both are supported.

---

## Resource Model

```
homeassistant_dashboard_fragment   →  .section_yaml  (one grid section)
         ↓ referenced in
homeassistant_dashboard  →  view { section_fragments = [...] }
         ↓
  lovelace/dashboards/create  +  lovelace/config/save  via WebSocket
```

**Fragment** = one `section` block (a grid of cards). Fragments are the unit of reuse.
A view in the `sections` layout contains one or more sections, each backed by a fragment.
A fragment can appear in multiple views across multiple dashboards.

**For masonry/panel views** (older layout), fragments produce a flat card list instead.
The fragment detects this from the parent view's `type`.

---

## `homeassistant_dashboard_fragment`

A reusable section of cards. Its primary output is `section_yaml` — a YAML string
representing one `type: grid` section block, ready to be spliced into a `sections:`
list. It also exposes `cards_yaml` for use in masonry/panel views.

### Top-level arguments

| Name | Type | Description |
|---|---|---|
| `heading` | string (optional) | Prepends a `type: heading` card at the top of the section |
| `heading_style` | string (optional) | `"title"` (default) or `"subtitle"` |
| `columns` | number (optional) | Grid column hint for the section (default: HA auto) |
| `card_*` | block (multiple) | Typed card blocks |
| `card_raw` | block (multiple) | Verbatim YAML per card — escape hatch |

### Computed outputs

| Name | Type | Description |
|---|---|---|
| `section_yaml` | string | Full `type: grid` section block — use in sections-layout views |
| `cards_yaml` | string | Flat card list — use in masonry/panel views |

---

### Card DSL

Every typed card block accepts a `raw_yaml` string that is deep-merged last, so any
field not in the typed schema can be set without losing type safety on common fields.

---

#### `card_heading`

```hcl
card_heading {
  heading       = "Lights"
  heading_style = "title"   # title | subtitle
  icon          = "mdi:lightbulb"
}
```

---

#### `card_entity` — single entity tile (modern 2026+ card)

```hcl
card_entity {
  entity         = "climate.living_room"
  name           = "Living Room"
  icon           = "mdi:thermometer"
  secondary_info = "last-changed"   # last-changed | last-updated | last-triggered
}
```

---

#### `card_entities` — grouped entity list

```hcl
card_entities {
  title              = "Lights"
  entities           = data.homeassistant_entities.living_room_lights.entity_ids
  show_header_toggle = true
  state_color        = true
  raw_yaml = <<-YAML
    footer:
      type: buttons
      entities:
        - entity: scene.movie_mode
          name: Movie Mode
  YAML
}
```

| Arg | Type | Default |
|---|---|---|
| `title` | string | — |
| `entities` | `list(string)` | required |
| `show_header_toggle` | bool | `true` |
| `state_color` | bool | `false` |
| `theme` | string | — |
| `raw_yaml` | string | — |

---

#### `card_tile` — compact tile (modern default card)

```hcl
card_tile {
  entity     = "light.bedroom_ceiling"
  name       = "Ceiling"
  icon       = "mdi:ceiling-light"
  color      = "amber"
  hide_state = false
  vertical   = false
}
```

---

#### `card_glance` — compact multi-entity overview

```hcl
card_glance {
  title      = "Sensors"
  entities   = data.homeassistant_entities.bedroom_sensors.entity_ids
  columns    = 4
  show_state = true
  show_name  = true
  show_icon  = true
}
```

---

#### `card_button` — action button

```hcl
card_button {
  entity   = homeassistant_script.good_morning.entity_id
  name     = "Good Morning"
  icon     = "mdi:coffee"
  show_state = false

  tap_action {
    action = "toggle"
  }
  hold_action {
    action = "more-info"
  }
}
```

**`tap_action` / `hold_action` / `double_tap_action` block:**

| Arg | Values |
|---|---|
| `action` | `toggle`, `call-service`, `navigate`, `more-info`, `none` |
| `service` | string (for `call-service`) |
| `service_data` | `map(string)` |
| `navigation_path` | string (for `navigate`) |

---

#### `card_gauge`

```hcl
card_gauge {
  entity = "sensor.bedroom_humidity"
  name   = "Humidity"
  min    = 0
  max    = 100
  unit   = "%"
  needle = true
  severity {
    green  = 30
    yellow = 60
    red    = 80
  }
}
```

---

#### `card_history_graph`

```hcl
card_history_graph {
  title         = "Temperature (24h)"
  entities      = ["sensor.bedroom_temp", "sensor.outdoor_temp"]
  hours_to_show = 24
}
```

---

#### `card_thermostat`

```hcl
card_thermostat {
  entity = "climate.bedroom"
  name   = "Bedroom"
  raw_yaml = "features:\n  - type: climate-hvac-modes"
}
```

---

#### `card_weather_forecast`

```hcl
card_weather_forecast {
  entity        = "weather.forecast_home"
  forecast_type = "daily"    # daily | hourly | twice_daily
  show_forecast = true
}
```

---

#### `card_picture_entity`

```hcl
card_picture_entity {
  entity      = "camera.front_door"
  camera_view = "live"
  show_name   = false
  show_state  = false
}
```

---

#### `card_markdown`

```hcl
card_markdown {
  content = <<-MD
    ## {{ now().strftime('%A, %-d %B') }}
    Outdoor: **{{ states('sensor.outdoor_temp') }}°C**
  MD
}
```

---

#### `card_map`

```hcl
card_map {
  title        = "Family"
  entities     = data.homeassistant_entities.trackers.entity_ids
  aspect_ratio = "16x9"
  default_zoom = 14
}
```

---

#### `card_media_control`

```hcl
card_media_control {
  entity = "media_player.living_room_tv"
}
```

---

#### `card_conditional`

```hcl
card_conditional {
  condition {
    entity = "binary_sensor.someone_home"
    state  = "on"
  }
  # Exactly one inner card block:
  card_entities {
    title    = "Welcome Home"
    entities = ["light.entrance", "lock.front_door"]
  }
}
```

---

#### `card_vertical_stack` / `card_horizontal_stack`

Nest any card blocks, including other stacks, to arbitrary depth.

```hcl
card_horizontal_stack {
  card_gauge {
    entity = "sensor.router_cpu"
    name   = "CPU"
    min    = 0
    max    = 100
  }
  card_gauge {
    entity = "sensor.router_memory"
    name   = "RAM"
    min    = 0
    max    = 100
  }
}
```

---

#### `card_raw` — full escape hatch

```hcl
card_raw {
  yaml = <<-YAML
    type: custom:mini-graph-card
    entities:
      - sensor.bedroom_temperature
    hours_to_show: 48
    line_width: 2
    color: "#e74c3c"
  YAML
}
```

---

## `homeassistant_dashboard`

Creates and manages a complete Lovelace dashboard via WebSocket.

Two WS calls happen on every create/update:
1. `lovelace/dashboards/create` (or update metadata if the dashboard already exists)
2. `lovelace/config/save` with the full assembled view+section+card config

On `tofu plan`, the provider reads back the live config via `lovelace/config` and diffs
it against the desired state.

### Top-level arguments

| Name | Type | Description |
|---|---|---|
| `title` | string | Dashboard title shown in sidebar |
| `url_path` | string | URL path and dashboard ID, e.g. `"home"` → `/home` |
| `icon` | string (optional) | Sidebar icon, e.g. `"mdi:home-variant"` |
| `show_in_sidebar` | bool (optional) | Default `true` |
| `require_admin` | bool (optional) | Default `false` |
| `view` | block (multiple) | One block per tab |

### `view` block

| Name | Type | Description |
|---|---|---|
| `title` | string | Tab title |
| `path` | string (optional) | URL path segment; defaults to slugified title |
| `icon` | string (optional) | Tab icon |
| `type` | string (optional) | `"sections"` (default, 2026+), `"masonry"`, `"panel"`, `"sidebar"` |
| `theme` | string (optional) | Theme name |
| `visible` | bool or `list({user=string})` | Tab visibility |
| `section_fragments` | `list(string)` | For `type = "sections"`: ordered list of `fragment.section_yaml` |
| `card_fragments` | `list(string)` | For `type = "masonry"` / `"panel"`: ordered list of `fragment.cards_yaml` |
| `raw_yaml` | string (optional) | Verbatim YAML appended to the view block |

### Computed outputs

| Name | Type | Description |
|---|---|---|
| `id` | string | Dashboard ID (same as `url_path`) |
| `config_json` | string | The full config as sent to HA (useful for debugging) |

### Import

```bash
tofu import homeassistant_dashboard.home home
# → calls lovelace/config to read existing config and populate state
```

---

## Full Phase 4 Example

```hcl
# ── Data from earlier phases ──────────────────────────────────────────────────

data "homeassistant_entities" "room_lights" {
  for_each = toset(local.rooms)
  domain   = "light"
  area     = each.key
}

data "homeassistant_entities" "room_climate" {
  for_each = toset(local.rooms)
  domain   = "climate"
  area     = each.key
}

data "homeassistant_entities" "all_cameras" {
  domain = "camera"
}

# ── Fragments ─────────────────────────────────────────────────────────────────

# Per-room light section — one per room
resource "homeassistant_dashboard_fragment" "room_lights" {
  for_each = toset(local.rooms)
  heading  = "Lights"

  card_entities {
    entities    = data.homeassistant_entities.room_lights[each.key].entity_ids
    state_color = true
  }
}

# Per-room climate section — one per room
resource "homeassistant_dashboard_fragment" "room_climate" {
  for_each = toset(local.rooms)
  heading  = "Climate"

  dynamic "card_thermostat" {
    for_each = data.homeassistant_entities.room_climate[each.key].entity_ids
    content {
      entity = card_thermostat.value
    }
  }

  card_history_graph {
    title         = "Temperature (24h)"
    entities      = data.homeassistant_entities.room_climate[each.key].entity_ids
    hours_to_show = 24
  }
}

# Quick actions — reused on Overview and Bedroom
resource "homeassistant_dashboard_fragment" "quick_actions" {
  heading = "Quick Actions"

  card_horizontal_stack {
    card_button {
      entity = homeassistant_script.good_morning.entity_id
      name   = "Good Morning"
      icon   = "mdi:coffee"
      tap_action { action = "toggle" }
    }
    card_button {
      entity = homeassistant_script.goodnight.entity_id
      name   = "Goodnight"
      icon   = "mdi:weather-night"
      tap_action { action = "toggle" }
    }
  }

  card_entity {
    entity = homeassistant_input_select.house_mode.entity_id
    name   = "House Mode"
    icon   = "mdi:home-variant"
  }
}

# Weather — reused on Overview
resource "homeassistant_dashboard_fragment" "weather" {
  card_weather_forecast {
    entity        = "weather.forecast_home"
    forecast_type = "daily"
  }
  card_markdown {
    content = "**{{ now().strftime('%A, %-d %B') }}** · {{ states('sensor.outdoor_temp') }}°C outside"
  }
}

# Security — cameras + alarm, reused on Overview and Security view
resource "homeassistant_dashboard_fragment" "security" {
  heading = "Cameras"

  dynamic "card_picture_entity" {
    for_each = data.homeassistant_entities.all_cameras.entity_ids
    content {
      entity      = card_picture_entity.value
      camera_view = "live"
      show_name   = true
    }
  }

  card_raw {
    yaml = <<-YAML
      type: custom:alarm-panel
      entity: alarm_control_panel.home
    YAML
  }
}

# ── Dashboard ──────────────────────────────────────────────────────────────────

resource "homeassistant_dashboard" "home" {
  title           = "My Home"
  url_path        = "home"
  icon            = "mdi:home-variant"
  show_in_sidebar = true

  # Overview: weather + actions + security, each as a separate section
  view {
    title = "Overview"
    path  = "overview"
    icon  = "mdi:view-dashboard"
    type  = "sections"

    section_fragments = [
      homeassistant_dashboard_fragment.weather.section_yaml,
      homeassistant_dashboard_fragment.quick_actions.section_yaml,  # ← reused below
      homeassistant_dashboard_fragment.security.section_yaml,       # ← reused below
    ]
  }

  # One sections view per room, dynamically generated
  dynamic "view" {
    for_each = toset(local.rooms)
    content {
      title = view.key
      path  = replace(lower(view.key), " ", "-")
      icon  = "mdi:door"
      type  = "sections"

      section_fragments = [
        homeassistant_dashboard_fragment.room_lights[view.key].section_yaml,
        homeassistant_dashboard_fragment.room_climate[view.key].section_yaml,
      ]
    }
  }

  # Bedroom gets quick actions bolted on as an extra section — same fragment
  view {
    title = "Bedroom ☀️"
    path  = "bedroom-morning"
    icon  = "mdi:bed"
    type  = "sections"

    section_fragments = [
      homeassistant_dashboard_fragment.room_lights["bedroom"].section_yaml,
      homeassistant_dashboard_fragment.quick_actions.section_yaml,   # ← same fragment, second use
      homeassistant_dashboard_fragment.room_climate["bedroom"].section_yaml,
    ]
  }

  # Security: full-width panel view for cameras — masonry would also work
  view {
    title = "Security"
    path  = "security"
    icon  = "mdi:shield-home"
    type  = "sections"

    section_fragments = [
      homeassistant_dashboard_fragment.security.section_yaml,        # ← same fragment
    ]
  }
}
```

The config sent to HA via `lovelace/config/save` looks like:

```json
{
  "views": [
    {
      "type": "sections",
      "title": "Overview",
      "path": "overview",
      "icon": "mdi:view-dashboard",
      "sections": [
        {
          "type": "grid",
          "cards": [
            { "type": "weather-forecast", "entity": "weather.forecast_home", "forecast_type": "daily" },
            { "type": "markdown", "content": "**Monday, 6 April** · 12°C outside" }
          ]
        },
        {
          "type": "grid",
          "cards": [
            { "type": "heading", "heading": "Quick Actions" },
            { "type": "horizontal-stack", "cards": [ ... ] },
            { "type": "entity", "entity": "input_select.house_mode", "name": "House Mode" }
          ]
        },
        {
          "type": "grid",
          "cards": [
            { "type": "heading", "heading": "Cameras" },
            { "type": "picture-entity", "entity": "camera.front_door", "camera_view": "live" }
          ]
        }
      ]
    },
    ...
  ]
}
```

---

## Config Serialisation

```go
// internal/resources/dashboard.go

type DashboardConfig struct {
    Views []ViewConfig `json:"views"`
}

type ViewConfig struct {
    Type     string          `json:"type"`
    Title    string          `json:"title"`
    Path     string          `json:"path,omitempty"`
    Icon     string          `json:"icon,omitempty"`
    Sections []SectionConfig `json:"sections,omitempty"` // for type: sections
    Cards    []CardConfig    `json:"cards,omitempty"`    // for type: masonry / panel
}

type SectionConfig struct {
    Type  string       `json:"type"` // always "grid"
    Cards []CardConfig `json:"cards"`
}

// Fragment.section_yaml serialises to a SectionConfig JSON object.
// Dashboard assembles these into ViewConfig.Sections.
// The provider JSON-encodes the final DashboardConfig and sends it via
// lovelace/config/save.
```

Since the config is sent as a JSON object (not a YAML file), use `encoding/json`
rather than a YAML library for the final serialisation. Card blocks are still
built as `map[string]interface{}` and have `raw_yaml` deep-merged before encoding.

---

## Drift Detection

On `Read`, the provider calls `lovelace/config` and compares the returned config
against what's stored in Terraform state. Because the config is a nested JSON object,
use a **normalised JSON comparison** (marshal → unmarshal → re-marshal) rather than
string comparison to avoid false diffs from key ordering.

```go
func normalizeConfig(config map[string]interface{}) (string, error) {
    b, err := json.Marshal(config)
    if err != nil {
        return "", err
    }
    var v interface{}
    json.Unmarshal(b, &v)
    b2, err := json.Marshal(v)
    return string(b2), err
}
```

---

## Updated Reference Doc Impact

Phase 4's transport column in `00-reference.md` changes from `local` to `WebSocket`.
The provider's `config_path` argument is no longer needed at all — remove it from the
provider schema. Update the resource table:

| Resource | Transport | Phase |
|---|---|---|
| `homeassistant_dashboard_fragment` | local (pure computation) | 4 |
| `homeassistant_dashboard` | **WebSocket** | 4 |

---

## Implementation Checklist

- [ ] `WSClient.SendCommand` already exists from Phase 1/2 — reuse directly
- [ ] `homeassistant_dashboard`:
  - Create: `lovelace/dashboards/create` then `lovelace/config/save`
  - Read: `lovelace/config` → normalise → store hash for drift detection
  - Update: if metadata changed (`title`, `icon`, etc.) call a metadata update
    command; always call `lovelace/config/save` with the new config
  - Delete: `lovelace/dashboards/delete`
  - Import: `ImportState` uses `url_path` as the ID → calls `lovelace/config`
- [ ] `homeassistant_dashboard_fragment`: pure computed, no API
  - Compute `section_yaml` and `cards_yaml` at plan time (use
    `resource.ModifyPlan` so downstream resources can plan correctly)
  - `card_conditional`: validate exactly one inner card block — error at plan time
  - `card_vertical_stack` / `card_horizontal_stack`: recursive card block support
  - `card_raw`: parse YAML, re-encode as JSON for the final payload
  - `raw_yaml` deep merge: merge after typed fields, raw wins on conflict
- [ ] `card_heading` is first-class (not a `raw_yaml` workaround) — important for
      the sections format where headings divide content visually
- [ ] Serialise cards as `map[string]interface{}` → `encoding/json` → send in the
      `lovelace/config/save` payload (not YAML — HA's WS API takes JSON objects)
- [ ] Normalised JSON comparison for drift detection
- [ ] Remove `config_path` from provider schema (it served no purpose once
      dashboards are WS-managed; YAML-mode dashboards are explicitly out of scope)
- [ ] Acceptance test: create dashboard → list dashboards (confirm present) →
      update title → read (confirm diff) → destroy → list (confirm gone)
- [ ] Test fragment reuse: same fragment in two views → confirm identical section
      YAML in both places in the final config
