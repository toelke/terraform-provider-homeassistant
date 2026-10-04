# Changelog

All notable changes to this provider are listed here. Versions follow semver; before v1.0,
breaking changes may land in minor versions and are marked **Breaking** (ADR-0018, ADR-0019).

## Unreleased

- Data source `homeassistant_entity`: state and attributes of one entity.
- Data source `homeassistant_template`: renders a Jinja template on the instance.
- Resource `homeassistant_floor`, with import.
- Resource `homeassistant_automation`, with import. `config` takes the automation as an object,
  e.g. from `yamldecode(file(...))`.
- Resource `homeassistant_input_boolean`, with import.
- Provider functions `tile_card`, `entities_card`, `heading_card` and `grid_section`, which build
  dashboard cards and sections for `homeassistant_dashboard`.

## 0.0.1

First pre-release.

- Provider configuration: `url`, `token`, `insecure` and `timeout`, as attributes or as
  `HOMEASSISTANT_*` environment variables.
- Data source `homeassistant_config`.
- Resource `homeassistant_label`.
- Resource `homeassistant_dashboard`.
