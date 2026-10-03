# OpenTofu Provider `homeassistant` — Central Reference

This document is the authoritative overview of the provider's architecture, transport
layer, naming conventions, and API surface. Each phase of development has its own
document (`01-phase-*.md` through `05-phase-*.md`).

---

## Provider Configuration

```hcl
terraform {
  required_providers {
    homeassistant = {
      source  = "your-org/homeassistant"
      version = "~> 0.1"
    }
  }
}

provider "homeassistant" {
  url   = "http://homeassistant.local:8123"
  token = var.ha_token       # Long-Lived Access Token
}
```

Environment variable equivalents: `HOMEASSISTANT_URL`, `HOMEASSISTANT_TOKEN`.

---

## Transport Layers

The provider uses two transports. Both authenticate with the same Long-Lived Access Token.

### REST (`/api/*`)

Standard HTTP with `Authorization: Bearer <token>`. Stateless, one request per operation.
Used by Phase 0 data sources and Phase 3 automation/script/helper resources.

```
GET  /api/states                          → all entity states
GET  /api/states/<entity_id>             → single entity state
POST /api/template                        → render Jinja2 template
GET  /api/config                          → HA instance metadata
GET  /api/config/automation/config/<id>  → read automation
POST /api/config/automation/config/<id>  → create / update automation
DEL  /api/config/automation/config/<id>  → delete automation
# same pattern for script, scene, input_boolean, input_number, input_text,
# input_select, input_datetime, input_button
```

### WebSocket (`/api/websocket`)

Persistent connection; messages are JSON objects with an auto-incrementing `id`.
Used by Phase 1 (config flows / integrations), Phase 2 (registries).

**Auth handshake** (must complete before any commands):
```json
← { "type": "auth_required" }
→ { "type": "auth", "access_token": "TOKEN" }
← { "type": "auth_ok" }
```

**Command envelope:**
```json
→ { "id": 1, "type": "<command>", ...params }
← { "id": 1, "type": "result", "success": true, "result": { ... } }
```

**Lovelace / dashboard commands (Phase 4):**
```json
{ "type": "lovelace/dashboards/list" }
{ "type": "lovelace/dashboards/create", "url_path": "home", "title": "My Home",
  "icon": "mdi:home", "show_in_sidebar": true, "require_admin": false }
{ "type": "lovelace/config/save", "url_path": "home", "config": { "views": [...] } }
{ "type": "lovelace/config", "url_path": "home" }
{ "type": "lovelace/dashboards/delete", "dashboard_id": "home" }
```

The provider should open **one persistent WebSocket connection per provider instance**
and multiplex all commands over it with a sequenced id counter. Reconnect automatically
on drop. Use a goroutine per in-flight command keyed by id to correlate responses.

---

## Resource & Data Source Inventory

### Data Sources

| Resource | Transport | Phase | Description |
|---|---|---|---|
| `data.homeassistant_entities` | REST + template | 0 | Filter entities by domain/area/label/state |
| `data.homeassistant_entity` | REST | 0 | Single entity state + attributes |
| `data.homeassistant_template` | REST | 0 | Render arbitrary Jinja2 |
| `data.homeassistant_config` | REST | 0 | HA instance metadata |
| `data.homeassistant_area` | WebSocket | 2 | Single area by name or id |
| `data.homeassistant_areas` | WebSocket | 2 | All areas |
| `data.homeassistant_device` | WebSocket | 2 | Device by name/model/manufacturer |
| `data.homeassistant_devices` | WebSocket | 2 | Filter devices |
| `data.homeassistant_integration` | WebSocket | 1 | Config entry by domain/title |

### Resources

| Resource | Transport | Phase | Description |
|---|---|---|---|
| `homeassistant_esphome` | WebSocket | 1 | ESPHome config entry |
| `homeassistant_mqtt` | WebSocket | 1 | MQTT broker config entry |
| `homeassistant_hue_bridge` | WebSocket | 1 | Philips Hue bridge config entry |
| `homeassistant_zwave_js` | WebSocket | 1 | Z-Wave JS config entry |
| `homeassistant_generic_hygrostat` | WebSocket | 1 | Generic hygrostat helper integration |
| `homeassistant_integration` | WebSocket | 1 | Escape hatch — any integration/domain |
| `homeassistant_area` | WebSocket | 2 | Area (room/zone) |
| `homeassistant_floor` | WebSocket | 2 | Floor (building level) |
| `homeassistant_label` | WebSocket | 2 | Label (cross-cutting tag) |
| `homeassistant_entity_customization` | WebSocket | 2 | Entity registry metadata |
| `homeassistant_device_customization` | WebSocket | 2 | Device registry metadata |
| `homeassistant_automation` | REST | 3 | Automation |
| `homeassistant_script` | REST | 3 | Script |
| `homeassistant_scene` | REST | 3 | Scene |
| `homeassistant_input_boolean` | REST | 3 | Toggle helper |
| `homeassistant_input_number` | REST | 3 | Number helper |
| `homeassistant_input_text` | REST | 3 | Text helper |
| `homeassistant_input_select` | REST | 3 | Dropdown helper |
| `homeassistant_input_datetime` | REST | 3 | Date/time helper |
| `homeassistant_input_button` | REST | 3 | Button helper |
| `homeassistant_dashboard_fragment` | local | 4 | Reusable section of cards (pure computation) |
| `homeassistant_dashboard` | WebSocket | 4 | Full Lovelace dashboard (storage-mode via WS) |

---

## Naming & ID Conventions

**Config entry IDs** (Phase 1): HA assigns a random hex UUID on creation, e.g.
`a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4`. This becomes the Terraform resource ID. Import
uses this ID: `tofu import homeassistant_esphome.bedroom a1b2c3...`.

**Automation/script/helper IDs** (Phase 3): You supply the ID as part of the resource.
Convention: lowercase alphanumeric + underscores. Stored in TF state and used as the
`/api/config/<type>/config/<id>` path segment.

**Registry IDs** (Phase 2):
- Areas: HA generates a slug from the name, e.g. `bedroom`. Stable across renames.
- Labels/Floors: HA-assigned IDs similar to config entries.
- Entities: `entity_id` is the canonical ID (`light.bedroom_ceiling`).
- Devices: HA-assigned UUID.

---

## Config Flow State Machine (Phase 1 internals)

Config flows are the mechanism by which integrations are set up. Every typed resource
in Phase 1 drives this same state machine:

```
Init flow
  → step: "user"       (most integrations, submit host/credentials)
  → step: "confirm"    (some integrations ask for confirmation)
  → step: "discovery_confirm" (if device was auto-discovered)
  → result: "create_entry"  ✓  or  "abort" / "error"

Options flow (for reconfiguration / tofu apply with changed args)
  → step: "init"
  → step: "user" or integration-specific
  → result: "create_entry"
```

The provider must handle the full step loop — keep submitting until `type` is
`"create_entry"` or `"abort"`. For typed resources, steps and their data schemas are
hardcoded. For `homeassistant_integration` (generic), steps and data are passed
verbatim from the config.

---

## Provider Code Structure (Go)

```
internal/
  provider/
    provider.go          # provider schema, configure(), shared client
  client/
    rest.go              # REST client (HTTP, retries, error parsing)
    websocket.go         # WS client (connect, auth, mux, reconnect)
    config_flow.go       # Config flow driver (init → step loop → result)
  datasources/
    entities.go
    entity.go
    template.go
    config.go
    area.go              # Phase 2
    device.go            # Phase 2
  resources/
    # Phase 1
    esphome.go
    mqtt.go
    hue_bridge.go
    zwave_js.go
    integration.go       # generic escape hatch
    # Phase 2
    area.go
    floor.go
    label.go
    entity_customization.go
    device_customization.go
    # Phase 3
    automation.go
    script.go
    scene.go
    input_boolean.go
    input_number.go
    input_text.go
    input_select.go
    input_datetime.go
    input_button.go
    # Phase 4
    dashboard_fragment.go
    dashboard.go
```

**Shared client** — passed via `provider.Configure()` to all resources and data sources:

```go
type HAClient struct {
    REST *RESTClient
    WS   *WSClient
}
```

Both clients are initialised in `provider.go` from the provider schema config and stored
in the resource `Configure` method via the standard `resource.ConfigureRequest`.

---

## Error Handling Conventions

- **404 on GET** → resource was deleted outside Terraform → signal `d.SetId("")` to
  mark as needing re-create.
- **WS `"success": false`** → surface `result.error.message` as a Terraform diagnostic.
- **Config flow `"abort"`** → map common abort reasons to readable messages:
  - `"already_configured"` → suggest `tofu import`
  - `"cannot_connect"` → host unreachable, surface as error
  - `"invalid_auth"` → bad credentials
- **REST 401** → invalid/expired token, fail fast with clear message.
- All resources must handle the case where HA is temporarily unreachable during plan
  by returning the last known state rather than erroring (use `terraform plan -refresh=false`
  as the escape hatch guidance in docs).

---

## Import Support

Every resource must implement `ImportState`. Pattern:

```go
func (r *EsphomeResource) ImportState(
    ctx context.Context,
    req resource.ImportStateRequest,
    resp *resource.ImportStateResponse,
) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

Then `Read` fetches the full state from HA using the imported ID. Document the import
ID format in the resource schema description.

---

## Testing Strategy

- **Unit tests**: pure Go, test YAML generation (Phase 4), config flow step logic (Phase 1).
- **Acceptance tests**: use `TF_ACC=1` against a real HA instance (or a dockerised HA
  in CI). Each resource needs at least: create, read, update, delete, import.
- **Mocking**: for unit tests of the WS client, use a mock WS server that replays
  canned responses. The `config_flow.go` driver should accept an interface, not a
  concrete type, so it can be tested without a real HA.
