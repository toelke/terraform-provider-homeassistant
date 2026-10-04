package client

// Helper is an input helper as HA stores it in its collection, e.g.
// `{"id": "guest_mode", "name": "Guest mode", "initial": false}`. Optional fields that are not set
// are absent.
type Helper map[string]any

// HelperID returns the helper's ID.
func (h Helper) HelperID() string {
	id, _ := h["id"].(string)
	return id
}

// Helpers returns the collection of the helper domain, e.g. `input_boolean`
// (`<domain>/list|create|update|delete`, naming the helper by `<domain>_id`). An update replaces
// every field, so it must carry the full helper, including `name`.
func Helpers(ws WSCommander, domain string) Registry[Helper] {
	return Registry[Helper]{
		ws:     ws,
		prefix: domain,
		key:    domain + "_id",
		idOf:   Helper.HelperID,
	}
}
