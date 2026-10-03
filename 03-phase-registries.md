# Phase 2 — Organisational Registries

**Goal:** Manage HA's spatial and logical hierarchy — floors, areas, labels — and
customise how devices and entities appear in the UI. This phase gives Terraform full
control over the metadata layer that all automations and dashboards are built on top of.

**Transport:** WebSocket only. All registry commands require admin auth.

---

## The Registry Model

HA maintains four registries relevant here:

```
Floor        (building level)
  └─ Area    (room / zone, belongs to a floor)
       └─ Device   (hardware unit, assigned to an area)
            └─ Entity  (individual capability of a device)

Label        (cross-cutting tag; can be applied to areas, devices, and entities)
```

Devices and entities are **created by integrations** (Phase 1), not by the provider.
This phase manages their metadata: which area a device is in, what it's called, what
labels it has, whether it's hidden or disabled.

---

## WebSocket Commands Reference

```json
// Areas
{ "type": "config/area_registry/list" }
{ "type": "config/area_registry/create", "name": "Bedroom", "floor_id": "ground_floor", "aliases": [], "icon": "mdi:bed" }
{ "type": "config/area_registry/update", "area_id": "bedroom", "name": "Master Bedroom" }
{ "type": "config/area_registry/delete", "area_id": "bedroom" }

// Floors
{ "type": "config/floor_registry/list" }
{ "type": "config/floor_registry/create", "name": "Ground Floor", "level": 0, "aliases": [], "icon": "mdi:home-floor-0" }
{ "type": "config/floor_registry/update", "floor_id": "ground_floor", "name": "Ground Floor" }
{ "type": "config/floor_registry/delete", "floor_id": "ground_floor" }

// Labels
{ "type": "config/label_registry/list" }
{ "type": "config/label_registry/create", "name": "Managed", "color": "#4CAF50", "icon": "mdi:tag" }
{ "type": "config/label_registry/update", "label_id": "managed", "name": "Managed" }
{ "type": "config/label_registry/delete", "label_id": "managed" }

// Entity registry
{ "type": "config/entity_registry/list" }
{ "type": "config/entity_registry/get", "entity_id": "light.bedroom_ceiling" }
{ "type": "config/entity_registry/update",
  "entity_id": "light.bedroom_ceiling",
  "name": "Ceiling Light",
  "icon": "mdi:ceiling-light",
  "area_id": "bedroom",
  "labels": ["managed"],
  "disabled_by": null,
  "hidden_by": null
}

// Device registry
{ "type": "config/device_registry/list" }
{ "type": "config/device_registry/update",
  "device_id": "abc123",
  "name_by_user": "Bedroom Sensor",
  "area_id": "bedroom",
  "labels": ["managed"],
  "disabled_by": null
}
```

---

## Resources

### `homeassistant_floor`

```hcl
resource "homeassistant_floor" "ground" {
  name    = "Ground Floor"
  level   = 0
  icon    = "mdi:home-floor-0"
  aliases = ["Ground", "Main Floor"]
}

resource "homeassistant_floor" "first" {
  name    = "First Floor"
  level   = 1
  icon    = "mdi:home-floor-1"
}
```

**Arguments:**

| Name | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Display name |
| `level` | number | no | Floor level integer (0 = ground) |
| `icon` | string | no | MDI icon |
| `aliases` | `list(string)` | no | Alternative names for voice assistants |

**Computed:**

| Name | Type | Description |
|---|---|---|
| `id` / `floor_id` | string | HA-assigned floor ID slug |

**Import:**
```bash
tofu import homeassistant_floor.ground ground_floor
```

---

### `homeassistant_area`

```hcl
resource "homeassistant_area" "bedroom" {
  name     = "Bedroom"
  floor_id = homeassistant_floor.first.id
  icon     = "mdi:bed"
  aliases  = ["Master Bedroom", "Main Bedroom"]
}

resource "homeassistant_area" "living_room" {
  name     = "Living Room"
  floor_id = homeassistant_floor.ground.id
  icon     = "mdi:sofa"
}
```

**Arguments:**

| Name | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Display name |
| `floor_id` | string | no | ID of the floor this area belongs to |
| `icon` | string | no | MDI icon |
| `aliases` | `list(string)` | no | Alternative names |
| `picture` | string | no | URL to an image for the area card |
| `temperature_entity_id` | string | no | Entity to use as area temperature |
| `humidity_entity_id` | string | no | Entity to use as area humidity |

**Computed:**

| Name | Type | Description |
|---|---|---|
| `id` / `area_id` | string | HA-assigned area ID (slug derived from name) |

**Import:**
```bash
tofu import homeassistant_area.bedroom bedroom
```

---

### `homeassistant_label`

```hcl
resource "homeassistant_label" "managed" {
  name  = "Managed"
  color = "#4CAF50"
  icon  = "mdi:terraform"
}

resource "homeassistant_label" "critical" {
  name  = "Critical"
  color = "#F44336"
  icon  = "mdi:alert"
}
```

**Arguments:**

| Name | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Display name |
| `color` | string | no | Hex color, e.g. `"#4CAF50"` |
| `icon` | string | no | MDI icon |

**Computed:**

| Name | Type | Description |
|---|---|---|
| `id` / `label_id` | string | HA-assigned label ID |

---

### `homeassistant_entity_customization`

Updates the entity registry entry for an existing entity. Does **not** create or delete
entities — those are owned by the integration. Think of this resource as applying a
patch to an entity's metadata.

The resource ID is the `entity_id` string itself. This makes imports trivial and
reflects that these are overlays on top of integration-owned entities.

```hcl
# Basic: assign to area and label
resource "homeassistant_entity_customization" "bedroom_ceiling" {
  entity_id = "light.bedroom_ceiling"
  name      = "Ceiling Light"
  icon      = "mdi:ceiling-light"
  area_id   = homeassistant_area.bedroom.id
  labels    = [homeassistant_label.managed.id]
}

# Disable a noisy entity
resource "homeassistant_entity_customization" "hidden_diagnostic" {
  entity_id   = "sensor.esp_bedroom_wifi_signal"
  hidden_by   = "user"
  area_id     = homeassistant_area.bedroom.id
}

# Rename and expose to voice assistant
resource "homeassistant_entity_customization" "bedroom_light_voice" {
  entity_id        = "light.esp_bedroom_light_1"
  name             = "Bedroom Ceiling"
  area_id          = homeassistant_area.bedroom.id
  # voice assistant exposure is managed via a separate WS command:
  # "conversation/expose_entity" — include if needed
}

# Bulk assign all lights in an area (using data source from Phase 0)
resource "homeassistant_entity_customization" "bedroom_lights" {
  for_each = toset(data.homeassistant_entities.bedroom_lights.entity_ids)

  entity_id = each.key
  area_id   = homeassistant_area.bedroom.id
  labels    = [homeassistant_label.managed.id]
}
```

**Arguments:**

| Name | Type | Required | Description |
|---|---|---|---|
| `entity_id` | string | yes | e.g. `"light.bedroom_ceiling"` — also the resource ID |
| `name` | string | no | Override the entity's display name |
| `icon` | string | no | Override the entity's icon |
| `area_id` | string | no | Assign to area |
| `labels` | `list(string)` | no | Apply labels |
| `disabled_by` | string | no | `"user"` to disable, `null` to enable |
| `hidden_by` | string | no | `"user"` to hide, `null` to show |
| `entity_category` | string | no | `"config"` or `"diagnostic"` |

**Computed:**

| Name | Type | Description |
|---|---|---|
| `original_name` | string | Name from the integration (before customization) |
| `platform` | string | Integration domain that owns this entity |
| `device_id` | string | Device this entity belongs to |

**Import:**
```bash
tofu import homeassistant_entity_customization.bedroom_ceiling light.bedroom_ceiling
```

---

### `homeassistant_device_customization`

Updates the device registry entry for an existing device.

```hcl
resource "homeassistant_device_customization" "bedroom_esp" {
  device_id    = homeassistant_esphome.bedroom_sensor.device_id
  name_by_user = "Bedroom Environmental Sensor"
  area_id      = homeassistant_area.bedroom.id
  labels       = [homeassistant_label.managed.id]
}

# Disable a device that's temporarily unplugged
resource "homeassistant_device_customization" "offline_device" {
  device_id   = "abc123xyz"
  disabled_by = "user"
}
```

**Arguments:**

| Name | Type | Required | Description |
|---|---|---|---|
| `device_id` | string | yes | HA device registry UUID — also the resource ID |
| `name_by_user` | string | no | Override device name |
| `area_id` | string | no | Assign to area |
| `labels` | `list(string)` | no | Apply labels |
| `disabled_by` | string | no | `"user"` to disable, `null` to enable |

**Computed:**

| Name | Type | Description |
|---|---|---|
| `name` | string | Original device name (from integration) |
| `manufacturer` | string | |
| `model` | string | |
| `sw_version` | string | |
| `hw_version` | string | |
| `identifiers` | `list(list(string))` | Integration-specific identifiers |
| `entry_id` | string | Config entry that created this device |

---

## Data Sources

### `data.homeassistant_areas`

```hcl
data "homeassistant_areas" "all" {}

output "area_ids" {
  value = { for a in data.homeassistant_areas.all.areas : a.name => a.area_id }
}
```

**Computed:** `areas` — list of objects with `area_id`, `name`, `floor_id`, `icon`, `aliases`.

### `data.homeassistant_area`

```hcl
data "homeassistant_area" "bedroom" {
  name = "Bedroom"
  # or: area_id = "bedroom"
}

output "bedroom_area_id" {
  value = data.homeassistant_area.bedroom.area_id
}
```

### `data.homeassistant_devices`

```hcl
data "homeassistant_devices" "esphome" {
  integration = "esphome"
}

data "homeassistant_devices" "by_manufacturer" {
  manufacturer = "Espressif"
}

data "homeassistant_devices" "by_area" {
  area_id = homeassistant_area.bedroom.id
}
```

**Computed:** `devices` — list of objects with `device_id`, `name`, `manufacturer`,
`model`, `sw_version`, `area_id`, `labels`, `entry_id`.

### `data.homeassistant_device`

```hcl
data "homeassistant_device" "bedroom_esp" {
  name = "esp-bedroom"
  # or: device_id = "abc123"
}
```

---

## Full Phase 2 Example

```hcl
# ── Prerequisite: Phase 1 resources must exist ───────────────────────────────

# ── Labels ───────────────────────────────────────────────────────────────────

resource "homeassistant_label" "managed" {
  name  = "Managed by Terraform"
  color = "#5C6BC0"
  icon  = "mdi:terraform"
}

# ── Floors ───────────────────────────────────────────────────────────────────

resource "homeassistant_floor" "ground" {
  name  = "Ground Floor"
  level = 0
  icon  = "mdi:home-floor-0"
}

resource "homeassistant_floor" "first" {
  name  = "First Floor"
  level = 1
  icon  = "mdi:home-floor-1"
}

# ── Areas ────────────────────────────────────────────────────────────────────

locals {
  ground_rooms = {
    living_room = { icon = "mdi:sofa",    name = "Living Room" }
    kitchen     = { icon = "mdi:chef-hat", name = "Kitchen" }
    garage      = { icon = "mdi:garage",   name = "Garage" }
  }
  first_rooms = {
    bedroom     = { icon = "mdi:bed",       name = "Bedroom" }
    office      = { icon = "mdi:desk",       name = "Office" }
    bathroom    = { icon = "mdi:shower",     name = "Bathroom" }
  }
}

resource "homeassistant_area" "ground" {
  for_each = local.ground_rooms
  name     = each.value.name
  icon     = each.value.icon
  floor_id = homeassistant_floor.ground.id
}

resource "homeassistant_area" "first" {
  for_each = local.first_rooms
  name     = each.value.name
  icon     = each.value.icon
  floor_id = homeassistant_floor.first.id
}

# ── Device assignment ─────────────────────────────────────────────────────────

resource "homeassistant_device_customization" "esphome_devices" {
  for_each = homeassistant_esphome.devices

  device_id    = each.value.device_id
  name_by_user = "ESPHome — ${each.key}"
  area_id      = homeassistant_area.first["bedroom"].id  # example
  labels       = [homeassistant_label.managed.id]
}

# ── Bulk entity assignment ────────────────────────────────────────────────────

# All entities in the bedroom area get the managed label
data "homeassistant_entities" "bedroom_all" {
  area = "bedroom"
}

resource "homeassistant_entity_customization" "bedroom_entities" {
  for_each = toset(data.homeassistant_entities.bedroom_all.entity_ids)

  entity_id = each.key
  area_id   = homeassistant_area.first["bedroom"].id
  labels    = [homeassistant_label.managed.id]
}

# ── Hide diagnostic entities ─────────────────────────────────────────────────

data "homeassistant_entities" "wifi_sensors" {
  domain       = "sensor"
  device_class = "signal_strength"
}

resource "homeassistant_entity_customization" "hide_wifi_sensors" {
  for_each = toset(data.homeassistant_entities.wifi_sensors.entity_ids)

  entity_id = each.key
  hidden_by = "user"
}
```

---

## Implementation Checklist

- [ ] `WSClient.SendCommand(ctx, type, params)` → returns result map — reuse from Phase 1
- [ ] `homeassistant_floor`: Create, Read, Update, Delete, Import
- [ ] `homeassistant_area`: Create, Read, Update, Delete, Import; expose `area_id` as
      computed (HA slugifies the name)
- [ ] `homeassistant_label`: Create, Read, Update, Delete, Import
- [ ] `homeassistant_entity_customization`: Read (entity_registry/get), Update, Import.
      No Create/Delete — existence is determined by the integration. On destroy, send an
      update that resets all fields to `null`/defaults (not a delete command).
- [ ] `homeassistant_device_customization`: same pattern as entity customization.
- [ ] `data.homeassistant_areas`: `config/area_registry/list`
- [ ] `data.homeassistant_area`: same list, filter by name or area_id
- [ ] `data.homeassistant_devices`: `config/device_registry/list`, filter by integration/
      manufacturer/area
- [ ] `data.homeassistant_device`: same list, filter by name or device_id
- [ ] Area `temperature_entity_id` and `humidity_entity_id` are newer HA fields —
      check HA version before sending; gracefully skip if not supported
- [ ] On `for_each` over entities, handle the case where the entity doesn't exist in the
      registry (it may not have loaded yet) — surface a helpful error, not a panic
