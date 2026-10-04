package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// templateErrorPrefix starts the body of a template that HA failed to render. Some HA versions
// send it with HTTP 200 instead of an error status.
const templateErrorPrefix = "Error rendering template"

// TemplateError is a template that Home Assistant failed to render.
type TemplateError struct {
	Message string
}

func (e *TemplateError) Error() string {
	return e.Message
}

// RenderTemplate renders a Jinja template with `POST /api/template` and returns the result text.
// A render error that HA reports with HTTP 200 becomes a *TemplateError; one reported with an
// error status stays an *HTTPError.
func (c *RESTClient) RenderTemplate(ctx context.Context, template string) (string, error) {
	result, err := c.DoText(ctx, http.MethodPost, "template", map[string]string{"template": template})
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(result, templateErrorPrefix) {
		return "", fmt.Errorf("POST template: %w", &TemplateError{Message: result})
	}
	return result, nil
}
