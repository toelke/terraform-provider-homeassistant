# A tile card; further card options go in an optional trailing object.
locals {
  kitchen_tile = provider::homeassistant::tile_card("light.kitchen", {
    name     = "Kitchen"
    features = [{ type = "light-brightness" }]
  })
  # Same as { type = "tile", entity = "lock.front_door" }
  door_tile = provider::homeassistant::tile_card("lock.front_door")
}
