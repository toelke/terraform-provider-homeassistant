# Entity lookups instead of hard-coded entity IDs

This configuration builds one map, `local.entity`, from a lookup key to an entity ID, and the
rest of a configuration uses only the keys. Most entities are found with the
`homeassistant_entities` data source by domain, area and a glob on their friendly name; a
`postcondition` fails the plan with a readable message when a lookup finds no entity or several,
e.g. after a device was renamed. Entities that cannot be told apart by name are pinned by ID with
`homeassistant_entity`, which still fails the plan if they disappear. Zigbee entities whose only
name is their IEEE address get a proper name through `homeassistant_entity_settings`, which
manages just that one field. A `for` expression generates the lookups for eight identically
named bulbs, a `homeassistant_device` lookup provides a device ID for device-targeted actions,
and a small automation built with the `time_trigger` and `action` provider functions shows the
map in use.
