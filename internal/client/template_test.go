package client

import (
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestRenderTemplateReturnsText(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/template" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != `{"template":"{{ 1 + 1 }}"}` {
			t.Errorf("body = %s", b)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("2"))
	})
	got, err := c.RenderTemplate(t.Context(), "{{ 1 + 1 }}")
	if err != nil {
		t.Fatal(err)
	}
	if got != "2" {
		t.Errorf("result = %q", got)
	}
}

func TestRenderTemplateErrorWithStatus200(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("Error rendering template: UndefinedError: 'foo' is undefined"))
	})
	_, err := c.RenderTemplate(t.Context(), "{{ foo.bar }}")
	var tmplErr *TemplateError
	if !errors.As(err, &tmplErr) {
		t.Fatalf("err = %v, want *TemplateError", err)
	}
	if tmplErr.Message != "Error rendering template: UndefinedError: 'foo' is undefined" {
		t.Errorf("message = %q", tmplErr.Message)
	}
}

func TestRenderTemplateErrorStatus(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Error rendering template: TemplateSyntaxError"}`, http.StatusBadRequest)
	})
	_, err := c.RenderTemplate(t.Context(), "{{")
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %v, want HTTP 400", err)
	}
}
