package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"coachwise/src/logger"
)

// discordClient carries the proxy config; built once by InitDiscord.
var discordClient = &http.Client{Timeout: 10 * time.Second}

// InitDiscord builds the HTTP client used for every Discord webhook. A proxy is
// needed to reach Discord from Iran; empty falls back to the *_PROXY env vars.
// Call once at startup (cmd/app, cmd/worker).
func InitDiscord(proxy string) {
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if proxy != "" {
		if u, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(u)
		} else {
			logger.Errorf("[discord] bad proxy %q: %v", proxy, err)
		}
	}
	discordClient = &http.Client{Timeout: 10 * time.Second, Transport: transport}
}

// DiscordClient is the shared, proxy-aware client (also used by the alert sink).
func DiscordClient() *http.Client { return discordClient }

// PostDiscordWebhook sends a plain message to a Discord webhook (fire-and-forget).
// When the URL is empty it logs the message instead, so flows stay testable in dev.
func PostDiscordWebhook(webhookURL, content string) {
	if webhookURL == "" {
		logger.Infof("[discord] webhook not configured; message:\n%s", content)
		return
	}
	go func() {
		body, _ := json.Marshal(map[string]string{"content": content})
		resp, err := discordClient.Post(webhookURL, "application/json", bytes.NewReader(body))
		if err != nil {
			logger.Errorf("[discord] post failed: %v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
			logger.Errorf("[discord] webhook rejected: %d %s", resp.StatusCode, b)
		}
	}()
}
