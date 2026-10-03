package acctest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Credentials of the owner user that onboarding creates.
const (
	ownerName     = "Acceptance Test"
	ownerUsername = "acctest"
	ownerPassword = "acctest-password"
)

// onboard walks a fresh Home Assistant through onboarding without a browser and returns a
// long-lived access token of the owner user (ADR-0017). baseURL has no trailing slash.
func onboard(ctx context.Context, baseURL string) (string, error) {
	// HA's IndieAuth requires the client ID to be a URL; the redirect URI must share its host.
	clientID := baseURL + "/"
	o := onboarder{base: baseURL, http: &http.Client{}}

	var user struct {
		AuthCode string `json:"auth_code"`
	}
	if err := o.postJSON(ctx, "/api/onboarding/users", map[string]string{
		"client_id": clientID,
		"name":      ownerName,
		"username":  ownerUsername,
		"password":  ownerPassword,
		"language":  "en",
	}, &user); err != nil {
		return "", fmt.Errorf("create owner user: %w", err)
	}

	accessToken, err := o.exchangeAuthCode(ctx, clientID, user.AuthCode)
	if err != nil {
		return "", fmt.Errorf("exchange auth code: %w", err)
	}
	o.token = accessToken

	steps := []struct {
		path string
		body any
	}{
		{"/api/onboarding/core_config", map[string]any{}},
		{"/api/onboarding/analytics", map[string]any{}},
		{"/api/onboarding/integration", map[string]string{"client_id": clientID, "redirect_uri": clientID}},
	}
	for _, s := range steps {
		if err := o.postJSON(ctx, s.path, s.body, nil); err != nil {
			return "", fmt.Errorf("onboarding step %s: %w", s.path, err)
		}
	}

	token, err := mintLongLivedToken(ctx, baseURL, accessToken)
	if err != nil {
		return "", fmt.Errorf("mint long-lived token: %w", err)
	}
	return token, nil
}

type onboarder struct {
	base  string
	http  *http.Client
	token string
}

// postJSON posts body as JSON and decodes a 200 response into out, if out is not nil.
func (o onboarder) postJSON(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.token != "" {
		req.Header.Set("Authorization", "Bearer "+o.token)
	}
	return o.do(req, out)
}

func (o onboarder) exchangeAuthCode(ctx context.Context, clientID, code string) (string, error) {
	form := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
		"client_id":  {clientID},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base+"/auth/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := o.do(req, &tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("response has no access_token")
	}
	return tok.AccessToken, nil
}

func (o onboarder) do(req *http.Request, out any) error {
	resp, err := o.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s: HTTP %d: %s", req.Method, req.URL.Path, resp.StatusCode, body)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

// mintLongLivedToken authenticates on the WebSocket API with a short-lived access token and
// sends `auth/long_lived_access_token`.
func mintLongLivedToken(ctx context.Context, baseURL, accessToken string) (string, error) {
	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/api/websocket"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.CloseNow() }()

	var msg struct {
		ID      int             `json:"id"`
		Type    string          `json:"type"`
		Success bool            `json:"success"`
		Result  json.RawMessage `json:"result"`
		Error   *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := wsjson.Read(ctx, conn, &msg); err != nil || msg.Type != "auth_required" {
		return "", fmt.Errorf("expected auth_required, got %q: %v", msg.Type, err)
	}
	if err := wsjson.Write(ctx, conn, map[string]string{"type": "auth", "access_token": accessToken}); err != nil {
		return "", err
	}
	if err := wsjson.Read(ctx, conn, &msg); err != nil {
		return "", err
	}
	if msg.Type != "auth_ok" {
		return "", fmt.Errorf("authentication failed: %s %s", msg.Type, msg.Message)
	}

	if err := wsjson.Write(ctx, conn, map[string]any{
		"id":          1,
		"type":        "auth/long_lived_access_token",
		"client_name": "terraform-provider-homeassistant acceptance tests",
		"lifespan":    3650,
	}); err != nil {
		return "", err
	}
	if err := wsjson.Read(ctx, conn, &msg); err != nil {
		return "", err
	}
	if !msg.Success {
		if msg.Error != nil {
			return "", fmt.Errorf("%s: %s", msg.Error.Code, msg.Error.Message)
		}
		return "", fmt.Errorf("command failed")
	}
	var token string
	if err := json.Unmarshal(msg.Result, &token); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")
	return token, nil
}
