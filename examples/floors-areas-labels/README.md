# Floors, areas and labels as code

The house's layout lives in two maps: floors with their level (outdoors has none) and areas with
the floor they are on (a stairwell has none). `for_each` turns them into `homeassistant_floor` and
`homeassistant_area` resources with fixed, readable IDs, so automations can target
`area_id: kitchen` or `floor_id: upper` without surprises; a script built with the `action`
provider function does exactly that. A `homeassistant_label` marks the single bulbs of a string
light, which a one-character glob (`name_pattern = "?"`) finds, and
`homeassistant_entity_settings` puts the label on each of them. The label then works as a filter
in the `homeassistant_entities` data source: the wall-tablet dashboard lists every living-room
light except the labelled bulbs, built with the `grid_section` and `tile_card` functions.
