package client

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

// fakeConfigEntries is a WSCommander that answers `config_entries/get` with a fixed reply, as HA
// sends it, and records the params.
type fakeConfigEntries struct {
	reply  string
	params map[string]any
}

func (f *fakeConfigEntries) Command(_ context.Context, typ string, params map[string]any, result any) error {
	if typ != "config_entries/get" {
		return &WSError{Code: "unknown_command", Message: typ}
	}
	f.params = params
	return json.Unmarshal([]byte(f.reply), result)
}

func TestConfigEntriesGet(t *testing.T) {
	ws := &fakeConfigEntries{reply: `[
		{"entry_id":"E1","domain":"local_calendar","title":"Chores","source":"user","state":"loaded",
		 "supports_options":false,"disabled_by":null,"reason":null},
		{"entry_id":"E2","domain":"local_calendar","title":"Trash","source":"user","state":"not_loaded",
		 "disabled_by":"user"}
	]`}
	entries := NewConfigEntries(ws, nil)

	got, ok, err := entries.Get(t.Context(), "local_calendar", "E2")
	if err != nil || !ok {
		t.Fatalf("Get = %v, %v", ok, err)
	}
	user := "user"
	want := ConfigEntry{EntryID: "E2", Domain: "local_calendar", Title: "Trash", State: "not_loaded", DisabledBy: &user}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v, want %+v", got, want)
	}
	if want := map[string]any{"domain": "local_calendar"}; !reflect.DeepEqual(ws.params, want) {
		t.Errorf("params = %v", ws.params)
	}

	if _, ok, err := entries.Get(t.Context(), "local_calendar", "E3"); err != nil || ok {
		t.Errorf("Get(E3) = %v, %v; want not found", ok, err)
	}
}

func TestConfigEntriesDelete(t *testing.T) {
	var deleted []string
	rest := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		deleted = append(deleted, r.URL.Path)
		if r.URL.Path == "/api/config/config_entries/entry/gone" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Invalid entry specified"}`))
			return
		}
		_, _ = w.Write([]byte(`{"require_restart":false}`))
	})
	entries := NewConfigEntries(nil, rest)

	if err := entries.Delete(t.Context(), "E1"); err != nil {
		t.Errorf("Delete(E1) = %v", err)
	}
	// An entry that is already gone is not an error.
	if err := entries.Delete(t.Context(), "gone"); err != nil {
		t.Errorf("Delete(gone) = %v", err)
	}
	want := []string{"/api/config/config_entries/entry/E1", "/api/config/config_entries/entry/gone"}
	if !reflect.DeepEqual(deleted, want) {
		t.Errorf("deleted = %v", deleted)
	}
}
