package datasources

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/toelke/terraform-provider-homeassistant/internal/client"
)

func TestIsRenderError(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want bool
	}{
		"error text with 200": {fmt.Errorf("x: %w", &client.TemplateError{Message: "Error rendering template"}), true},
		"HTTP 400":            {fmt.Errorf("x: %w", &client.HTTPError{StatusCode: http.StatusBadRequest}), true},
		"HTTP 500":            {fmt.Errorf("x: %w", &client.HTTPError{StatusCode: http.StatusInternalServerError}), false},
		"unreachable":         {fmt.Errorf("x: %w", client.ErrUnreachable), false},
		"other":               {errors.New("boom"), false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := isRenderError(tc.err); got != tc.want {
				t.Errorf("isRenderError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
