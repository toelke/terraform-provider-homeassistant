package client

// Area is an entry of the area registry.
type Area struct {
	AreaID              string   `json:"area_id"`
	Name                string   `json:"name"`
	FloorID             *string  `json:"floor_id"`
	Icon                *string  `json:"icon"`
	Aliases             []string `json:"aliases"`
	Labels              []string `json:"labels"`
	Picture             *string  `json:"picture"`
	TemperatureEntityID *string  `json:"temperature_entity_id"`
	HumidityEntityID    *string  `json:"humidity_entity_id"`
}

// Areas returns the area registry (`config/area_registry/*`).
func Areas(ws WSCommander) Registry[Area] {
	return Registry[Area]{
		ws:     ws,
		prefix: "config/area_registry",
		key:    "area_id",
		idOf:   func(a Area) string { return a.AreaID },
	}
}
