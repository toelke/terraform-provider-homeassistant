#!/usr/bin/env bash
# The scripted terminal session of docs/demo.cast. Record it against a fresh ./up.sh with
#
#   asciinema rec --cols 100 --rows 32 --idle-time-limit 2 -c ./cast.sh ../../docs/demo.cast
#   agg --theme monokai --font-size 14 --last-frame-duration 6 ../../docs/demo.cast ../../docs/images/demo.gif
#
# Run `tofu init` first. The session works on a copy of this directory, so the edits it makes
# leave the example unchanged.
set -euo pipefail

work=$(mktemp -d)
cd "$(dirname "$0")"
cp -r main.tf automations "$work/"
if [[ -d .terraform ]]; then cp -r .terraform .terraform.lock.hcl "$work/"; fi
cd "$work"

# Off camera: apply the example once, so the session starts with a configured Home Assistant.
tofu apply -auto-approve -input=false >/dev/null
clear

say() {
  printf '\e[1;32m# %s\e[0m\n' "$1"
  sleep 1.5
}

run() {
  printf '\e[1m$\e[0m '
  local i
  for ((i = 0; i < ${#1}; i++)); do
    printf '%s' "${1:i:1}"
    sleep 0.04
  done
  sleep 0.5
  echo
  eval "$1"
  sleep 2.5
}

say "A Home Assistant, managed with OpenTofu. This is already applied:"
run "tofu state list"

say "Automations are YAML files, as copied out of the automation editor:"
run "cat automations/sunset_lights.yaml"

say "Now someone changes the brightness to 100 % in the Home Assistant UI ..."
# The request the automation editor sends when it saves.
sed 's/brightness_pct: 70/brightness_pct: 100/' automations/sunset_lights.yaml >edited.yaml
curl -sf -X POST -H "Authorization: Bearer $HOMEASSISTANT_TOKEN" \
  -d "$(echo 'jsonencode(yamldecode(file("edited.yaml")))' | tofu console | jq -r .)" \
  "$HOMEASSISTANT_URL/api/config/automation/config/sunset_lights" >/dev/null
sleep 1

say "... and tofu plan shows the drift:"
run "tofu plan -concise"

say "tofu apply puts it back:"
run "tofu apply -auto-approve -concise"

say "Both dashboards share the quick actions section. Add a thermostat to it:"
sed -i 's|    provider::homeassistant::tile_card("lock.front_door"),|&\n    provider::homeassistant::tile_card("climate.hvac"),|' main.tf
run "grep -A6 '  quick_actions =' main.tf"
run "tofu apply -auto-approve -concise | grep -E 'will be|Apply complete'"
sleep 3

cd / && rm -rf "$work"
