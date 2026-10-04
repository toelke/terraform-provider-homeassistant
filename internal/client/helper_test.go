package client

import (
	"context"
	"encoding/json"
	"maps"
	"testing"
)

// fakeHelpers is a WSCommander that serves `input_boolean/*` from memory, replacing the whole
// helper on update like HA does.
type fakeHelpers struct {
	items []map[string]any
}

func (f *fakeHelpers) Command(_ context.Context, typ string, params map[string]any, result any) error {
	var reply any
	switch typ {
	case "input_boolean/list":
		reply = f.items
	case "input_boolean/create":
		h := maps.Clone(params)
		h["id"] = Slugify(params["name"].(string))
		f.items = append(f.items, h)
		reply = h
	case "input_boolean/update":
		for i, h := range f.items {
			if h["id"] == params["input_boolean_id"] {
				n := maps.Clone(params)
				delete(n, "input_boolean_id")
				n["id"] = h["id"]
				f.items[i] = n
				reply = n
			}
		}
	case "input_boolean/delete":
		for i, h := range f.items {
			if h["id"] == params["input_boolean_id"] {
				f.items = append(f.items[:i], f.items[i+1:]...)
				return nil
			}
		}
		return &WSError{Code: "not_found", Message: "Unable to find input_boolean_id"}
	}
	if result == nil {
		return nil
	}
	b, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, result)
}

func TestHelpers(t *testing.T) {
	ctx := context.Background()
	helpers := Helpers(&fakeHelpers{}, "input_boolean")

	created, err := helpers.Create(ctx, map[string]any{"name": "Guest Mode", "icon": "mdi:account"})
	if err != nil {
		t.Fatal(err)
	}
	if created.HelperID() != "guest_mode" || created["icon"] != "mdi:account" {
		t.Errorf("created = %v", created)
	}

	if _, err := helpers.Update(ctx, "guest_mode", map[string]any{"name": "Guests", "initial": true}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := helpers.Get(ctx, "guest_mode")
	if err != nil || !ok {
		t.Fatalf("Get = %v, %v, %v", got, ok, err)
	}
	if got["name"] != "Guests" || got["initial"] != true || got["icon"] != nil {
		t.Errorf("after update = %v", got)
	}

	if err := helpers.Delete(ctx, "guest_mode"); err != nil {
		t.Fatal(err)
	}
	if err := helpers.Delete(ctx, "guest_mode"); err != nil {
		t.Errorf("second Delete: %v", err)
	}
}
