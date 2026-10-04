# Import a config entry by its entry_id. The steps cannot be read back: write them into the
# configuration afterwards, and the next apply records them without touching the entry.
tofu import homeassistant_integration.shelly 01JABCDEF0123456789ABCDEFG
