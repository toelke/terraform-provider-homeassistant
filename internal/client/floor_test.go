package client

import (
	"context"
	"testing"
)

func TestFloorsDecode(t *testing.T) {
	floors := Floors(fakeWS{"config/floor_registry/list": fixed(`[
		{"floor_id": "ground_floor", "name": "Ground Floor", "level": 0, "icon": "mdi:home-floor-0",
		 "aliases": ["EG"], "created_at": 0, "modified_at": 0},
		{"floor_id": "attic", "name": "Attic", "level": null, "icon": null, "aliases": []}
	]`)})

	g, ok, err := floors.Get(context.Background(), "ground_floor")
	if err != nil || !ok {
		t.Fatalf("Get(ground_floor) = %+v, %v, %v", g, ok, err)
	}
	if g.Name != "Ground Floor" || g.Level == nil || *g.Level != 0 || g.Icon == nil ||
		*g.Icon != "mdi:home-floor-0" || len(g.Aliases) != 1 || g.Aliases[0] != "EG" {
		t.Errorf("ground_floor = %+v", g)
	}

	a, ok, err := floors.Get(context.Background(), "attic")
	if err != nil || !ok {
		t.Fatalf("Get(attic) = %+v, %v, %v", a, ok, err)
	}
	if a.Level != nil || a.Icon != nil || len(a.Aliases) != 0 {
		t.Errorf("attic = %+v", a)
	}
}
