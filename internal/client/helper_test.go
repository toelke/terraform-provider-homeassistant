package client

import (
	"context"
	"testing"
)

func TestHelpers(t *testing.T) {
	ctx := context.Background()
	helpers := Helpers(fakeStore("input_boolean", "id", "input_boolean_id", true), "input_boolean")

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
