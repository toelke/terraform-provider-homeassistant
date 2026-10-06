package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
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

func TestConfigItemsFindEntityByAttribute(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/states" {
			t.Errorf("path = %s", r.URL.Path)
		}
		// A script with the same id, an entity with a numeric id, and the automation, whose entity
		// ID comes from its alias.
		_, _ = w.Write([]byte(`[{"entity_id":"script.away","attributes":{"id":"away"}},
			{"entity_id":"sensor.x","attributes":{"id":5}},
			{"entity_id":"automation.leaving_home","attributes":{"id":"away"}}]`))
	})
	got, err := NewAutomations(c).FindEntity(t.Context(), "away", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "automation.leaving_home" {
		t.Errorf("entity = %q", got)
	}
}

func TestConfigItemsFindEntityNotFound(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})
	_, err := NewAutomations(c).FindEntity(t.Context(), "away", "")
	if !errors.Is(err, ErrEntityNotFound) {
		t.Errorf("err = %v, want ErrEntityNotFound", err)
	}
}

// With the entity ID found before, a refresh reads only that entity's state.
func TestConfigItemsFindEntityChecksKnownFirst(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/states/automation.leaving_home" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"entity_id":"automation.leaving_home","attributes":{"id":"away"}}`))
	})
	got, err := NewAutomations(c).FindEntity(t.Context(), "away", "automation.leaving_home")
	if err != nil {
		t.Fatal(err)
	}
	if got != "automation.leaving_home" {
		t.Errorf("entity = %q", got)
	}
}

// A known entity that is gone (renamed by the user) or now belongs to another item falls back to
// all states. One in another domain is not read at all.
func TestConfigItemsFindEntityKnownStale(t *testing.T) {
	for name, tc := range map[string]struct {
		known string
		state func(http.ResponseWriter)
	}{
		"renamed": {"automation.old_name", func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Entity not found."}`))
		}},
		"other item": {"automation.old_name", func(w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{"entity_id":"automation.old_name","attributes":{"id":"other"}}`))
		}},
		"other domain": {"script.away", func(http.ResponseWriter) { t.Error("state of another domain read") }},
	} {
		t.Run(name, func(t *testing.T) {
			c := serve(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/states" {
					_, _ = w.Write([]byte(`[{"entity_id":"automation.new_name","attributes":{"id":"away"}}]`))
					return
				}
				tc.state(w)
			})
			got, err := NewAutomations(c).FindEntity(t.Context(), "away", tc.known)
			if err != nil {
				t.Fatal(err)
			}
			if got != "automation.new_name" {
				t.Errorf("entity = %q", got)
			}
		})
	}
}

func TestScriptsSaveLeavesIDOut(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/config/script/config/goodnight" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != `{"alias":"Goodnight"}` {
			t.Errorf("body = %s", b)
		}
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	})
	if err := NewScripts(c).Save(t.Context(), "goodnight", json.RawMessage(`{"alias":"Goodnight"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestScriptsFindEntityByEntityID(t *testing.T) {
	var reloaded atomic.Bool
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/states/script.goodnight" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if !reloaded.Load() {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Entity not found."}`))
			return
		}
		_, _ = w.Write([]byte(`{"entity_id":"script.goodnight","attributes":{}}`))
	})
	scripts := NewScripts(c)
	if _, err := scripts.FindEntity(t.Context(), "goodnight", ""); !errors.Is(err, ErrEntityNotFound) {
		t.Errorf("before the reload: err = %v, want ErrEntityNotFound", err)
	}
	reloaded.Store(true)
	got, err := scripts.FindEntity(t.Context(), "goodnight", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "script.goodnight" {
		t.Errorf("entity = %q", got)
	}
	if fixed, ok := scripts.FixedEntityID("goodnight"); !ok || fixed != "script.goodnight" {
		t.Errorf("FixedEntityID = %q, %v", fixed, ok)
	}
	if _, ok := NewAutomations(c).FixedEntityID("goodnight"); ok {
		t.Error("automations have a fixed entity ID")
	}
}

func TestScenesSaveAddsIDAndFindEntityByAttribute(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/config/scene/config/movie":
			b, _ := io.ReadAll(r.Body)
			if string(b) != `{"id":"movie","name":"Movie"}` {
				t.Errorf("body = %s", b)
			}
			_, _ = w.Write([]byte(`{"result":"ok"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/states":
			// An automation with the same id, and the scene, whose entity ID comes from its name.
			_, _ = w.Write([]byte(`[{"entity_id":"automation.movie","attributes":{"id":"movie"}},
				{"entity_id":"scene.movie_night","attributes":{"id":"movie"}}]`))
		default:
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
	})
	scenes := NewScenes(c)
	if err := scenes.Save(t.Context(), "movie", json.RawMessage(`{"name":"Movie"}`)); err != nil {
		t.Fatal(err)
	}
	got, err := scenes.FindEntity(t.Context(), "movie", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "scene.movie_night" {
		t.Errorf("entity = %q", got)
	}
}
