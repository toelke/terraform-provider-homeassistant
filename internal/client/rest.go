package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// maxErrorBody bounds how much of an error response ends up in a diagnostic.
const maxErrorBody = 512

// Get sends `GET /api/<path>` and decodes the JSON response into out.
func (c *RESTClient) Get(ctx context.Context, path string, out any) error {
	return c.Do(ctx, http.MethodGet, path, nil, out)
}

// Do sends a request to `/api/<path>`. A non-nil body is sent as JSON, and a non-nil out receives
// the decoded JSON response. Failures map to ErrUnreachable, ErrUnauthorized, ErrNotFound, or an
// *HTTPError.
func (c *RESTClient) Do(ctx context.Context, method, path string, body, out any) error {
	return c.send(ctx, method, path, body, func(endpoint string, r io.Reader) error {
		if out == nil {
			return nil
		}
		if err := json.NewDecoder(r).Decode(out); err != nil {
			return fmt.Errorf("decoding response of %s %s: %w", method, endpoint, err)
		}
		return nil
	})
}

// DoText is Do for endpoints that answer with plain text instead of JSON, e.g. `/api/template`.
// It returns the response body.
func (c *RESTClient) DoText(ctx context.Context, method, path string, body any) (string, error) {
	var text string
	err := c.send(ctx, method, path, body, func(endpoint string, r io.Reader) error {
		b, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("reading response of %s %s: %w", method, endpoint, err)
		}
		text = string(b)
		return nil
	})
	return text, err
}

// Upload sends content as a file to `POST /api/file_upload` and returns its file_id, which a
// flow's file field takes. HA deletes the file once the flow has read it.
func (c *RESTClient) Upload(ctx context.Context, filename string, content []byte) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filename)
	if err == nil {
		_, err = part.Write(content)
	}
	if err == nil {
		err = w.Close()
	}
	if err != nil {
		return "", fmt.Errorf("encoding upload of %s: %w", filename, err)
	}
	var out struct {
		FileID string `json:"file_id"`
	}
	body := rawBody{contentType: w.FormDataContentType(), data: buf.Bytes()}
	err = c.Do(ctx, http.MethodPost, "file_upload", body, &out)
	if err == nil && out.FileID == "" {
		err = fmt.Errorf("uploading %s: no file_id in the response", filename)
	}
	return out.FileID, err
}

// rawBody is a request body that is sent as is instead of as JSON.
type rawBody struct {
	contentType string
	data        []byte
}

// send sends the request and passes a successful response body to read.
func (c *RESTClient) send(ctx context.Context, method, path string, body any,
	read func(endpoint string, r io.Reader) error,
) error {
	endpoint := c.cfg.URL.JoinPath("api", path).String()

	var reqBody io.Reader
	contentType := ""
	switch b := body.(type) {
	case nil:
	case rawBody:
		reqBody, contentType = bytes.NewReader(b.data), b.contentType
	default:
		js, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request to %s: %w", endpoint, err)
		}
		reqBody, contentType = bytes.NewReader(js), "application/json"
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%s %s: %w", method, endpoint, ctxErr)
		}
		return fmt.Errorf("%w at %s: %w", ErrUnreachable, endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := statusError(resp); err != nil {
		return fmt.Errorf("%s %s: %w", method, endpoint, err)
	}
	return read(endpoint, resp.Body)
}

func statusError(resp *http.Response) error {
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	// HA's views answer errors with `{"message": "..."}`; the message alone reads better.
	var msg struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(b, &msg) == nil && msg.Message != "" {
		return &HTTPError{StatusCode: resp.StatusCode, Body: msg.Message}
	}
	return &HTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(b))}
}
