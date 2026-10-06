package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

// fakeConfigEditor serves the REST config editor and the states of one domain. A saved item's
// entity appears after `reloadAfter` more requests for states, or never if it is negative.
type fakeConfigEditor struct {
	t      *testing.T
	domain string

	mu          sync.Mutex
	items       map[string]string // id → config JSON, as HA returns it
	states      map[string]string // entity ID → attributes JSON
	saveStatus  int               // non-zero: the status a save answers with
	reloadAfter int
	pending     map[string]string // entity ID → attributes JSON, added on reload
	requests    []string
}

func newFakeConfigEditor(t *testing.T, domain string) *fakeConfigEditor {
	return &fakeConfigEditor{
		t: t, domain: domain,
		items: map[string]string{}, states: map[string]string{}, pending: map[string]string{},
	}
}

func (f *fakeConfigEditor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	notFound := func() {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Resource not found"}`))
	}
	if r.URL.Path == "/api/states" || strings.HasPrefix(r.URL.Path, "/api/states/") {
		if f.reloadAfter == 0 {
			for id, attrs := range f.pending {
				f.states[id] = attrs
			}
		}
		f.reloadAfter--
	}

	switch id, ok := strings.CutPrefix(r.URL.Path, "/api/config/"+f.domain+"/config/"); {
	case ok && r.Method == http.MethodGet:
		config, ok := f.items[id]
		if !ok {
			notFound()
			return
		}
		_, _ = w.Write([]byte(config))
	case ok && r.Method == http.MethodPost:
		if f.saveStatus != 0 {
			w.WriteHeader(f.saveStatus)
			_, _ = w.Write([]byte(`{"message":"Message malformed: extra keys not allowed @ data['triggers'][0]['plaform']"}`))
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			f.t.Error(err)
		}
		b, _ := json.Marshal(body)
		f.items[id] = string(b)
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	case r.URL.Path == "/api/states":
		var states []map[string]any
		for entityID, attrs := range f.states {
			states = append(states, map[string]any{"entity_id": entityID, "attributes": json.RawMessage(attrs)})
		}
		_ = json.NewEncoder(w).Encode(states)
	case strings.HasPrefix(r.URL.Path, "/api/states/"):
		entityID := strings.TrimPrefix(r.URL.Path, "/api/states/")
		attrs, ok := f.states[entityID]
		if !ok {
			notFound()
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"entity_id": entityID, "attributes": json.RawMessage(attrs)})
	default:
		f.t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		notFound()
	}
}

func (f *fakeConfigEditor) requestLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.requests...)
}

// resource returns a config-item resource of f's domain, talking to f.
func (f *fakeConfigEditor) resource(newItems func(*client.RESTClient) client.ConfigItems) *configItemResource {
	srv := httptest.NewServer(f)
	f.t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		f.t.Fatal(err)
	}
	rest := client.New(client.Config{URL: u, Token: "t", Timeout: 5 * time.Second}).REST
	return &configItemResource{spec: configItemSpec{domain: f.domain}, items: newItems(rest)}
}

// fakePrivate is private state.
type fakePrivate map[string][]byte

func (p fakePrivate) SetKey(_ context.Context, key string, value []byte) diag.Diagnostics {
	p[key] = value
	return nil
}

func (p fakePrivate) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return p[key], nil
}

func configItemPlan(t *testing.T, id, config string) configItemModel {
	t.Helper()
	return configItemModel{ID: types.StringValue(id), Config: mustDyn(t, config), EntityID: types.StringUnknown()}
}

func TestConfigItemCreateWaitsForEntity(t *testing.T) {
	fastPolls(t)
	f := newFakeConfigEditor(t, "automation")
	f.reloadAfter = 3
	f.pending["automation.leaving_home"] = `{"id":"away"}`
	f.states["script.away"] = `{"id":"away"}`
	r := f.resource(client.NewAutomations)

	plan := configItemPlan(t, "away", `{"alias":"Leaving home"}`)
	private := fakePrivate{}
	var diags diag.Diagnostics
	if !r.create(t.Context(), &plan, time.Minute, private, &diags) || diags.HasError() {
		t.Fatalf("create failed: %v", diags)
	}
	if plan.EntityID.ValueString() != "automation.leaving_home" {
		t.Errorf("entity_id = %s", plan.EntityID)
	}
	if private[baselineKey] == nil {
		t.Error("no baseline stored")
	}
}

// Saved but no entity: the item goes into state with a null entity_id and an error, so it is
// tainted, not orphaned.
func TestConfigItemCreateWithoutEntityKeepsState(t *testing.T) {
	fastPolls(t)
	f := newFakeConfigEditor(t, "automation")
	f.reloadAfter = -1
	r := f.resource(client.NewAutomations)

	plan := configItemPlan(t, "away", `{"alias":"Leaving home"}`)
	var diags diag.Diagnostics
	if !r.create(t.Context(), &plan, 20*time.Millisecond, fakePrivate{}, &diags) {
		t.Fatal("a saved item is not kept in state")
	}
	if !plan.EntityID.IsNull() {
		t.Errorf("entity_id = %s, want null", plan.EntityID)
	}
	if !diags.HasError() {
		t.Fatal("no error")
	}
	detail := diags.Errors()[0].Detail()
	for _, want := range []string{"attributes.id", "within 20ms", "timeouts.create"} {
		if !strings.Contains(detail, want) {
			t.Errorf("error %q does not mention %q", detail, want)
		}
	}
}

// An existing item with the ID is not overwritten (ADR-0024).
func TestConfigItemCreateFailsOnExistingID(t *testing.T) {
	f := newFakeConfigEditor(t, "automation")
	f.items["away"] = `{"id":"away","alias":"Made in the UI"}`
	r := f.resource(client.NewAutomations)

	plan := configItemPlan(t, "away", `{"alias":"Leaving home"}`)
	var diags diag.Diagnostics
	if r.create(t.Context(), &plan, time.Minute, fakePrivate{}, &diags) {
		t.Error("an item that was not saved is kept in state")
	}
	if len(diags.Errors()) != 1 {
		t.Fatalf("diags = %v", diags)
	}
	d := diags.Errors()[0]
	if p, ok := d.(diag.DiagnosticWithPath); !ok || !p.Path().Equal(path.Root("id")) {
		t.Errorf("error is not on `id`: %v", d)
	}
	if !strings.Contains(d.Detail(), "tofu import homeassistant_automation.<name> away") {
		t.Errorf("detail = %q", d.Detail())
	}
	if f.items["away"] != `{"id":"away","alias":"Made in the UI"}` {
		t.Errorf("existing item overwritten: %s", f.items["away"])
	}
}

// Home Assistant's validation error is attached to `config`.
func TestConfigItemRejectedConfigIsOnConfig(t *testing.T) {
	f := newFakeConfigEditor(t, "automation")
	f.saveStatus = http.StatusBadRequest
	r := f.resource(client.NewAutomations)

	plan := configItemPlan(t, "away", `{"alias":"Leaving home"}`)
	var diags diag.Diagnostics
	if r.create(t.Context(), &plan, time.Minute, fakePrivate{}, &diags) {
		t.Error("a rejected item is kept in state")
	}
	if len(diags.Errors()) != 1 {
		t.Fatalf("diags = %v", diags)
	}
	d := diags.Errors()[0]
	if p, ok := d.(diag.DiagnosticWithPath); !ok || !p.Path().Equal(path.Root("config")) {
		t.Errorf("error is not on `config`: %v", d)
	}
	if !strings.Contains(d.Detail(), "extra keys not allowed") {
		t.Errorf("detail = %q", d.Detail())
	}
}

func TestAddSaveErrorOnlyAttachesRejections(t *testing.T) {
	for name, tc := range map[string]struct {
		err      error
		onConfig bool
	}{
		"REST 400":    {&client.HTTPError{StatusCode: 400, Body: "malformed"}, true},
		"WS error":    {&client.WSError{Code: "invalid_format", Message: "bad"}, true},
		"REST 500":    {&client.HTTPError{StatusCode: 500, Body: "boom"}, false},
		"unreachable": {client.ErrUnreachable, false},
	} {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			addSaveError(&diags, "Saving", tc.err)
			_, onPath := diags.Errors()[0].(diag.DiagnosticWithPath)
			if onPath != tc.onConfig {
				t.Errorf("on config = %v, want %v", onPath, tc.onConfig)
			}
		})
	}
}

// A refresh checks the known entity only, and keeps it while a reload is still running.
func TestConfigItemReadEntity(t *testing.T) {
	for name, tc := range map[string]struct {
		states   map[string]string
		known    types.String
		want     string
		requests []string
	}{
		"known entity": {
			states: map[string]string{"automation.leaving_home": `{"id":"away"}`},
			known:  types.StringValue("automation.leaving_home"), want: "automation.leaving_home",
			requests: []string{"GET /api/config/automation/config/away", "GET /api/states/automation.leaving_home"},
		},
		"renamed entity": {
			states: map[string]string{"automation.renamed": `{"id":"away"}`},
			known:  types.StringValue("automation.leaving_home"), want: "automation.renamed",
			requests: []string{"GET /api/config/automation/config/away", "GET /api/states/automation.leaving_home", "GET /api/states"},
		},
		"not reloaded yet": {
			known: types.StringValue("automation.leaving_home"), want: "automation.leaving_home",
			requests: []string{"GET /api/config/automation/config/away", "GET /api/states/automation.leaving_home", "GET /api/states"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakeConfigEditor(t, "automation")
			f.items["away"] = `{"id":"away","alias":"Leaving home"}`
			f.states = tc.states
			r := f.resource(client.NewAutomations)

			state := configItemPlan(t, "away", `{"alias":"Leaving home"}`)
			state.EntityID = tc.known
			var diags diag.Diagnostics
			if !r.read(t.Context(), &state, fakePrivate{}, &diags) || diags.HasError() {
				t.Fatalf("read failed: %v", diags)
			}
			if state.EntityID.ValueString() != tc.want {
				t.Errorf("entity_id = %s, want %s", state.EntityID, tc.want)
			}
			if got := f.requestLog(); strings.Join(got, "\n") != strings.Join(tc.requests, "\n") {
				t.Errorf("requests = %q, want %q", got, tc.requests)
			}
		})
	}
}

// A script's entity ID is fixed, so a refresh doesn't look it up.
func TestScriptReadNeedsNoStates(t *testing.T) {
	f := newFakeConfigEditor(t, "script")
	f.items["goodnight"] = `{"alias":"Goodnight"}`
	r := f.resource(client.NewScripts)

	state := configItemPlan(t, "goodnight", `{"alias":"Goodnight"}`)
	state.EntityID = types.StringNull()
	var diags diag.Diagnostics
	if !r.read(t.Context(), &state, fakePrivate{}, &diags) || diags.HasError() {
		t.Fatalf("read failed: %v", diags)
	}
	if state.EntityID.ValueString() != "script.goodnight" {
		t.Errorf("entity_id = %s", state.EntityID)
	}
	if got := f.requestLog(); len(got) != 1 {
		t.Errorf("requests = %q, want only the config", got)
	}
}

func TestConfigItemReadGone(t *testing.T) {
	f := newFakeConfigEditor(t, "automation")
	r := f.resource(client.NewAutomations)
	state := configItemPlan(t, "away", `{"alias":"Leaving home"}`)
	var diags diag.Diagnostics
	if r.read(t.Context(), &state, fakePrivate{}, &diags) || diags.HasError() {
		t.Errorf("a deleted item is kept: %v", diags)
	}
}
