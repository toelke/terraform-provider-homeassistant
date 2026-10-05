# Generated dashboards

Two dashboards built from small YAML files with `templatefile`. The "Rooms" dashboard has one
subview per file in `views/`, ordered by the file name prefix, and its home view gets a
navigation button for every room, generated from each view's own `title`, `icon` and `path`, so
adding a room is adding a file. Cards used in several places are defined once: an alerts card
(a weather warning, the oven's child lock) is decoded in `main.tf` and dropped into views as
`${jsonencode(alerts_card)}`, which works because JSON is valid YAML, and the per-person markdown
cards are rendered from one `.md.tftpl` file with Jinja for the live part. `%{ for }` loops
generate repetitive cards such as the thermostat preset buttons. Both dashboards are
`homeassistant_dashboard` resources, so a change in UI edit mode shows up as drift in the next
`tofu plan`.
