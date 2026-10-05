package client

import (
	"context"
	"encoding/json"
	"testing"
)

// fakeEntityRegistry is a WSCommander that serves `config/entity_registry/get` and `/update` for
// one entity.
type fakeEntityRegistry struct {
	entry  map[string]any
	params map[string]any
}

func (f *fakeEntityRegistry) Command(_ context.Context, typ string, params map[string]any, result any) error {
	if params["entity_id"] != f.entry["entity_id"] {
		return &WSError{Code: "not_found", Message: "Entity not found"}
	}
	var reply any
	switch typ {
	case "config/entity_registry/get":
		reply = f.entry
	case "config/entity_registry/update":
		f.params = params
		for k, v := range params {
			f.entry[k] = v
		}
		reply = map[string]any{"entity_entry": f.entry}
	}
	b, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, result)
}

func TestEntityRegistry(t *testing.T) {
	ctx := context.Background()
	f := &fakeEntityRegistry{entry: map[string]any{
		"entity_id": "sensor.moon_phase", "platform": "moon", "name": nil, "labels": []string{},
		"hidden_by": "integration",
	}}
	entities := Entities(f)

	got, ok, err := entities.Get(ctx, "sensor.moon_phase")
	if err != nil || !ok || got.Platform != "moon" || got.Name != nil || *got.HiddenBy != "integration" {
		t.Errorf("Get = %+v, %v, %v", got, ok, err)
	}
	if _, ok, err := entities.Get(ctx, "sensor.missing"); ok || err != nil {
		t.Errorf("Get(missing) = %v, %v; want not found and no error", ok, err)
	}

	updated, err := entities.Update(ctx, "sensor.moon_phase", map[string]any{"name": "Moon"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name == nil || *updated.Name != "Moon" {
		t.Errorf("updated = %+v", updated)
	}
	if f.params["entity_id"] != "sensor.moon_phase" {
		t.Errorf("update params = %v", f.params)
	}
}
