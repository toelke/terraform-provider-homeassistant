package resources

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// fakeRegistry mimics HA's generate_id_from_name: the ID is the slug of the name, with _2, _3, …
// appended while it is taken.
type fakeRegistry struct {
	names     map[string]string // ID → name
	renameErr error
	// createErr, if set, is returned by every create.
	createErr error
	calls     []string
}

func (f *fakeRegistry) ops() generatedID {
	return generatedID{
		kind:     "label",
		typeName: "homeassistant_label",
		create: func(_ context.Context, name string) (string, error) {
			f.calls = append(f.calls, "create "+name)
			if f.createErr != nil {
				return "", f.createErr
			}
			for _, n := range f.names {
				if n == name {
					return "", &client.WSError{Code: "invalid_info", Message: "name already in use"}
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
		delete: func(ctx context.Context, id string) error {
			f.calls = append(f.calls, "delete "+id)
			// Like the WebSocket client, a cancelled context fails the command.
			if err := ctx.Err(); err != nil {
				return err
			}
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

func TestGeneratedIDNameHintOnlyForHARejections(t *testing.T) {
	f := &fakeRegistry{names: map[string]string{}, createErr: client.ErrUnreachable}
	_, err := f.ops().Create(context.Background(), "Office", "")
	if !errors.Is(err, client.ErrUnreachable) || strings.Contains(err.Error(), "import it") {
		t.Fatalf("err = %v, want ErrUnreachable without the name hint", err)
	}

	f = &fakeRegistry{names: map[string]string{}, renameErr: context.DeadlineExceeded}
	_, err = f.ops().Create(context.Background(), "Home Office", "office")
	if !errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "import it") {
		t.Fatalf("err = %v, want the timeout without the name hint", err)
	}
}

// Ctrl-C cancels the context while the rename is in flight: the rename fails with
// context.Canceled, and the object must still be deleted again.
func TestGeneratedIDRenameCancelledStillDeletes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &fakeRegistry{names: map[string]string{}}
	ops := f.ops()
	ops.rename = func(context.Context, string, string) error {
		cancel()
		return context.Canceled
	}
	_, err := ops.Create(ctx, "Home Office", "office")
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "deleted again") {
		t.Fatalf("err = %v", err)
	}
	if len(f.names) != 0 {
		t.Errorf("left behind: %v", f.names)
	}
}

func TestGeneratedIDCollisionCancelledStillDeletes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &fakeRegistry{names: map[string]string{"living_room": "Lounge"}}
	ops := f.ops()
	create := ops.create
	ops.create = func(ctx context.Context, name string) (string, error) {
		defer cancel()
		return create(ctx, name)
	}
	if _, err := ops.Create(ctx, "Living Room", ""); err == nil || strings.Contains(err.Error(), "failed") {
		t.Fatalf("err = %v, want the collision without a failed delete", err)
	}
	if _, ok := f.names["living_room_2"]; ok {
		t.Error("living_room_2 was left behind")
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

func TestWithoutNulls(t *testing.T) {
	icon := "mdi:stairs"
	level := int64(1)
	got := withoutNulls(map[string]any{
		"icon": &icon, "color": (*string)(nil), "level": &level, "floor": (*int64)(nil),
		"aliases": []string{}, "labels": []string(nil),
	})
	want := map[string]any{"icon": &icon, "level": &level, "aliases": []string{}, "labels": []string(nil)}
	if !maps.EqualFunc(got, want, func(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) }) {
		t.Errorf("withoutNulls = %v, want %v", got, want)
	}
}
