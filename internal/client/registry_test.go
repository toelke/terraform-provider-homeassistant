package client

import (
	"context"
	"encoding/json"
	"testing"
)

// fakeLabelRegistry is a WSCommander that serves `config/label_registry/*` from memory.
type fakeLabelRegistry struct {
	labels []map[string]any
	cmds   []string
}

func (f *fakeLabelRegistry) Command(_ context.Context, typ string, params map[string]any, result any) error {
	f.cmds = append(f.cmds, typ)
	var reply any
	switch typ {
	case "config/label_registry/list":
		reply = f.labels
	case "config/label_registry/create":
		l := map[string]any{"label_id": Slugify(params["name"].(string))}
		for k, v := range params {
			l[k] = v
		}
		f.labels = append(f.labels, l)
		reply = l
	case "config/label_registry/update":
		for _, l := range f.labels {
			if l["label_id"] == params["label_id"] {
				for k, v := range params {
					l[k] = v
				}
				reply = l
			}
		}
	case "config/label_registry/delete":
		for i, l := range f.labels {
			if l["label_id"] == params["label_id"] {
				f.labels = append(f.labels[:i], f.labels[i+1:]...)
				return nil
			}
		}
		return &WSError{Code: "invalid_info", Message: "Label ID doesn't exist"}
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

func TestLabelRegistry(t *testing.T) {
	ctx := context.Background()
	f := &fakeLabelRegistry{}
	labels := Labels(f)

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
