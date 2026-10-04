# Integrations (config entries)

Config flows are REST. Reading config entries is WebSocket. A config entry's input data cannot be
read back (ADR-0014).

## Config-flow driver

```
ConfigFlows.Run(ctx, handler string, entryID string /* "" or reconfigure target */, steps []FlowStep) (entryID, error)

POST /api/config/config_entries/flow            {"handler": h, "show_advanced_options": true[, "entry_id": id]}
loop on result.type:
  "form"         → if result.errors is non-empty → abort + error with the errors
                   find the step with result.step_id (else abort + error);
                   a step already submitted once → abort + error (no loops)
                   POST /api/config/config_entries/flow/<flow_id>   step.data ∪ step.sensitive_data
                   (on a HTTP error, e.g. 400 for data not matching the schema → abort + error)
  "create_entry" → return result.result.entry_id
  "abort"        → reconfigure (entryID set) with reason "reconfigure_successful" → return entryID;
                   otherwise error with result.reason (HA already ended the flow, no DELETE)
  other          → abort + error (unsupported: external steps, progress, menus)
```

"abort" in the right-hand column means `DELETE /api/config/config_entries/flow/<flow_id>`. It is
sent even if the context was cancelled. Steps are matched by `step_id`, not by position. On a key
present in both maps, `sensitive_data` wins.

Unit-tested against a fake HTTP server, with no real HA.

## `homeassistant_integration` (generic)

```hcl
resource "homeassistant_integration" "shelly" {
  domain = "shelly"
  steps = {
    user = { host = "192.168.1.50" }
  }
}

resource "homeassistant_integration" "cloud" {
  domain = "my_cloud"
  steps = {
    user = { username = "me@example.com" }
  }
  sensitive_steps = {
    user = { password = var.cloud_password }
  }
}
```

| Argument | Type | |
|---|---|---|
| `domain` | string | required, forces replacement |
| `steps` | object: `step_id` → object of fields (dynamic) | required, at least one step; forces replacement |
| `sensitive_steps` | same shape as `steps`, sensitive | optional; forces replacement |

The steps are two dynamic objects keyed by `step_id` (ADR-0021, which supersedes ADR-0014's
`step` blocks). The driver matches steps by `step_id`, so order carries no meaning, and a
`step_id` cannot appear twice. A step's driver input is `steps[id]` as `data` and
`sensitive_steps[id]` as `sensitive_data`. A step may appear in either object or in both; the two
are merged per step, and on a field present in both, `sensitive_steps` wins. A form without
fields is answered by `{}`.

| Computed | |
|---|---|
| `id` | `entry_id` |
| `title`, `state`, `disabled_by` | from `config_entries/get` |

- **Create:** run the config-flow driver, then read the new entry for the computed attributes.
- **Read:** `config_entries/get` filtered by `domain`, matched on `entry_id`. If gone → remove
  from state. Inputs are never compared.
- **Delete:** `DELETE /api/config/config_entries/entry/<id>`. An entry that is already gone is
  not an error.
- **Import:** by `entry_id`, read across all domains. The steps cannot be recovered, so after
  import they are null and the user writes them. The next plan is an in-place update that only
  records them in state; nothing is sent to HA. After that, changing them forces replacement as
  usual. (A null `steps` in state marks the import, since `steps` is required otherwise.)

## `data.homeassistant_integration`

Arguments: `domain` (required) and `title` (optional). It must match exactly one entry; no match
and several matches are errors that list the candidates.
Computed: `id`/`entry_id`, `title`, `state`, `disabled_by`.

## Later: typed integrations

`homeassistant_esphome` (`host`, `port`, `noise_psk` sensitive) and `homeassistant_mqtt` (broker
fields, TLS, discovery, birth/will). They use the same driver with fixed steps. Updates use a
reconfigure flow (`entry_id` passed at init) where the integration implements one; otherwise the
change forces replacement. Hue and Z-Wave JS are not planned. Exact step schemas must be read
from each integration's `config_flow.py` in the oldest release of the support window.
