package client

import (
	"context"
	"testing"
)

func TestLabelRegistry(t *testing.T) {
	ctx := context.Background()
	labels := Labels(fakeStore("config/label_registry", "label_id", "label_id", false))

	icon := "mdi:tag"
	created, err := labels.Create(ctx, map[string]any{"name": "Managed", "icon": &icon, "color": (*string)(nil)})
	if err != nil {
		t.Fatal(err)
	}
	if created.LabelID != "managed" || created.Icon == nil || *created.Icon != icon || created.Color != nil {
		t.Errorf("created = %+v", created)
	}

	updated, err := labels.Update(ctx, "managed", map[string]any{"name": "Managed by Tofu"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Managed by Tofu" {
		t.Errorf("updated = %+v", updated)
	}

	got, ok, err := labels.Get(ctx, "managed")
	if err != nil || !ok || got.Name != "Managed by Tofu" {
		t.Errorf("Get = %+v, %v, %v", got, ok, err)
	}
	if _, ok, _ := labels.Get(ctx, "missing"); ok {
		t.Error("Get(missing) found something")
	}

	if err := labels.Delete(ctx, "managed"); err != nil {
		t.Fatal(err)
	}
	// Deleting again: already gone is success.
	if err := labels.Delete(ctx, "managed"); err != nil {
		t.Errorf("second Delete: %v", err)
	}
}
