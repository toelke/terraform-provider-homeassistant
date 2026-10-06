package client

import (
	"context"
	"testing"
)

func TestAreasDecode(t *testing.T) {
	areas := Areas(fakeWS{"config/area_registry/list": fixed(`[
		{"area_id": "kitchen", "name": "Kitchen", "floor_id": "ground_floor", "icon": "mdi:stove",
		 "aliases": ["Cookhouse"], "labels": ["managed"], "picture": "/api/image/serve/abc/512x512",
		 "temperature_entity_id": "sensor.kitchen_temperature",
		 "humidity_entity_id": "sensor.kitchen_humidity", "created_at": 0, "modified_at": 0},
		{"area_id": "hall", "name": "Hall", "floor_id": null, "icon": null, "aliases": [],
		 "labels": [], "picture": null, "temperature_entity_id": null, "humidity_entity_id": null}
	]`)})

	k, ok, err := areas.Get(context.Background(), "kitchen")
	if err != nil || !ok {
		t.Fatalf("Get(kitchen) = %+v, %v, %v", k, ok, err)
	}
	if k.Name != "Kitchen" || k.FloorID == nil || *k.FloorID != "ground_floor" || k.Icon == nil ||
		*k.Icon != "mdi:stove" || len(k.Aliases) != 1 || k.Aliases[0] != "Cookhouse" ||
		len(k.Labels) != 1 || k.Labels[0] != "managed" || k.Picture == nil ||
		*k.Picture != "/api/image/serve/abc/512x512" || k.TemperatureEntityID == nil ||
		*k.TemperatureEntityID != "sensor.kitchen_temperature" || k.HumidityEntityID == nil ||
		*k.HumidityEntityID != "sensor.kitchen_humidity" {
		t.Errorf("kitchen = %+v", k)
	}

	h, ok, err := areas.Get(context.Background(), "hall")
	if err != nil || !ok {
		t.Fatalf("Get(hall) = %+v, %v, %v", h, ok, err)
	}
	if h.FloorID != nil || h.Icon != nil || len(h.Aliases) != 0 || len(h.Labels) != 0 ||
		h.Picture != nil || h.TemperatureEntityID != nil || h.HumidityEntityID != nil {
		t.Errorf("hall = %+v", h)
	}
}
