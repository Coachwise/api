package utils

import (
	"bytes"
	"encoding/json"
	"coachwise/src/logger"
	"net/http"
)

// PostDiscordWebhook sends a plain message to a Discord webhook (fire-and-forget).
// When the URL is empty it logs the message instead, so flows stay testable in dev.
func PostDiscordWebhook(webhookURL, content string) {
	if webhookURL == "" {
		logger.Infof("[discord] webhook not configured; message:\n%s", content)
		return
	}
	go func() {
		body, _ := json.Marshal(map[string]string{"content": content})
		resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
		if err != nil {
			logger.Errorf("[discord] post failed: %v", err)
			return
		}
		resp.Body.Close()
	}()
}
