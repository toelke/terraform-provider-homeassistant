# Integrations (config entries)

Config flows are REST. Reading config entries is WebSocket. A config entry's input data cannot be
read back (ADR-0014).

## Config-flow driver

```
RunFlow(ctx, handler string, entryID string /* "" or reconfigure target */, steps []Step) (entryID, error)

POST /api/config/config_entries/flow            {"handler": h, "show_advanced_options": true[, "entry_id": id]}
loop on result.type:
  "form"         → find the step with result.step_id (else abort + error);
                   if result.errors is non-empty after a submit → abort + error with the errors
                   POST /api/config/config_entries/flow/<flow_id>   step.data ∪ step.sensitive_data
  "create_entry" → return result.result.entry_id
  "abort"        → error with result.reason (reconfigure success also ends as abort "reconfigure_successful")
  other          → abort + error (unsupported: external steps, progress, menus)
```

Unit-tested against a fake HTTP server, with no real HA.

## `homeassistant_integration` (generic)

```hcl
resource "homeassistant_integration" "shelly" {
  domain = "shelly"
  step {
    step_id = "user"
    data    = { host = "192.168.1.50" }
  }
}

resource "homeassistant_integration" "cloud" {
  domain = "my_cloud"
  step {
    step_id        = "user"
    data           = { username = "me@example.com" }
    sensitive_data = { password = var.cloud_password }
  }
}
```

| Argument | Type | |
|---|---|---|
| `domain` | string | required, forces replacement |
| `step` | list of blocks, min 1 | forces replacement |
| `step.step_id` | string | required |
| `step.data` | map(dynamic) | optional |
| `step.sensitive_data` | map(dynamic), sensitive | optional |

| Computed | |
|---|---|
| `id` | `entry_id` |
| `title`, `state`, `disabled_by` | from `config_entries/get` |

- **Read:** `config_entries/get` filtered by `domain`, matched on `entry_id`. If gone → remove
  from state. Inputs are never compared.
- **Delete:** `DELETE /api/config/config_entries/entry/<id>`.
- **Import:** by `entry_id`. The `step` blocks cannot be recovered, so after import the user
  writes them. Because they cannot be compared, this causes no diff and no replacement until the
  steps are edited.

## `data.homeassistant_integration`

Arguments: `domain` (required) and `title` (optional). It must match exactly one entry.
Computed: `id`/`entry_id`, `title`, `state`, `disabled_by`.

## Later: typed integrations

`homeassistant_esphome` (`host`, `port`, `noise_psk` sensitive) and `homeassistant_mqtt` (broker
fields, TLS, discovery, birth/will). They use the same driver with fixed steps. Updates use a
reconfigure flow (`entry_id` passed at init) where the integration implements one; otherwise the
change forces replacement. Hue and Z-Wave JS are not planned. Exact step schemas must be read
from each integration's `config_flow.py` in the oldest release of the support window.
