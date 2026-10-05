package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newTestREST(t *testing.T, rawURL string) *RESTClient {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return New(Config{URL: u, Token: "secret", Timeout: 5 * time.Second}).REST
}

func serve(t *testing.T, h http.HandlerFunc) *RESTClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return newTestREST(t, srv.URL)
}

func TestRESTSendsBearerTokenAndDecodes(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q", got)
		}
		if r.URL.Path != "/api/config" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"version":"2026.9.4","unit_system":{"temperature":"°C"},"components":["mqtt"]}`))
	})

	cfg, err := c.Config(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != "2026.9.4" || cfg.UnitSystem.Temperature != "°C" || len(cfg.Components) != 1 {
		t.Errorf("decoded %+v", cfg)
	}
}

func TestRESTSendsJSONBody(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != `{"template":"x"}` {
			t.Errorf("body = %q", b)
		}
	})
	if err := c.Do(t.Context(), http.MethodPost, "template", map[string]string{"template": "x"}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRESTErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusNotFound, ErrNotFound},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			})
			err := c.Get(t.Context(), "config", nil)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestRESTOtherStatusIsHTTPError(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	err := c.Get(t.Context(), "config", nil)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("err = %v, want *HTTPError", err)
	}
	if httpErr.StatusCode != http.StatusInternalServerError || httpErr.Body != "boom" {
		t.Errorf("got %+v", httpErr)
	}
	if errors.Is(err, ErrUnreachable) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrUnauthorized) {
		t.Errorf("err %v matches a sentinel", err)
	}
}

func TestRESTHTTPErrorUsesHAMessage(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message": "Message malformed: required key not provided"}`))
	})
	err := c.Get(t.Context(), "config", nil)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("err = %v, want *HTTPError", err)
	}
	if httpErr.StatusCode != http.StatusBadRequest || httpErr.Body != "Message malformed: required key not provided" {
		t.Errorf("got %+v", httpErr)
	}
}

func TestRESTConnectionFailureIsUnreachable(t *testing.T) {
	// Grab a free port, then close it, so nothing listens there.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	c := newTestREST(t, "http://"+addr)
	err = c.Get(t.Context(), "config", nil)
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if !strings.Contains(err.Error(), addr) {
		t.Errorf("err %q does not name the URL", err)
	}
}

func TestRESTCancelledContextIsNotUnreachable(t *testing.T) {
	c := serve(t, func(http.ResponseWriter, *http.Request) {})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := c.Get(ctx, "config", nil)
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v, want context.Canceled only", err)
	}
}

func TestRESTUndecodableBody(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})
	var out struct{}
	if err := c.Get(t.Context(), "config", &out); err == nil {
		t.Fatal("want decode error")
	}
}

func TestErrorDetailHints(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"unreachable", wrap(ErrUnreachable), "-refresh=false"},
		{"unauthorized", wrap(ErrUnauthorized), "long-lived access token"},
		{"other", errors.New("plain"), "plain"},
		{"already configured", &FlowAbortedError{Flow: "config flow for mqtt", Reason: "already_configured"}, "tofu import"},
		{"single instance", &FlowAbortedError{Flow: "config flow for met", Reason: "single_instance_allowed"}, "tofu import"},
		{"cannot connect abort", &FlowAbortedError{Flow: "config flow for shelly", Reason: "cannot_connect"}, "host and port"},
		{"cannot connect form", fmt.Errorf("creating: %w", &FlowFormError{
			Flow: "config flow for shelly", StepID: "user", Errors: map[string]string{"base": "cannot_connect"},
		}), "host and port"},
		{"invalid auth form", &FlowFormError{
			Flow: "config flow for shelly", StepID: "credentials", Errors: map[string]string{"password": "invalid_auth"},
		}, "credentials"},
		{"invalid auth abort", &FlowAbortedError{Flow: "config flow for shelly", Reason: "invalid_auth"}, "credentials"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ErrorDetail(tc.err)
			if !strings.Contains(got, tc.want) || !strings.HasPrefix(got, tc.err.Error()) {
				t.Errorf("ErrorDetail = %q, want it to start with the error and contain %q", got, tc.want)
			}
		})
	}
}

func TestErrorDetailNoHintForOtherFlowErrors(t *testing.T) {
	for _, err := range []error{
		&FlowAbortedError{Flow: "config flow for shelly", Reason: "no_devices_found"},
		&FlowFormError{Flow: "config flow for shelly", StepID: "user", Errors: map[string]string{"host": "invalid_host"}},
	} {
		if got := ErrorDetail(err); got != err.Error() {
			t.Errorf("ErrorDetail = %q, want just %q", got, err.Error())
		}
	}
}

func wrap(err error) error {
	return errors.Join(errors.New("GET http://ha:8123/api/config"), err)
}
