# An entities card: each row is an entity ID or an entity row object.
locals {
  lights = provider::homeassistant::entities_card([
    "light.kitchen",
    { entity = "light.hall", name = "Hall" },
  ], { title = "Lights" })
}
