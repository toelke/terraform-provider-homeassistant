# Action calls; target and data go in an optional trailing object.
locals {
  # Same as { action = "light.turn_on", target = { area_id = "kitchen" }, data = { brightness_pct = 80 } }
  kitchen_on = provider::homeassistant::action("light.turn_on", {
    target = { area_id = "kitchen" }
    data   = { brightness_pct = 80 }
  })
  reload = provider::homeassistant::action("homeassistant.reload_all")
}
