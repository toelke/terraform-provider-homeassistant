# One template, many automations

Automations that differ only in their entities are written once, as a `.yaml.tftpl` template,
and `local.templated_automations` lists the instances of each template with their variables. A
single `for_each` creates them all, keyed `<template>_<instance>`, e.g.
`climate_sync_kitchen`. The templates use `%{ for }` loops, so `light-to-switch` and
`switch-to-light` generate a trigger and a `choose` branch for every bulb of a wall switch, however
many it has; together they keep smart bulbs and the relay channels of a decoupled wall switch in
sync, in both directions. `climate-sync` keeps a room's climate entity and its radiator
thermostat at the same set-point. `cold-window` creates instances of a blueprint
(`use_blueprint`): optional blueprint inputs are passed as a map and written with `jsonencode`,
which is valid YAML, so one room can override the threshold while the others keep the blueprint's
defaults. The blueprint itself has to be imported into Home Assistant first; the provider does
not manage blueprints.
