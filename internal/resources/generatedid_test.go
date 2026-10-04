package resources

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fakeRegistry mimics HA's generate_id_from_name: the ID is the slug of the name, with _2, _3, …
// appended while it is taken.
type fakeRegistry struct {
	names     map[string]string // ID → name
	renameErr error
	calls     []string
}

func (f *fakeRegistry) ops() generatedID {
	return generatedID{
		kind:     "label",
		typeName: "homeassistant_label",
		create: func(_ context.Context, name string) (string, error) {
			f.calls = append(f.calls, "create "+name)
			for _, n := range f.names {
				if n == name {
					return "", errors.New("name already in use")
				}
			}
			base := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
			id := base
			for i := 2; f.names[id] != ""; i++ {
				id = base + "_" + string(rune('0'+i))
			}
			f.names[id] = name
			return id, nil
		},
		rename: func(_ context.Context, id, name string) error {
			f.calls = append(f.calls, "rename "+id+" "+name)
			if f.renameErr != nil {
				return f.renameErr
			}
			f.names[id] = name
			return nil
		},
		delete: func(_ context.Context, id string) error {
			f.calls = append(f.calls, "delete "+id)
			delete(f.names, id)
			return nil
		},
	}
}

func TestGeneratedIDFromName(t *testing.T) {
	f := &fakeRegistry{names: map[string]string{}}
	id, err := f.ops().Create(context.Background(), "Living Room", "")
	if err != nil {
		t.Fatal(err)
	}
	if id != "living_room" {
		t.Errorf("id = %q, want living_room", id)
	}
	if want := []string{"create Living Room"}; !slices.Equal(f.calls, want) {
		t.Errorf("calls = %q, want %q", f.calls, want)
	}
}

func TestGeneratedIDCreateThenRename(t *testing.T) {
	f := &fakeRegistry{names: map[string]string{}}
	id, err := f.ops().Create(context.Background(), "Managed by Tofu", "managed")
	if err != nil {
		t.Fatal(err)
	}
	if id != "managed" || f.names["managed"] != "Managed by Tofu" {
		t.Errorf("id = %q, names = %v", id, f.names)
	}
	if want := []string{"create managed", "rename managed Managed by Tofu"}; !slices.Equal(f.calls, want) {
		t.Errorf("calls = %q, want %q", f.calls, want)
	}
}

func TestGeneratedIDCollisionDeletes(t *testing.T) {
	// The existing label was renamed, so its name is free but its ID is not.
	f := &fakeRegistry{names: map[string]string{"living_room": "Lounge"}}
	_, err := f.ops().Create(context.Background(), "Living Room", "")
	if err == nil {
		t.Fatal("want an error")
	}
	for _, want := range []string{`"living_room"`, `"living_room_2"`, "tofu import homeassistant_label.<name> living_room"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
	if _, ok := f.names["living_room_2"]; ok {
		t.Error("living_room_2 was left behind")
	}
}

func TestGeneratedIDRenameFailureDeletes(t *testing.T) {
	f := &fakeRegistry{names: map[string]string{}, renameErr: errors.New("name already in use")}
	_, err := f.ops().Create(context.Background(), "Taken", "mine")
	if err == nil || !strings.Contains(err.Error(), "name already in use") {
		t.Fatalf("err = %v", err)
	}
	if len(f.names) != 0 {
		t.Errorf("left behind: %v", f.names)
	}
}

func TestGeneratedIDNameInUse(t *testing.T) {
	f := &fakeRegistry{names: map[string]string{"x": "Taken"}}
	_, err := f.ops().Create(context.Background(), "Taken", "")
	if err == nil || !strings.Contains(err.Error(), "import it") {
		t.Fatalf("err = %v", err)
	}
}

func TestSlugValidator(t *testing.T) {
	for in, valid := range map[string]bool{
		"managed":         true,
		"living_room_2":   true,
		"Managed":         false,
		"living-room":     false,
		"_x":              false,
		"a__b":            false,
		"":                false,
		"managed_by_tofu": true,
	} {
		resp := &validator.StringResponse{}
		slugValidator{}.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("id"),
			ConfigValue: types.StringValue(in),
		}, resp)
		if got := !resp.Diagnostics.HasError(); got != valid {
			t.Errorf("%q: valid = %v, want %v", in, got, valid)
		}
	}
}
