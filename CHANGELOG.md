# Changelog

All notable changes to this provider are listed here. Versions follow semver; before v1.0,
breaking changes may land in minor versions and are marked **Breaking** (ADR-0018, ADR-0019).

## Unreleased

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

## 0.0.1

First pre-release.

- Provider configuration: `url`, `token`, `insecure` and `timeout`, as attributes or as
  `HOMEASSISTANT_*` environment variables.
- Data source `homeassistant_config`.
- Resource `homeassistant_label`.
- Resource `homeassistant_dashboard`.
