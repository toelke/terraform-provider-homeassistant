package provider

import (
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func env(vars map[string]string) lookupFunc {
	return func(key string) (string, bool) {
		v, ok := vars[key]
		return v, ok
	}
}

func nullModel() providerModel {
	return providerModel{
		URL:      types.StringNull(),
		Token:    types.StringNull(),
		Insecure: types.BoolNull(),
		Timeout:  types.StringNull(),
	}
}

func TestResolveConfig_ArgumentOverEnvironment(t *testing.T) {
	model := providerModel{
		URL:      types.StringValue("https://arg.example:8123/"),
		Token:    types.StringValue("arg-token"),
		Insecure: types.BoolValue(false),
		Timeout:  types.StringValue("5s"),
	}
	lookup := env(map[string]string{
		envURL:      "http://env.example:8123",
		envToken:    "env-token",
		envInsecure: "true",
		envTimeout:  "1m",
	})

	cfg, diags := resolveConfig(model, lookup)
	requireNoErrors(t, diags)

	if got := cfg.URL.String(); got != "https://arg.example:8123" {
		t.Errorf("URL = %q, want argument value without trailing slash", got)
	}
	if cfg.Token != "arg-token" {
		t.Errorf("Token = %q, want argument value", cfg.Token)
	}
	if cfg.Insecure {
		t.Errorf("Insecure = true, want argument value false")
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %s, want argument value 5s", cfg.Timeout)
	}
}

func TestResolveConfig_EnvironmentFallback(t *testing.T) {
	lookup := env(map[string]string{
		envURL:      "http://env.example:8123",
		envToken:    "env-token",
		envInsecure: "true",
		envTimeout:  "1m",
	})

	cfg, diags := resolveConfig(nullModel(), lookup)
	requireNoErrors(t, diags)

	if got := cfg.URL.String(); got != "http://env.example:8123" {
		t.Errorf("URL = %q, want environment value", got)
	}
	if cfg.Token != "env-token" {
		t.Errorf("Token = %q, want environment value", cfg.Token)
	}
	if !cfg.Insecure {
		t.Errorf("Insecure = false, want environment value true")
	}
	if cfg.Timeout != time.Minute {
		t.Errorf("Timeout = %s, want environment value 1m", cfg.Timeout)
	}
}

func TestResolveConfig_Defaults(t *testing.T) {
	model := nullModel()
	model.URL = types.StringValue("http://ha.local:8123")
	model.Token = types.StringValue("t")

	cfg, diags := resolveConfig(model, env(nil))
	requireNoErrors(t, diags)

	if cfg.Insecure {
		t.Errorf("Insecure = true, want default false")
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %s, want default 30s", cfg.Timeout)
	}
}

func TestResolveConfig_MissingURLAndToken(t *testing.T) {
	_, diags := resolveConfig(nullModel(), env(nil))

	if got := diags.ErrorsCount(); got != 2 {
		t.Fatalf("got %d errors, want 2: %v", got, diags)
	}
	requireErrorMentioning(t, diags, "`url`", envURL)
	requireErrorMentioning(t, diags, "`token`", envToken)
}

func TestResolveConfig_InvalidValues(t *testing.T) {
	tests := map[string]struct {
		model   func(*providerModel)
		env     map[string]string
		mention string
	}{
		"url without scheme": {
			model:   func(m *providerModel) { m.URL = types.StringValue("homeassistant.local:8123") },
			mention: envURL,
		},
		"url with /api": {
			model:   func(m *providerModel) { m.URL = types.StringValue("http://ha.local:8123/api") },
			mention: "without /api",
		},
		"insecure env not a bool": {
			env:     map[string]string{envInsecure: "yes please"},
			mention: envInsecure,
		},
		"timeout not a duration": {
			model:   func(m *providerModel) { m.Timeout = types.StringValue("30") },
			mention: envTimeout,
		},
		"timeout negative": {
			env:     map[string]string{envTimeout: "-5s"},
			mention: envTimeout,
		},
		"unknown url": {
			model:   func(m *providerModel) { m.URL = types.StringUnknown() },
			mention: "not known until apply",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			model := nullModel()
			model.URL = types.StringValue("http://ha.local:8123")
			model.Token = types.StringValue("t")
			if tc.model != nil {
				tc.model(&model)
			}

			_, diags := resolveConfig(model, env(tc.env))

			if got := diags.ErrorsCount(); got != 1 {
				t.Fatalf("got %d errors, want 1: %v", got, diags)
			}
			requireErrorMentioning(t, diags, tc.mention)
		})
	}
}

func requireNoErrors(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
}

// requireErrorMentioning asserts that one error's summary or detail contains all of substrs.
func requireErrorMentioning(t *testing.T, diags diag.Diagnostics, substrs ...string) {
	t.Helper()
	for _, d := range diags.Errors() {
		text := d.Summary() + " " + d.Detail()
		found := true
		for _, s := range substrs {
			if !strings.Contains(text, s) {
				found = false
				break
			}
		}
		if found {
			return
		}
	}
	t.Errorf("no error mentions all of %q: %v", substrs, diags)
}
