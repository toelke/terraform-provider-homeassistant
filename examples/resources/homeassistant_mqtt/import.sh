# Import the MQTT config entry by its entry_id. The inputs cannot be read back: write them into
# the configuration afterwards, and the next apply records them without touching the entry.
tofu import homeassistant_mqtt.broker 01JABCDEF0123456789ABCDEFG
