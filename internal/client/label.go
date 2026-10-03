package client

// Label is an entry of the label registry.
type Label struct {
	LabelID     string  `json:"label_id"`
	Name        string  `json:"name"`
	Color       *string `json:"color"`
	Icon        *string `json:"icon"`
	Description *string `json:"description"`
}

// Labels returns the label registry (`config/label_registry/*`).
func Labels(ws WSCommander) Registry[Label] {
	return Registry[Label]{
		ws:     ws,
		prefix: "config/label_registry",
		key:    "label_id",
		idOf:   func(l Label) string { return l.LabelID },
	}
}
