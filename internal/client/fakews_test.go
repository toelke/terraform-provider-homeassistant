package client

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
)

// fakeWS is a WSCommander that answers each command type with its handler. The reply goes through
// JSON, as HA's would; a json.RawMessage is sent as it is. A type without a handler fails like
// an unknown command does in HA.
type fakeWS map[string]func(params map[string]any) (any, error)

func (f fakeWS) Command(_ context.Context, typ string, params map[string]any, result any) error {
	h, ok := f[typ]
	if !ok {
		return &WSError{Code: "unknown_command", Message: typ}
	}
	reply, err := h(params)
	if err != nil || result == nil {
		return err
	}
	b, ok := reply.(json.RawMessage)
	if !ok {
		if b, err = json.Marshal(reply); err != nil {
			return err
		}
	}
	return json.Unmarshal(b, result)
}

// fixed returns a handler that always replies with the JSON document js.
func fixed(js string) func(map[string]any) (any, error) {
	return func(map[string]any) (any, error) { return json.RawMessage(js), nil }
}

// fakeStore returns a fakeWS that serves `<prefix>/list`, `/create`, `/update`, and `/delete`
// from memory, like an HA registry or helper collection. Items are keyed by their field id;
// commands name an item by the parameter idParam. Create derives the ID from the name. Update
// merges the parameters into the item, or replaces the item if replace is set, as HA does for
// helpers.
func fakeStore(prefix, id, idParam string, replace bool) fakeWS {
	var items []map[string]any
	find := func(params map[string]any) int {
		return slices.IndexFunc(items, func(it map[string]any) bool { return it[id] == params[idParam] })
	}
	notFound := &WSError{Code: "not_found", Message: "Unable to find " + idParam}
	return fakeWS{
		prefix + "/list": func(map[string]any) (any, error) { return items, nil },
		prefix + "/create": func(params map[string]any) (any, error) {
			it := maps.Clone(params)
			it[id] = Slugify(params["name"].(string))
			items = append(items, it)
			return it, nil
		},
		prefix + "/update": func(params map[string]any) (any, error) {
			i := find(params)
			if i < 0 {
				return nil, notFound
			}
			if replace {
				items[i] = map[string]any{id: items[i][id]}
			}
			maps.Copy(items[i], params)
			if idParam != id {
				delete(items[i], idParam)
			}
			return items[i], nil
		},
		prefix + "/delete": func(params map[string]any) (any, error) {
			i := find(params)
			if i < 0 {
				return nil, notFound
			}
			items = slices.Delete(items, i, i+1)
			return nil, nil
		},
	}
}
