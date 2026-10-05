#!/usr/bin/env bash
# The scripted terminal session of docs/demo.cast. Record it against a fresh ./up.sh with
#
#   asciinema rec -f asciicast-v2 --window-size 100x32 --idle-time-limit 2 -c ./cast.sh ../../docs/demo.cast
#   agg --theme monokai --font-size 14 --last-frame-duration 6 ../../docs/demo.cast ../../docs/images/demo.gif
#
# Run `tofu init` first, against a Home Assistant that ./up.sh has just started: the session
# begins with the first apply. It works on a copy of this directory, so the edits it makes leave
# the example unchanged.
set -euo pipefail

work=$(mktemp -d)
cd "$(dirname "$0")"
cp -r main.tf automations templates "$work/"
if [[ -d .terraform ]]; then cp -r .terraform .terraform.lock.hcl "$work/"; fi
cd "$work"
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

ha() {
  curl -sf -H "Authorization: Bearer $HOMEASSISTANT_TOKEN" "$@"
}

say "A freshly onboarded Home Assistant. One apply sets up areas, automations and dashboards:"
run "tofu apply -auto-approve -concise | grep 'Apply complete'"

say "Every area gets the same automation, written once as a template:"
run "cat templates/lights_left_on.yaml.tftpl"
run "tofu state list | grep lights_left_on"

say "Someone changes the kitchen's copy to 4 hours in the Home Assistant UI ..."
# The request the automation editor sends when it saves.
url=$HOMEASSISTANT_URL/api/config/automation/config/lights_left_on_kitchen
ha "$url" | jq -c '.triggers[0].for.hours = 4' | ha -X POST -d @- "$url" >/dev/null
sleep 1

say "... tofu plan shows the drift, and apply makes it match the others again:"
run "tofu plan -concise | sed -n '/will be/,/Plan:/p'"
run "tofu apply -auto-approve -concise | grep 'Apply complete'"

say "One edit to the template changes every area:"
sed -i 's/hours: 2/hours: 1/; s/two hours/an hour/' templates/lights_left_on.yaml.tftpl
run "tofu apply -auto-approve -concise | grep -E 'will be|Apply complete'"

say "Both dashboards share the quick actions section. Add a thermostat to it:"
sed -i 's|    provider::homeassistant::tile_card("lock.front_door"),|&\n    provider::homeassistant::tile_card("climate.hvac"),|' main.tf
run "grep -A6 '  quick_actions =' main.tf"
run "tofu apply -auto-approve -concise | grep -E 'will be|Apply complete'"
sleep 3

cd / && rm -rf "$work"
