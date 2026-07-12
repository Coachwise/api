package sms

// Kavenegar SMS gateway — Iran (+98). OTP uses the verify/lookup endpoint (a
// pre-approved template, faster + higher delivery than free-text); generic Send
// uses sms/send.json. Docs: https://kavenegar.com/rest.html
//   otp:  GET /v1/{API-KEY}/verify/lookup.json?receptor=&token=&template=
//   send: GET /v1/{API-KEY}/sms/send.json?receptor=&sender=&message=
// Both reply { "return": {"status":200,"message":...}, "entries":[...] }.

import (
	"coachwise/src/logger"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var kavenegarHTTPClient = &http.Client{Timeout: 15 * time.Second}

type kavenegarProvider struct {
	apiKey    string
	sender    string
	template  string
	baseURL   string
	countries []string
}

func newKavenegar(apiKey, sender, template, baseURL string, countries []string) kavenegarProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.kavenegar.com"
	}
	if len(countries) == 0 {
		countries = []string{"98"}
	}
	return kavenegarProvider{
		apiKey:    apiKey,
		sender:    sender,
		template:  template,
		baseURL:   strings.TrimRight(baseURL, "/"),
		countries: countries,
	}
}

func (k kavenegarProvider) Name() string { return "kavenegar" }

func (k kavenegarProvider) Supports(phone string) bool {
	return hasDialCode(phone, k.countries)
}

func (k kavenegarProvider) SendOTP(phone, code string) error {
	if k.apiKey == "" || k.template == "" {
		return fmt.Errorf("%w: kavenegar api_key/template not configured", ErrSMS)
	}
	q := url.Values{}
	q.Set("receptor", toLocalIran(phone))
	q.Set("token", code)
	q.Set("template", k.template)
	logger.Infof("SMS(otp) to %s: code=%s template=%s (Kavehnegar)", phone, code, k.template)
	return k.call("verify/lookup.json", q)
}

func (k kavenegarProvider) Send(phone, text string) error {
	if k.apiKey == "" {
		return fmt.Errorf("%w: kavenegar api_key not configured", ErrSMS)
	}
	q := url.Values{}
	q.Set("receptor", toLocalIran(phone))
	q.Set("message", text)
	if k.sender != "" {
		q.Set("sender", k.sender)
	}
	logger.Infof("SMS(otp) to %s: text=%s (Kavehnegar)", phone, text)
	return k.call("sms/send.json", q)
}

func (k kavenegarProvider) call(endpoint string, q url.Values) error {
	u := fmt.Sprintf("%s/v1/%s/%s?%s", k.baseURL, k.apiKey, endpoint, q.Encode())
	resp, err := kavenegarHTTPClient.Get(u)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSMS, err)
	}
	defer resp.Body.Close()
	var out struct {
		Return struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"return"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("%w: bad response: %v", ErrSMS, err)
	}
	// Kavenegar returns 200 in the `return` envelope on success (per-message
	// delivery status is in entries[], which we don't block on).
	if out.Return.Status != 200 {
		return fmt.Errorf("%w: kavenegar status %d: %s", ErrSMS, out.Return.Status, out.Return.Message)
	}
	return nil
}
