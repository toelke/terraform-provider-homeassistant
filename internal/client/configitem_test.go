package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestConfigItemsGetStripsID(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/config/automation/config/away" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"away","alias":"Away","triggers":[]}`))
	})
	got, err := NewAutomations(c).Get(t.Context(), "away")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"alias":"Away","triggers":[]}` {
		t.Errorf("config = %s", got)
	}
}

func TestConfigItemsGetMissingIsNotFound(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Resource not found"}`))
	})
	if _, err := NewAutomations(c).Get(t.Context(), "away"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestConfigItemsSaveAddsID(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/config/automation/config/away" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != `{"alias":"Away","id":"away"}` {
			t.Errorf("body = %s", b)
		}
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	})
	if err := NewAutomations(c).Save(t.Context(), "away", json.RawMessage(`{"alias":"Away"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestConfigItemsSaveRejectsNonObject(t *testing.T) {
	c := serve(t, func(http.ResponseWriter, *http.Request) { t.Error("request sent") })
	if err := NewAutomations(c).Save(t.Context(), "away", json.RawMessage(`[1]`)); err == nil {
		t.Error("no error for an array config")
	}
}

func TestConfigItemsDelete(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/config/automation/config/away" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	})
	if err := NewAutomations(c).Delete(t.Context(), "away"); err != nil {
		t.Fatal(err)
	}
}

func TestConfigItemsFindEntityWaitsForReload(t *testing.T) {
	var calls atomic.Int32
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/states" {
			t.Errorf("path = %s", r.URL.Path)
		}
		// Before the reload: only a script with the same id, and an entity with a numeric id.
		states := `[{"entity_id":"script.away","attributes":{"id":"away"}},
			{"entity_id":"sensor.x","attributes":{"id":5}}]`
		if calls.Add(1) >= 3 {
			states = `[{"entity_id":"script.away","attributes":{"id":"away"}},
				{"entity_id":"automation.leaving_home","attributes":{"id":"away"}}]`
		}
		_, _ = w.Write([]byte(states))
	})
	got, err := NewAutomations(c).FindEntity(t.Context(), "away", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got != "automation.leaving_home" {
		t.Errorf("entity = %q", got)
	}
	if n := calls.Load(); n != 3 {
		t.Errorf("polled %d times, want 3", n)
	}
}

func TestConfigItemsFindEntityGivesUp(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})
	_, err := NewAutomations(c).FindEntity(t.Context(), "away", 0)
	if !errors.Is(err, ErrEntityNotFound) {
		t.Errorf("err = %v, want ErrEntityNotFound", err)
	}
}
