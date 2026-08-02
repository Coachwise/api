package push

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"coachwise/src/config"
	"coachwise/src/logger"

	"github.com/golang-jwt/jwt/v5"
)

// FCM HTTP v1: the legacy server key is gone, so auth is a service-account JWT
// traded for a short-lived OAuth2 token.
const (
	fcmScope    = "https://www.googleapis.com/auth/firebase.messaging"
	fcmSendURL  = "https://fcm.googleapis.com/v1/projects/%s/messages:send"
	jwtBearer   = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	tokenSkew   = time.Minute
	httpTimeout = 15 * time.Second
)

// Android presentation — app resource ids and the brand accent, not copy.
const (
	androidChannel = "coachwise_default"
	androidIcon    = "ic_stat_notify"
	androidColor   = "#0097E6"
)

type serviceAccount struct {
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	TokenURI     string `json:"token_uri"`
}

type fcmSender struct {
	projectID string
	email     string
	keyID     string
	key       *rsa.PrivateKey
	tokenURI  string
	client    *http.Client

	mu      sync.Mutex
	token   string
	expires time.Time
}

func newFCM() (*fcmSender, error) {
	cfg := config.Config.Push
	if cfg.CredentialsFile == "" {
		return nil, fmt.Errorf("credentials_file is empty")
	}
	raw, err := os.ReadFile(cfg.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}
	var sa serviceAccount
	if err := json.Unmarshal(raw, &sa); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(sa.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	projectID := cfg.ProjectID
	if projectID == "" {
		projectID = sa.ProjectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("project_id missing from config and credentials")
	}
	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if cfg.Proxy != "" {
		u, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, fmt.Errorf("bad proxy %q: %w", cfg.Proxy, err)
		}
		transport.Proxy = http.ProxyURL(u)
	}

	return &fcmSender{
		projectID: projectID,
		email:     sa.ClientEmail,
		keyID:     sa.PrivateKeyID,
		key:       key,
		tokenURI:  tokenURI,
		client:    &http.Client{Timeout: httpTimeout, Transport: transport},
	}, nil
}

// accessToken returns the cached token, re-minting near expiry. Serialized so a
// burst of notifications mints once.
func (f *fcmSender) accessToken(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.token != "" && time.Now().Before(f.expires.Add(-tokenSkew)) {
		return f.token, nil
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   f.email,
		"scope": fcmScope,
		"aud":   f.tokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if f.keyID != "" {
		tok.Header["kid"] = f.keyID
	}
	assertion, err := tok.SignedString(f.key)
	if err != nil {
		return "", fmt.Errorf("sign assertion: %w", err)
	}

	form := url.Values{"grant_type": {jwtBearer}, "assertion": {assertion}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("token endpoint %d: %s", resp.StatusCode, body)
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("token endpoint returned no access_token")
	}
	f.token = out.AccessToken
	f.expires = now.Add(time.Duration(out.ExpiresIn) * time.Second)
	return f.token, nil
}

// send posts one message; v1 has no multicast, so it's one call per device.
func (f *fcmSender) send(ctx context.Context, token string, m Message) error {
	access, err := f.accessToken(ctx)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(map[string]any{"message": f.buildMessage(token, m)})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf(fcmSendURL, f.projectID), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 300 {
		return nil
	}
	if isDeadToken(resp.StatusCode, body) {
		return ErrDeadToken
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		f.mu.Lock()
		f.token = "" // credentials problem: re-mint next time
		f.mu.Unlock()
	}
	return fmt.Errorf("fcm %d: %s", resp.StatusCode, body)
}

// buildMessage fills the platform blocks. No top-level `notification` — that
// field only takes literal text; loc keys exist only in the android/apns blocks.
func (f *fcmSender) buildMessage(token string, m Message) map[string]any {
	androidNotif := map[string]any{
		"channel_id": androidChannel,
		"icon":       androidIcon,
		"color":      androidColor,
	}
	if m.TitleLocKey != "" {
		androidNotif["title_loc_key"] = m.TitleLocKey
		if len(m.TitleLocArgs) > 0 {
			androidNotif["title_loc_args"] = m.TitleLocArgs
		}
	}
	if m.BodyLocKey != "" {
		androidNotif["body_loc_key"] = m.BodyLocKey
		if len(m.BodyLocArgs) > 0 {
			androidNotif["body_loc_args"] = m.BodyLocArgs
		}
	}

	alert := map[string]any{}
	if m.TitleLocKey != "" {
		alert["title-loc-key"] = m.TitleLocKey
		if len(m.TitleLocArgs) > 0 {
			alert["title-loc-args"] = m.TitleLocArgs
		}
	}
	if m.BodyLocKey != "" {
		alert["loc-key"] = m.BodyLocKey
		if len(m.BodyLocArgs) > 0 {
			alert["loc-args"] = m.BodyLocArgs
		}
	}

	msg := map[string]any{
		"token": token,
		"android": map[string]any{
			"priority":     "high",
			"notification": androidNotif,
		},
		"apns": map[string]any{
			"headers": map[string]any{"apns-priority": "10"},
			"payload": map[string]any{
				"aps": map[string]any{"alert": alert, "sound": "default"},
			},
		},
	}
	if len(m.Data) > 0 {
		msg["data"] = m.Data
	}
	return msg
}

// isDeadToken: only UNREGISTERED / 404. INVALID_ARGUMENT usually means a bad
// payload, and treating it as dead would wipe the table on our own bug.
func isDeadToken(status int, body []byte) bool {
	if status == http.StatusNotFound {
		return true
	}
	var out struct {
		Error struct {
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return false
	}
	for _, d := range out.Error.Details {
		if d.ErrorCode == "UNREGISTERED" {
			return true
		}
		if d.ErrorCode == "INVALID_ARGUMENT" {
			logger.Errorf("push: FCM rejected message as invalid (token kept): %s", body)
		}
	}
	return false
}
