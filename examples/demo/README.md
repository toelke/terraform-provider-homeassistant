# Demo

A self-contained configuration for a fresh Home Assistant in Docker. It shows:

- **Automations from YAML files**, as copied out of the automation editor
  (`automations/*.yaml`, one `homeassistant_automation` each). Change one in the UI, and
  `tofu plan` shows the drift; `tofu apply` restores it.
- **Areas and entity settings:** the three areas onboarding creates are imported, two more are
  created, and each of the demo integration's lights is put into its area.
- **Dashboards:** "Home" has one view per area, built with `for` over `homeassistant_entities`.
  A quick-actions section, defined once as a local, is on every view of "Home" and on "Wall
  Tablet".

It needs Docker and OpenTofu ≥ 1.8.

```sh
eval "$(./up.sh)"    # start and onboard Home Assistant; HA_VERSION=2026.4.4 for another version
tofu init
tofu apply
```

Open <http://localhost:8123> and log in as `demo` / `demo-password`. Edit the automation
"Living room lights at sunset" and run `tofu plan`. `./down.sh` removes the container.

| File | |
|---|---|
| `main.tf` | the configuration |
| `automations/` | one automation per file; the file name is its ID |
| `config/` | the Home Assistant configuration: a fresh install's, plus the `demo` integration |
| `up.sh`, `onboard.py`, `down.sh` | start and onboard Home Assistant (the steps of `internal/acctest`), and stop it |
| `cast.sh` | the scripted session recorded as `docs/demo.cast` |
| `screenshots.py` | takes the README's dashboard screenshots |
