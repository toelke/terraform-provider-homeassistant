package provider

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

const (
	envURL      = "HOMEASSISTANT_URL"
	envToken    = "HOMEASSISTANT_TOKEN"
	envInsecure = "HOMEASSISTANT_INSECURE"
	envTimeout  = "HOMEASSISTANT_TIMEOUT"

	defaultTimeout = 30 * time.Second
)

// lookupFunc reads an environment variable; it has the signature of os.LookupEnv.
type lookupFunc func(key string) (string, bool)

// resolveConfig merges the provider block with the environment: an argument set in config wins
// over its environment variable, which wins over the default.
func resolveConfig(model providerModel, lookup lookupFunc) (client.Config, diag.Diagnostics) {
	var diags diag.Diagnostics
	var cfg client.Config

	for _, a := range []struct {
		name    string
		unknown bool
	}{
		{"url", model.URL.IsUnknown()},
		{"token", model.Token.IsUnknown()},
		{"insecure", model.Insecure.IsUnknown()},
		{"timeout", model.Timeout.IsUnknown()},
	} {
		if a.unknown {
			diags.AddAttributeError(path.Root(a.name), "Unknown provider argument",
				fmt.Sprintf("The provider cannot be configured because %q is not known until apply. "+
					"Set it to a value known at plan time, or use its environment variable.", a.name))
		}
	}
	if diags.HasError() {
		return cfg, diags
	}

	rawURL := stringSetting(model.URL.ValueString(), envURL, lookup)
	if rawURL == "" {
		diags.AddAttributeError(path.Root("url"), "Missing Home Assistant URL",
			fmt.Sprintf("Set the `url` argument in the provider block or the %s environment variable "+
				"to the base URL of Home Assistant, e.g. http://homeassistant.local:8123.", envURL))
	} else {
		u, err := parseBaseURL(rawURL)
		if err != nil {
			diags.AddAttributeError(path.Root("url"), "Invalid Home Assistant URL",
				fmt.Sprintf("%q (from `url` or %s): %s.", rawURL, envURL, err))
		}
		cfg.URL = u
	}

	cfg.Token = stringSetting(model.Token.ValueString(), envToken, lookup)
	if cfg.Token == "" {
		diags.AddAttributeError(path.Root("token"), "Missing Home Assistant token",
			fmt.Sprintf("Set the `token` argument in the provider block or the %s environment variable "+
				"to a long-lived access token of an admin user.", envToken))
	}

	if !model.Insecure.IsNull() {
		cfg.Insecure = model.Insecure.ValueBool()
	} else if v, _ := lookup(envInsecure); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			diags.AddAttributeError(path.Root("insecure"), "Invalid insecure setting",
				fmt.Sprintf("%s=%q is not a boolean (use true or false).", envInsecure, v))
		}
		cfg.Insecure = b
	}

	cfg.Timeout = defaultTimeout
	if v := stringSetting(model.Timeout.ValueString(), envTimeout, lookup); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			diags.AddAttributeError(path.Root("timeout"), "Invalid timeout",
				fmt.Sprintf("%q (from `timeout` or %s) is not a positive duration such as \"30s\" or \"2m\".",
					v, envTimeout))
		}
		cfg.Timeout = d
	}

	return cfg, diags
}

// stringSetting returns the config value if set, otherwise the environment variable.
func stringSetting(configured, env string, lookup lookupFunc) string {
	if configured != "" {
		return configured
	}
	v, _ := lookup(env)
	return v
}

// parseBaseURL accepts an http(s) URL without query or fragment and strips a trailing slash.
func parseBaseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("scheme must be http or https")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("host is missing")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("must not contain a query or fragment")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	if strings.HasSuffix(u.Path, "/api") {
		return nil, fmt.Errorf("must be the base URL without /api")
	}
	return u, nil
}
