# Dashboard buttons that start a parameterised script

Computers that cannot wake on LAN are started by switching on their smart plug, and the plug
should go off again once the computer has shut down. One `homeassistant_script` handles every
computer: it takes the plug as a `fields` variable, runs in `parallel` mode, waits on the plug's
power reading with `repeat … until`, and sends a message at the end. Per computer, `for_each`
creates an `homeassistant_input_button` for the dashboard and an automation, built in HCL with
the `state_trigger` and `action` provider functions, that calls `script.turn_on` with the
computer as a variable. The script's YAML is a template: `jsonencode` turns a Tofu map of plug
names to labels into a Jinja dictionary literal, so the message names the computer properly.
