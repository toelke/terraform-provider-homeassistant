# Examples

- [`demo/`](demo/): a complete demo against a fresh Home Assistant in Docker, recorded in
  [`docs/demo.cast`](../docs/demo.cast).
- `provider/`, `resources/`, `data-sources/`, `functions/`: the snippets that the generated
  documentation in [`docs/`](../docs/) shows.

## Use cases

Small, self-contained configurations taken from a real home managed with this provider, with
names and identifiers replaced. Each directory has a `main.tf`, the files it reads, and a
`README.md`. Set `HOMEASSISTANT_URL` and `HOMEASSISTANT_TOKEN` (or the `ha_url` and `ha_token`
variables) before running `tofu plan`; the entity IDs are examples and need adapting to your
instance.

- [`automations-from-yaml/`](automations-from-yaml/): one automation per YAML file in a directory, pasted from the automation editor.
- [`entity-lookups/`](entity-lookups/): find entity IDs by domain, area and name, with a postcondition that catches renamed devices.
- [`floors-areas-labels/`](floors-areas-labels/): floors, areas and labels from maps, and a label used to filter a dashboard.
- [`automation-templates/`](automation-templates/): one `.tftpl` template, many automations, including blueprint instances with optional inputs.
- [`remotes-and-scenes/`](remotes-and-scenes/): Zigbee remotes via MQTT and blueprints, switching scenes from YAML files, with a helper as a lock.
- [`solar-and-energy/`](solar-and-energy/): washing machine and dishwasher driven by solar power and grid export, with once-a-day helper flags.
- [`computer-power-button/`](computer-power-button/): dashboard buttons that start one parameterised script per computer.
- [`presence-notifications/`](presence-notifications/): per-person arrival messages, voice announcements, and find-my-phone scripts.
- [`room-dashboards/`](room-dashboards/): dashboards generated from a directory of views, with shared cards and navigation buttons.
- [`covers-and-sun/`](covers-and-sun/): sun-driven blinds and awning written in HCL with the builder functions.
