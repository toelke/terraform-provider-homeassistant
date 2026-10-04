# A section with a heading card in front of its cards, used in a dashboard.
resource "homeassistant_dashboard" "home" {
  url_path = "my-home"
  title    = "My Home"
  config = {
    views = [{
      type  = "sections"
      title = "Overview"
      sections = [
        provider::homeassistant::grid_section([
          provider::homeassistant::tile_card("light.kitchen"),
          { type = "button", entity = "script.good_morning" },
        ], { heading = "Kitchen" }),
        # No heading, and a section option.
        provider::homeassistant::grid_section(
          [provider::homeassistant::tile_card("lock.front_door")],
          { column_span = 2 },
        ),
      ]
    }]
  }
}
