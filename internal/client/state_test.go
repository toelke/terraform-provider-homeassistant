package client

import (
	"errors"
	"net/http"
	"testing"
)

func TestStateDecodesAndKeepsRawAttributes(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/states/sun.sun" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"entity_id":"sun.sun","state":"above_horizon",` +
			`"attributes":{"elevation":12.5,"rising":true},` +
			`"last_changed":"2026-10-04T10:00:00+00:00","last_updated":"2026-10-04T10:01:00+00:00"}`))
	})
	s, err := c.State(t.Context(), "sun.sun")
	if err != nil {
		t.Fatal(err)
	}
	if s.State != "above_horizon" || s.LastChanged != "2026-10-04T10:00:00+00:00" ||
		s.LastUpdated != "2026-10-04T10:01:00+00:00" {
		t.Errorf("decoded %+v", s)
	}
	if got := string(s.Attributes); got != `{"elevation":12.5,"rising":true}` {
		t.Errorf("attributes = %s", got)
	}
}

func TestStateMissingIsNotFound(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Entity not found."}`, http.StatusNotFound)
	})
	if _, err := c.State(t.Context(), "light.nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestStatesDecodesEveryEntity(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/states" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"entity_id":"sun.sun","state":"above_horizon","attributes":{"rising":true}},` +
			`{"entity_id":"light.hall","state":"off","attributes":{}}]`))
	})
	states, err := c.States(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 2 || states[0].EntityID != "sun.sun" || states[1].State != "off" ||
		string(states[0].Attributes) != `{"rising":true}` {
		t.Errorf("decoded %+v", states)
	}
}
