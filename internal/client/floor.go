package client

// Floor is an entry of the floor registry.
type Floor struct {
	FloorID string   `json:"floor_id"`
	Name    string   `json:"name"`
	Level   *int64   `json:"level"`
	Icon    *string  `json:"icon"`
	Aliases []string `json:"aliases"`
}

// Floors returns the floor registry (`config/floor_registry/*`).
func Floors(ws WSCommander) Registry[Floor] {
	return Registry[Floor]{
		ws:     ws,
		prefix: "config/floor_registry",
		key:    "floor_id",
		idOf:   func(f Floor) string { return f.FloorID },
	}
}
