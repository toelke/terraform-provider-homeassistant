# Energy- and solar-driven automations

Automations that react to power readings: a suggestion to start the washing machine when the
balcony solar panels deliver enough, an automatic dishwasher start as soon as the house exports
power to the grid (with a fixed time as fallback), a to-do item when the dishwasher is done, and
the kitchen light turning on with the coffee machine. "Already done today" flags are
`homeassistant_input_boolean` helpers created in the same configuration, and the automations get
their entity IDs through `templatefile` variables, so the helpers always exist before the
automations that use them. Tofu variables reach into the YAML too: `notify_action` chooses how
messages are sent (any `notify.*` action or a `rest_command` for a messenger), and the list
`laundry_days` is written into a `time` condition with `jsonencode`, which produces valid YAML.
