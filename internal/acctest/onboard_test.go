package acctest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// fakeHA mimics the onboarding endpoints of a fresh Home Assistant and records the calls.
type fakeHA struct {
	t        *testing.T
	clientID string

	mu    sync.Mutex
	calls []string
}

func (f *fakeHA) record(call string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
}

func (f *fakeHA) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/onboarding/users":
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			f.t.Errorf("users: %v", err)
		}
		if body["client_id"] != f.clientID || body["username"] == "" || body["password"] == "" {
			f.t.Errorf("users: unexpected body %v", body)
		}
		f.record("users")
		writeJSON(w, map[string]string{"auth_code": "code-1"})
	case "/auth/token":
		if err := r.ParseForm(); err != nil {
			f.t.Errorf("token: %v", err)
		}
		if r.PostForm.Get("grant_type") != "authorization_code" || r.PostForm.Get("code") != "code-1" ||
			r.PostForm.Get("client_id") != f.clientID {
			f.t.Errorf("token: unexpected form %v", r.PostForm)
		}
		f.record("token")
		writeJSON(w, map[string]any{"access_token": "short-lived", "token_type": "Bearer"})
	case "/api/onboarding/core_config", "/api/onboarding/analytics", "/api/onboarding/integration":
		if got := r.Header.Get("Authorization"); got != "Bearer short-lived" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		f.record(strings.TrimPrefix(r.URL.Path, "/api/onboarding/"))
		writeJSON(w, map[string]any{})
	case "/api/websocket":
		f.serveWS(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeHA) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		f.t.Errorf("ws accept: %v", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()
	ctx := r.Context()

	_ = wsjson.Write(ctx, conn, map[string]string{"type": "auth_required"})
	var auth map[string]string
	if err := wsjson.Read(ctx, conn, &auth); err != nil {
		return
	}
	if auth["access_token"] != "short-lived" {
		_ = wsjson.Write(ctx, conn, map[string]string{"type": "auth_invalid", "message": "bad token"})
		return
	}
	_ = wsjson.Write(ctx, conn, map[string]string{"type": "auth_ok"})

	var cmd map[string]any
	if err := wsjson.Read(ctx, conn, &cmd); err != nil {
		return
	}
	if cmd["type"] != "auth/long_lived_access_token" || cmd["client_name"] == "" || cmd["lifespan"] == nil {
		f.t.Errorf("ws: unexpected command %v", cmd)
	}
	f.record("long_lived_access_token")
	_ = wsjson.Write(ctx, conn, map[string]any{
		"id": cmd["id"], "type": "result", "success": true, "result": "long-lived",
	})
	_, _, _ = conn.Read(ctx) // wait for the client to close
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func TestOnboard(t *testing.T) {
	fake := &fakeHA{t: t}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	fake.clientID = srv.URL + "/"

	token, err := onboard(t.Context(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	if token != "long-lived" {
		t.Errorf("token = %q, want the long-lived token", token)
	}
	want := []string{"users", "token", "core_config", "analytics", "integration", "long_lived_access_token"}
	if !slices.Equal(fake.calls, want) {
		t.Errorf("calls = %v, want %v", fake.calls, want)
	}
}

func TestOnboard_ReportsFailingStep(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "User step already done", http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := onboard(t.Context(), srv.URL)

	if err == nil || !strings.Contains(err.Error(), "create owner user") || !strings.Contains(err.Error(), "403") {
		t.Errorf("err = %v, want it to name the step and the status", err)
	}
}
