package client

import (
	"errors"
	"fmt"
)

var (
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
