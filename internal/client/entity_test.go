package client

import (
	"context"
	"maps"
	"testing"
)

func TestEntityRegistry(t *testing.T) {
	ctx := context.Background()
	entry := map[string]any{
		"entity_id": "sensor.moon_phase", "platform": "moon", "name": nil, "labels": []string{},
		"hidden_by": "integration",
	}
	notFound := &WSError{Code: "not_found", Message: "Entity not found"}
	var updateParams map[string]any
	entities := Entities(fakeWS{
		"config/entity_registry/get": func(params map[string]any) (any, error) {
			if params["entity_id"] != entry["entity_id"] {
				return nil, notFound
			}
			return entry, nil
		},
		"config/entity_registry/update": func(params map[string]any) (any, error) {
			if params["entity_id"] != entry["entity_id"] {
				return nil, notFound
			}
			updateParams = params
			maps.Copy(entry, params)
			return map[string]any{"entity_entry": entry}, nil
		},
	})

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
	if updateParams["entity_id"] != "sensor.moon_phase" {
		t.Errorf("update params = %v", updateParams)
	}
}
