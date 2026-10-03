# Phase 1 — Integration Config Entries

**Goal:** Provision physical integrations — ESPHome devices, MQTT brokers, Hue bridges,
Z-Wave controllers, etc. — so that devices and entities appear in HA automatically.
This is the layer that makes things *exist* before anything else references them.

**Transport:** WebSocket only (config flow API). One persistent connection per provider
instance.

---

## How Config Flows Work

Every integration in HA is set up through a config flow. The flow is a wizard of named
steps; each step has a schema (what fields to fill in) and a result (next step, or done).

```
Init:   { "type": "config_entries/flow/init", "handler": "esphome" }
        ← { "flow_id": "abc", "step_id": "user",
             "data_schema": [{ "name": "host", "type": "string" }] }

Submit: { "type": "config_entries/flow", "flow_id": "abc",
          "data": { "host": "192.168.1.100" } }
        ← { "type": "create_entry", "entry_id": "xyz", "title": "esp-bedroom" }

-- or if more steps --
        ← { "step_id": "confirm", ... }
Submit: { "type": "config_entries/flow", "flow_id": "abc", "data": {} }
        ← { "type": "create_entry", "entry_id": "xyz" }
```

For **reconfiguration** (host changed, credentials rotated):
```
Init:   { "type": "config_entries/options/flow/init", "entry_id": "xyz" }
Submit: (same loop as above)
```

For **deletion** (`tofu destroy`):
```
        { "type": "config_entries/delete", "entry_id": "xyz" }
```

For **reading** current state:
```
        { "type": "config_entries/get" }
        ← [ { "entry_id": "xyz", "domain": "esphome", "title": "esp-bedroom",
              "state": "loaded", "disabled_by": null, ... } ]
```

### The `config_flow.go` driver

All typed resources share a single driver. It should not be written per-resource:

```go
// internal/client/config_flow.go

type FlowStep struct {
    StepID string
    Data   map[string]interface{}
}

// RunFlow drives a config flow to completion.
// steps is an ordered list of (stepID → data) pairs. The driver submits each
// step's data when HA asks for that step_id. Returns the created entry_id.
func (ws *WSClient) RunFlow(ctx context.Context, handler string, steps []FlowStep) (string, error) {
    // 1. Send config_entries/flow/init
    // 2. Loop:
    //    a. If result type == "create_entry" → return entry_id
    //    b. If result type == "abort"        → return error with reason
    //    c. If result type == "form"         → find matching step in steps slice, submit data
    // 3. Error if unknown result type or step not found in steps
}

// RunOptionsFlow runs the options (reconfigure) flow for an existing entry.
func (ws *WSClient) RunOptionsFlow(ctx context.Context, entryID string, steps []FlowStep) error
```

Each typed resource calls `RunFlow` with its known step list. The generic escape hatch
passes the user-supplied steps verbatim.

---

## Typed Resources

### `homeassistant_esphome`

ESPHome devices connect via the Native API (TCP, optionally encrypted). The config flow
has a single "user" step that takes `host` (IP or mDNS hostname). HA then connects to
the device and discovers all entities automatically.

```hcl
resource "homeassistant_esphome" "bedroom_sensor" {
  host = "192.168.1.100"
  # or host = "esp-bedroom.local"

  # Optional: noise encryption key (from esphome api: encryption: key: ...)
  noise_psk = var.esp_bedroom_key
}

resource "homeassistant_esphome" "kitchen_display" {
  host = "192.168.1.101"
}
```

**Arguments:**

| Name | Type | Required | Description |
|---|---|---|---|
| `host` | string | yes | IP address or mDNS hostname |
| `noise_psk` | string, sensitive | no | Noise protocol encryption key (base64) |

**Computed:**

| Name | Type | Description |
|---|---|---|
| `id` | string | Config entry ID (HA-assigned UUID) |
| `title` | string | Device name as discovered by HA |
| `device_id` | string | Device registry ID (use in Phase 2 customization) |

**Flow steps:**
```go
steps := []FlowStep{
    {StepID: "user", Data: map[string]interface{}{
        "host":      host,
        "noise_psk": noisePsk, // omit key if empty string
    }},
}
```

**Import:**
```bash
tofu import homeassistant_esphome.bedroom_sensor <config_entry_id>
```

---

### `homeassistant_mqtt`

Sets up the MQTT integration (one per broker). After this resource is applied, any MQTT
discovery messages published to the broker will automatically create entities in HA.

```hcl
resource "homeassistant_mqtt" "broker" {
  broker   = "mqtt.home.local"
  port     = 1883
  username = "homeassistant"
  password = var.mqtt_password

  # Optional
  protocol          = "3.1.1"  # or "5"
  tls               = false
  tls_insecure      = false
  discovery         = true
  discovery_prefix  = "homeassistant"
  birth_message = {
    topic   = "homeassistant/status"
    payload = "online"
    retain  = true
    qos     = 0
  }
  will_message = {
    topic   = "homeassistant/status"
    payload = "offline"
    retain  = true
    qos     = 0
  }
}
```

**Arguments:**

| Name | Type | Required | Description |
|---|---|---|---|
| `broker` | string | yes | Hostname or IP of MQTT broker |
| `port` | number | no | Default `1883` (`8883` for TLS) |
| `username` | string | no | |
| `password` | string, sensitive | no | |
| `protocol` | string | no | `"3.1.1"` or `"5"` |
| `tls` | bool | no | Enable TLS |
| `tls_insecure` | bool | no | Skip certificate verification |
| `discovery` | bool | no | Default `true` |
| `discovery_prefix` | string | no | Default `"homeassistant"` |
| `birth_message` | object | no | MQTT birth message config |
| `will_message` | object | no | MQTT will message config |

**Flow steps:** MQTT's flow has a "broker" step (connectivity) and optionally a
"certificate" step (TLS). The driver should skip the certificate step if `tls = false`.

```go
steps := []FlowStep{
    {StepID: "broker", Data: map[string]interface{}{
        "broker":   broker,
        "port":     port,
        "username": username,
        "password": password,
        "protocol": protocol,
        "tls":      tls,
    }},
}
```

---

### `homeassistant_hue_bridge`

```hcl
resource "homeassistant_hue_bridge" "living_room" {
  host = "192.168.1.20"

  # Hue requires pressing the link button on the bridge during first setup.
  # The provider will retry for up to `link_button_timeout` seconds.
  link_button_timeout = 60
}
```

**Arguments:**

| Name | Type | Required | Description |
|---|---|---|---|
| `host` | string | yes | Bridge IP |
| `link_button_timeout` | number | no | Seconds to wait for link button press. Default 30. |

**Flow steps:** Hue has a "link" step that requires the physical button press. The
provider must poll with retries until the user presses it or the timeout expires,
surfacing a clear error message if it times out.

```go
// Hue flow: init → "link" step (press button) → create_entry
// The "link" step is submitted with an empty data map; HA returns "link_button"
// error until the button is pressed. Retry with backoff up to timeout.
```

---

### `homeassistant_zwave_js`

Z-Wave JS connects to a USB controller via a serial port or a Z-Wave JS server.

```hcl
# Using a local Z-Wave JS server (recommended for docker/VM setups)
resource "homeassistant_zwave_js" "controller" {
  url = "ws://zwave-js-server:3000"
}

# Using the Z-Wave JS add-on (direct USB)
resource "homeassistant_zwave_js" "controller" {
  url = "ws://supervisor/zwave_js/ws"
}
```

**Arguments:**

| Name | Type | Required | Description |
|---|---|---|---|
| `url` | string | yes | WebSocket URL of the Z-Wave JS server |

---

### `homeassistant_integration` — Generic escape hatch

For any integration not covered by a typed resource. You supply the domain and the raw
step data. The provider drives the flow using the supplied steps in order.

```hcl
# Set up Shelly integration (not yet typed)
resource "homeassistant_integration" "shelly" {
  domain = "shelly"

  # Steps are submitted in order. step_id must match what HA returns.
  step {
    step_id = "user"
    data = {
      host = "192.168.1.50"
    }
  }
}

# Set up a cloud integration that has two steps
resource "homeassistant_integration" "my_cloud_service" {
  domain = "my_cloud"

  step {
    step_id = "user"
    data = {
      username = "me@example.com"
      password = var.cloud_password
    }
  }

  step {
    step_id = "mfa"
    data = {
      code = var.mfa_code
    }
  }
}
```

**Arguments:**

| Name | Type | Required | Description |
|---|---|---|---|
| `domain` | string | yes | Integration domain, e.g. `"shelly"` |
| `step` | block (multiple) | yes | Ordered steps to submit |
| `step.step_id` | string | yes | Must match the `step_id` returned by HA |
| `step.data` | `map(string)` | yes | Fields for this step |

**Note:** The generic resource does not support reconfiguration via options flow
(since the options schema varies per integration and is unknown). For integrations that
need options changes, use `tofu taint` and re-create, or switch to a typed resource.

---

## Data Source: `data.homeassistant_integration`

Read an existing config entry — useful when an integration was set up manually and you
want to reference its `entry_id` or the entity/device IDs it created.

```hcl
data "homeassistant_integration" "existing_mqtt" {
  domain = "mqtt"
}

data "homeassistant_integration" "hue_by_title" {
  domain = "hue"
  title  = "Philips Hue"  # disambiguate if multiple entries for the same domain
}
```

**Computed:**

| Name | Type | Description |
|---|---|---|
| `id` / `entry_id` | string | Config entry UUID |
| `title` | string | Human-readable name |
| `state` | string | `"loaded"`, `"setup_error"`, `"not_loaded"`, etc. |
| `disabled_by` | string | `null` if enabled |

---

## Full Phase 1 Example

```hcl
provider "homeassistant" {
  url   = "http://homeassistant.local:8123"
  token = var.ha_token
}

# ── MQTT broker ─────────────────────────────────────────────────────────────

resource "homeassistant_mqtt" "broker" {
  broker   = "mosquitto.home.local"
  port     = 1883
  username = "ha"
  password = var.mqtt_password
}

# ── ESPHome devices ─────────────────────────────────────────────────────────
# These reference IPs from a separate variables file or Terraform workspace.
# After apply, HA connects to each device and discovers its entities.

locals {
  esphome_devices = {
    bedroom_sensor  = "192.168.1.100"
    kitchen_display = "192.168.1.101"
    garage_door     = "192.168.1.102"
    front_motion    = "192.168.1.103"
  }
}

resource "homeassistant_esphome" "devices" {
  for_each  = local.esphome_devices
  host      = each.value
  noise_psk = var.esphome_keys[each.key]
}

# ── Hue bridge ───────────────────────────────────────────────────────────────

resource "homeassistant_hue_bridge" "main" {
  host                = "192.168.1.20"
  link_button_timeout = 60
}

# ── Z-Wave ───────────────────────────────────────────────────────────────────

resource "homeassistant_zwave_js" "controller" {
  url = "ws://zwave-js:3000"
}

# ── Outputs ──────────────────────────────────────────────────────────────────

output "esphome_entry_ids" {
  value = { for k, v in homeassistant_esphome.devices : k => v.id }
}
```

After `tofu apply`, all ESPHome devices are connected and their entities (sensors,
switches, lights, etc.) appear in HA based on each device's YAML configuration. Phase 2
can then assign those entities and devices to areas.

---

## Implementation Checklist

- [ ] `WSClient`: connect, auth handshake, message mux by id, reconnect on drop
- [ ] `WSClient.Send(ctx, msg)` → `(result map, error)` — blocks until response arrives
- [ ] `RunFlow(handler, steps)` → entry_id — handles multi-step, abort, retry for Hue
- [ ] `RunOptionsFlow(entryID, steps)` → error
- [ ] `GetConfigEntries()` → `[]ConfigEntry`
- [ ] `DeleteConfigEntry(entryID)` → error
- [ ] `homeassistant_esphome`: Create, Read, Update (options flow with new host), Delete, Import
- [ ] `homeassistant_mqtt`: Create, Read, Update, Delete, Import
- [ ] `homeassistant_hue_bridge`: Create with retry loop for link button, Read, Delete, Import
- [ ] `homeassistant_zwave_js`: Create, Read, Delete, Import
- [ ] `homeassistant_integration`: Create (generic steps), Read, Delete, Import
      (Update is intentionally not supported — document this)
- [ ] `data.homeassistant_integration`: Read by domain + optional title filter
- [ ] All resources: store entry_id as the Terraform resource ID
- [ ] Acceptance tests: requires a real HA instance; at minimum test ESPHome and generic
