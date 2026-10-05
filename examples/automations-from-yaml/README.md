# Automations from a directory of YAML files

Every file in `automations/` becomes one `homeassistant_automation`: `fileset` lists them,
`for_each` creates one resource per file, and the file name is the resource key. The files are
the YAML that Home Assistant's automation editor shows under "Edit in YAML", so an automation
built in the UI can be pasted in as-is; its `id:` line becomes the resource's `id`, and the rest
becomes `config`. The files are read with `templatefile`, so they can write `${e.bathroom_ceiling}`
instead of an entity ID, and `main.tf` keeps all entity IDs in one place. Because
`templatefile` interprets `${` and `%{`, a literal one in a file has to be written `$${` or `%%{`;
Jinja's `{{ }}` and `{% %}` are left alone. The automations themselves are small everyday ones: a
presence-controlled bathroom light, a subwoofer plug that follows the AV receiver, an oven child
lock that turns itself back on, and charge control for a wall tablet.
