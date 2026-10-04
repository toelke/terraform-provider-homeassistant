package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// fakeFlow is a fake config-flow endpoint. init answers the POST that starts the flow; each later
// POST to the flow is recorded and answered by the next entry of replies.
type fakeFlow struct {
	t       *testing.T
	init    string
	replies []string

	mu        sync.Mutex
	initBody  map[string]any
	submitted []map[string]any
	deleted   []string
}

func (f *fakeFlow) client() ConfigFlows {
	return NewConfigFlows(serve(f.t, f.handle))
}

func (f *fakeFlow) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/config/config_entries/flow":
		f.initBody = decodeBody(f.t, r)
		_, _ = w.Write([]byte(f.init))
	case r.Method == http.MethodPost && r.URL.Path == "/api/config/config_entries/flow/f1":
		f.submitted = append(f.submitted, decodeBody(f.t, r))
		if len(f.replies) == 0 {
			f.t.Errorf("unexpected submit %v", f.submitted[len(f.submitted)-1])
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		reply := f.replies[0]
		f.replies = f.replies[1:]
		if code, body, ok := strings.Cut(reply, " "); ok && code == "400" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(reply))
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/config/config_entries/flow/"):
		f.deleted = append(f.deleted, strings.TrimPrefix(r.URL.Path, "/api/config/config_entries/flow/"))
		_, _ = w.Write([]byte(`{"message":"Flow aborted"}`))
	default:
		f.t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusTeapot)
	}
}

func decodeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		t.Errorf("decoding body: %v", err)
	}
	return m
}

const (
	formUser    = `{"type":"form","flow_id":"f1","handler":"shelly","step_id":"user","errors":{}}`
	formCreds   = `{"type":"form","flow_id":"f1","handler":"shelly","step_id":"credentials","errors":null}`
	createEntry = `{"type":"create_entry","flow_id":"f1","handler":"shelly","result":{"entry_id":"E1","domain":"shelly"}}`
)

func TestConfigFlowSingleStep(t *testing.T) {
	f := &fakeFlow{t: t, init: formUser, replies: []string{createEntry}}

	id, err := f.client().Run(t.Context(), "shelly", "", []FlowStep{
		{StepID: "user", Data: map[string]any{"host": "192.168.1.50", "port": 80}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "E1" {
		t.Errorf("entry_id = %q", id)
	}
	if want := map[string]any{"handler": "shelly", "show_advanced_options": true}; !reflect.DeepEqual(f.initBody, want) {
		t.Errorf("init body = %v", f.initBody)
	}
	if want := []map[string]any{{"host": "192.168.1.50", "port": float64(80)}}; !reflect.DeepEqual(f.submitted, want) {
		t.Errorf("submitted = %v", f.submitted)
	}
	if len(f.deleted) != 0 {
		t.Errorf("deleted = %v", f.deleted)
	}
}

func TestConfigFlowMultipleStepsMergesSensitiveData(t *testing.T) {
	f := &fakeFlow{t: t, init: formUser, replies: []string{formCreds, createEntry}}

	// Steps are matched by step_id, not by position.
	id, err := f.client().Run(t.Context(), "shelly", "", []FlowStep{
		{
			StepID:        "credentials",
			Data:          map[string]any{"username": "admin", "password": "overridden"},
			SensitiveData: map[string]any{"password": "hunter2"},
		},
		{StepID: "user", Data: map[string]any{"host": "h"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "E1" {
		t.Errorf("entry_id = %q", id)
	}
	want := []map[string]any{
		{"host": "h"},
		{"username": "admin", "password": "hunter2"},
	}
	if !reflect.DeepEqual(f.submitted, want) {
		t.Errorf("submitted = %v", f.submitted)
	}
}

func TestConfigFlowUnexpectedStepAborts(t *testing.T) {
	f := &fakeFlow{t: t, init: formUser, replies: []string{formCreds}}

	_, err := f.client().Run(t.Context(), "shelly", "", []FlowStep{{StepID: "user"}})

	var missing *FlowStepMissingError
	if !errors.As(err, &missing) || missing.StepID != "credentials" || missing.Repeated {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), `asked for step "credentials", but the config only has "user"`) {
		t.Errorf("message = %q", err)
	}
	if !reflect.DeepEqual(f.deleted, []string{"f1"}) {
		t.Errorf("deleted = %v", f.deleted)
	}
}

func TestConfigFlowRepeatedStepAborts(t *testing.T) {
	f := &fakeFlow{t: t, init: formUser, replies: []string{formUser}}

	_, err := f.client().Run(t.Context(), "shelly", "", []FlowStep{{StepID: "user"}})

	var missing *FlowStepMissingError
	if !errors.As(err, &missing) || !missing.Repeated {
		t.Fatalf("err = %v", err)
	}
	if len(f.submitted) != 1 || !reflect.DeepEqual(f.deleted, []string{"f1"}) {
		t.Errorf("submitted = %v, deleted = %v", f.submitted, f.deleted)
	}
}

func TestConfigFlowFormErrorsAbort(t *testing.T) {
	f := &fakeFlow{t: t, init: formUser, replies: []string{
		`{"type":"form","flow_id":"f1","step_id":"user","errors":{"base":"cannot_connect","host":"invalid_host"}}`,
	}}

	_, err := f.client().Run(t.Context(), "shelly", "", []FlowStep{{StepID: "user"}})

	var formErr *FlowFormError
	if !errors.As(err, &formErr) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), `rejected step "user": base: cannot_connect, host: invalid_host`) {
		t.Errorf("message = %q", err)
	}
	if !reflect.DeepEqual(f.deleted, []string{"f1"}) {
		t.Errorf("deleted = %v", f.deleted)
	}
}

func TestConfigFlowRejectedSubmitAborts(t *testing.T) {
	// HA answers 400 when the data does not match the step's schema, and leaves the flow open.
	f := &fakeFlow{t: t, init: formUser, replies: []string{`400 {"message":"User input malformed: required key not provided @ data['host']"}`}}

	_, err := f.client().Run(t.Context(), "shelly", "", []FlowStep{{StepID: "user"}})

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "required key not provided") {
		t.Errorf("message = %q", err)
	}
	if !reflect.DeepEqual(f.deleted, []string{"f1"}) {
		t.Errorf("deleted = %v", f.deleted)
	}
}

func TestConfigFlowAbortReason(t *testing.T) {
	for name, tc := range map[string]struct {
		init    string
		replies []string
	}{
		"at init":     {init: `{"type":"abort","flow_id":"f1","reason":"already_configured"}`},
		"after start": {init: formUser, replies: []string{`{"type":"abort","flow_id":"f1","reason":"already_configured"}`}},
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeFlow{t: t, init: tc.init, replies: tc.replies}

			_, err := f.client().Run(t.Context(), "shelly", "", []FlowStep{{StepID: "user"}})

			var aborted *FlowAbortedError
			if !errors.As(err, &aborted) || aborted.Reason != "already_configured" {
				t.Fatalf("err = %v", err)
			}
			// HA already ended the flow; there is nothing to delete.
			if len(f.deleted) != 0 {
				t.Errorf("deleted = %v", f.deleted)
			}
		})
	}
}

func TestConfigFlowReconfigure(t *testing.T) {
	f := &fakeFlow{t: t, init: `{"type":"form","flow_id":"f1","step_id":"reconfigure","errors":{}}`, replies: []string{
		`{"type":"abort","flow_id":"f1","reason":"reconfigure_successful"}`,
	}}

	id, err := f.client().Run(t.Context(), "esphome", "E7", []FlowStep{
		{StepID: "reconfigure", Data: map[string]any{"host": "new"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "E7" {
		t.Errorf("entry_id = %q", id)
	}
	if want := map[string]any{"handler": "esphome", "show_advanced_options": true, "entry_id": "E7"}; !reflect.DeepEqual(f.initBody, want) {
		t.Errorf("init body = %v", f.initBody)
	}
}

func TestConfigFlowReconfigureSuccessfulOnlyCountsForReconfigure(t *testing.T) {
	f := &fakeFlow{t: t, init: `{"type":"abort","flow_id":"f1","reason":"reconfigure_successful"}`}

	_, err := f.client().Run(t.Context(), "esphome", "", nil)

	var aborted *FlowAbortedError
	if !errors.As(err, &aborted) {
		t.Fatalf("err = %v", err)
	}
}

func TestConfigFlowUnsupportedResultAborts(t *testing.T) {
	for _, typ := range []string{"menu", "external", "show_progress", "progress_done"} {
		t.Run(typ, func(t *testing.T) {
			f := &fakeFlow{t: t, init: formUser, replies: []string{
				`{"type":"` + typ + `","flow_id":"f1","step_id":"pick"}`,
			}}

			_, err := f.client().Run(t.Context(), "shelly", "", []FlowStep{{StepID: "user"}})

			var unsupported *FlowUnsupportedError
			if !errors.As(err, &unsupported) || unsupported.Type != typ {
				t.Fatalf("err = %v", err)
			}
			if !reflect.DeepEqual(f.deleted, []string{"f1"}) {
				t.Errorf("deleted = %v", f.deleted)
			}
		})
	}
}

func TestConfigFlowUnknownHandler(t *testing.T) {
	c := NewConfigFlows(serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Invalid handler specified"}`))
	}))

	_, err := c.Run(t.Context(), "nope", "", nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestConfigFlowAbortFailureIsReported(t *testing.T) {
	c := NewConfigFlows(serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"type":"menu","flow_id":"f1","step_id":"pick"}`))
	}))

	_, err := c.Run(t.Context(), "shelly", "", nil)

	var unsupported *FlowUnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "aborting config flow f1") {
		t.Fatalf("err = %v", err)
	}
}
