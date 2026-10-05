# Changelog

All notable changes to this provider are listed here. Versions follow semver; before v1.0,
breaking changes may land in minor versions and are marked **Breaking** (ADR-0018, ADR-0019).

## Unreleased

- Data source `homeassistant_entities`: new `hidden` filter, and a `hidden` flag on each entry of
  `entities`, so lookups can skip hidden duplicates (#30).
- `data.homeassistant_entities`: new `device_id` filter, for telling apart entities with the same
  friendly name on different devices (#30).
- `examples/demo/`: a demo against a fresh Home Assistant in Docker (one automation template
  for every area, automations from YAML files, drift, dashboards per area), recorded in
  `docs/demo.cast`.
- `examples/`: ten use-case configurations taken from a real home (automations from YAML files
  and templates, entity lookups, floors, areas and labels, scenes and remotes, solar- and
  presence-driven automations, generated dashboards), indexed in `examples/README.md`.

## 0.1.0

First release with the v0.1 feature set.

- Data source `homeassistant_entity`: state and attributes of one entity.
- Data source `homeassistant_template`: renders a Jinja template on the instance.
- Data source `homeassistant_entities`: lists entities, filtered by domain, area, label, device
  class, or a glob on the friendly name.
- Resource `homeassistant_floor`, with import.
- Resource `homeassistant_automation`, with import. `config` takes the automation as an object,
  e.g. from `yamldecode(file(...))`.
- Resource `homeassistant_input_boolean`, with import.
- Provider functions `tile_card`, `entities_card`, `heading_card` and `grid_section`, which build
  dashboard cards and sections for `homeassistant_dashboard`.
- Provider functions `state_trigger`, `numeric_state_trigger`, `time_trigger`, `state_condition`,
  `time_condition`, `action` and `delay`, which build automation and script steps.
- Resource `homeassistant_area`, with import.
- Resource `homeassistant_script`, with import. `config` takes the script as an object, e.g. from
  `yamldecode(file(...))`; `entity_id` is `script.<id>`.
- Resource `homeassistant_scene`, with import. `config` takes the scene as an object, e.g. from
  `yamldecode(file(...))`.
- Resources `homeassistant_input_number`, `homeassistant_input_text`, `homeassistant_input_select`,
  `homeassistant_input_datetime` and `homeassistant_input_button`, with import and plan-time
  validation.
- Resource `homeassistant_integration`, with import: creates a config entry by answering its
  config flow with `steps`, with secrets in `sensitive_steps`.
- Data source `homeassistant_integration`: one config entry by domain and optional title.
- Data sources `homeassistant_areas` and `homeassistant_area` (by ID or name).
- Data sources `homeassistant_devices`, filtered by integration, manufacturer, model, area, or
  label, and `homeassistant_device` (by ID or name).
- Resource `homeassistant_entity_settings`, with import: name, icon, area, labels, hidden and
  disabled of an entity an integration owns. Only the configured attributes are managed and,
  on destroy, reset.
- Resource `homeassistant_device_settings`, with import: name, area, labels and disabled of a
  device an integration owns, with the same field-granular semantics as
  `homeassistant_entity_settings`.
- Resource `homeassistant_esphome`, with import: adds an ESPHome device by `host`, with
  `noise_psk`, and sets `allow_service_calls` and `subscribe_logs`. Changing `host`, `port` or
  `password` reconfigures the entry in place.
- Resource `homeassistant_mqtt`, with import: the MQTT broker connection, with TLS and
  WebSockets, plus discovery and birth and will messages. Broker changes reconfigure the entry in
  place, and options go through the options flow.

## 0.0.1

First pre-release.

- Provider configuration: `url`, `token`, `insecure` and `timeout`, as attributes or as
  `HOMEASSISTANT_*` environment variables.
- Data source `homeassistant_config`.
- Resource `homeassistant_label`.
- Resource `homeassistant_dashboard`.
