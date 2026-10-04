# Registries: floors, areas, labels, entity and device settings

All of these use the WebSocket transport. The hierarchy is: Floor → Area → Device → Entity.
Labels apply across all of them. Devices and entities are created by integrations. The provider
may create the config entry behind them ([integrations.md](integrations.md)), and resources such
as helpers and automations bring their own entities, but this file only *edits* registry entries.

## Generated-ID resources (ADR-0009)

Floor, area, and label share this behaviour:

- `id` is optional and computed. If it is unset, HA assigns `slugify(name)`.
- **Create:**
  1. If `id` is set, validate it as a slug (`slugify(id) == id`) and create with `name = id`.
  2. Otherwise create with `name`.
  3. Check that the returned ID equals the expected one (`id`, or the provider's own
     `slugify(name)`, which must match HA's slugify). On a mismatch, delete the object again and
     fail. The message names the clashing ID and the ways out.
  4. If `id` is set, `update` the object to `name`. If that fails, delete the object again and
     fail.

  HA itself rejects a create or rename to a name that another object already uses (compared
  case- and whitespace-insensitively). The provider surfaces that error with the same ways out.
- **Slugify:** `client.Slugify` ports HA's `util.slugify` (python-slugify with `_` as
  separator). It transliterates with go-unidecode, corrected by a table of the code points where
  HA's text-unidecode differs. `scripts/gen-slug-tables.sh` regenerates that table from an HA
  image. The table stays under text-unidecode's Artistic License, not MPL-2.0; the script writes
  the license header, and `THIRD_PARTY_NOTICES.md` holds the attribution.
- **Update:** in place, including `name`. Changing `id` forces replacement.
- **Read:** list and find by ID. If missing → remove from state.
- **Import:** by ID.

### `homeassistant_floor`

Commands: `config/floor_registry/list|create|update|delete` (key `floor_id`).

| Argument | Type | |
|---|---|---|
| `name` | string | required |
| `id` | string | optional, computed |
| `level` | number | optional; an integer, as HA requires |
| `icon` | string | optional |
| `aliases` | set(string) | optional; defaults to `[]`, which is what HA reports for none |

### `homeassistant_area`

Commands: `config/area_registry/list|create|update|delete` (key `area_id`).

| Argument | Type | |
|---|---|---|
| `name` | string | required |
| `id` | string | optional, computed |
| `floor_id` | string | optional; changing it moves the area in place |
| `icon` | string | optional |
| `aliases` | set(string) | optional; defaults to `[]`, which is what HA reports for none |
| `labels` | set(string) | optional; label IDs; defaults to `[]`, as for `aliases` |
| `picture` | string | optional |
| `temperature_entity_id` | string | optional; HA requires an existing `sensor` with device class `temperature` |
| `humidity_entity_id` | string | optional; HA requires an existing `sensor` with device class `humidity` |

### `homeassistant_label`

Commands: `config/label_registry/list|create|update|delete` (key `label_id`).

| Argument | Type | |
|---|---|---|
| `name` | string | required |
| `id` | string | optional, computed |
| `color` | string | optional; `#RRGGBB` or a theme color such as `indigo` |
| `icon` | string | optional |
| `description` | string | optional |

## Settings resources (ADR-0010)

These edit registry entries owned by integrations. They are **field-granular**:

- Only attributes set in config are sent, read into state, or compared.
- **Create:** poll until the entry exists (`timeouts.create`, default 60s), then send an
  `update` with the set fields.
- **Update:** send the changed fields. A field removed from config is reset to its default.
- **Delete:** reset the configured fields to defaults (`null`, or `[]` for labels). If the entry
  is gone, do nothing.
- **Read:** if the entry is missing → remove from state.

### `homeassistant_entity_settings`

Commands: `config/entity_registry/get`, `config/entity_registry/update`.
ID and import ID: `entity_id`.

```hcl
resource "homeassistant_entity_settings" "ceiling" {
  entity_id = "light.bedroom_ceiling"
  name      = "Ceiling Light"
  area_id   = homeassistant_area.bedroom.id
  labels    = [homeassistant_label.managed.id]
}
```

| Argument | Type | Reset value |
|---|---|---|
| `entity_id` | string, required, forces replacement | — |
| `name` | string | `null` (back to the integration's name) |
| `icon` | string | `null` |
| `area_id` | string | `null` (inherits the device's area) |
| `labels` | set(string) | `[]` |
| `hidden` | bool → `hidden_by: "user"` or `null` | `null` |
| `disabled` | bool → `disabled_by: "user"` or `null` | `null` |

`hidden_by` and `disabled_by` can also be set by integrations. The provider only writes `"user"`
or `null`, and maps the read-back value `"user"` to `true` and anything else to `false`.

| Computed | |
|---|---|
| `platform` | integration that owns the entity |
| `device_id` | |
| `original_name` | name before the override |

### `homeassistant_device_settings`

Commands: `config/device_registry/list` (filtered by ID), `config/device_registry/update`.
ID and import ID: `device_id`.

| Argument | Type | Reset value |
|---|---|---|
| `device_id` | string, required, forces replacement | — |
| `name_by_user` | string | `null` |
| `area_id` | string | `null` |
| `labels` | set(string) | `[]` |
| `disabled` | bool | `null` |

| Computed |
|---|
| `name`, `manufacturer`, `model`, `sw_version`, `hw_version`, `identifiers` (list(list(string))), `config_entries` (list(string)) |

## Data sources

- `homeassistant_areas` — `config/area_registry/list`. `areas` is a list of objects with
  `area_id`, `name`, `floor_id`, `icon`, `aliases`, `labels`, sorted by `area_id`.
- `homeassistant_area` — the same list, matched by exactly one of `area_id` or `name` (compared
  exactly). Zero or multiple matches is an error.
- `homeassistant_devices` — `config/device_registry/list`. Filters, all compared exactly:
  `integration` (resolved through the domain of the config entries from `config_entries/get`; an
  unknown domain matches nothing), `manufacturer`, `model`, `area_id`, `label` (a label ID).
  `devices` is a list of objects with `device_id`, `name`, `name_by_user`, `manufacturer`,
  `model`, `sw_version`, `area_id`, `labels`, `config_entries`, sorted by `device_id`.
- `homeassistant_device` — matched by exactly one of `device_id` or `name`, which is compared with
  `name_by_user` first, then, only if no device matches that way, with `name`. Ambiguous matches
  are an error. A configured `name` stays as given in state, even if it matched `name_by_user`.
