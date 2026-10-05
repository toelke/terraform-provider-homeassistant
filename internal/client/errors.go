package client

import (
	"errors"
	"fmt"
	"slices"
)

var (
	// ErrNotFound means the requested object does not exist (HTTP 404).
	ErrNotFound = errors.New("not found")

	// ErrUnauthorized means Home Assistant rejected the token (HTTP 401, or `auth_invalid` on the
	// WebSocket).
	ErrUnauthorized = errors.New("token invalid or expired")

	// ErrUnreachable means Home Assistant could not be reached at all (ADR-0011).
	ErrUnreachable = errors.New("home assistant unreachable")

	// ErrConnectionLost means the WebSocket dropped while a command was in flight. The command may
	// or may not have been applied. It is retryable: the next command dials again.
	ErrConnectionLost = errors.New("websocket connection lost")
)

// WSError is a WebSocket result with `success: false`. Code is HA's machine-readable error code,
// e.g. `not_found` or `invalid_format`.
type WSError struct {
	Code    string
	Message string
}

func (e *WSError) Error() string {
	return fmt.Sprintf("home assistant error %s: %s", e.Code, e.Message)
}

// HTTPError is a REST response with an unexpected status that has no sentinel error of its own.
type HTTPError struct {
	StatusCode int
	// Body is the start of the response body, for the diagnostic.
	Body string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("home assistant returned HTTP %d: %s", e.StatusCode, e.Body)
}

// ErrorDetail renders err as the detail of an error diagnostic, adding a hint on what to do about
// it (ADR-0011).
func ErrorDetail(err error) string {
	switch {
	case errors.Is(err, ErrUnreachable):
		return err.Error() + "\n\nCheck that Home Assistant is running and that the provider `url` " +
			"is correct. To plan without contacting Home Assistant, run with -refresh=false."
	case errors.Is(err, ErrUnauthorized):
		return err.Error() + "\n\nCreate a new long-lived access token of an admin user in your " +
			"Home Assistant profile and set it as the provider `token`."
	}
	if hint := flowHint(err); hint != "" {
		return err.Error() + "\n\n" + hint
	}
	return err.Error()
}

// flowHint returns a hint for the config-flow aborts and form errors a user can act on, or "".
func flowHint(err error) string {
	var reasons []string
	var aborted *FlowAbortedError
	if errors.As(err, &aborted) {
		reasons = append(reasons, aborted.Reason)
	}
	var form *FlowFormError
	if errors.As(err, &form) {
		for _, r := range form.Errors {
			reasons = append(reasons, r)
		}
	}
	switch {
	case slices.Contains(reasons, "already_configured"), slices.Contains(reasons, "single_instance_allowed"):
		return "Home Assistant already has a config entry for this. Import it into this resource " +
			"with `tofu import <resource address> <entry_id>`, or delete it in Home Assistant. The " +
			"data source `homeassistant_integration` reads its entry_id."
	case slices.Contains(reasons, "cannot_connect"):
		return "Home Assistant could not connect to the device or service. Check the host and port, " +
			"and that Home Assistant can reach it over the network."
	case slices.Contains(reasons, "invalid_auth"):
		return "The device or service rejected the credentials. Check the username, password, or " +
			"key in the configuration."
	}
	return ""
}
