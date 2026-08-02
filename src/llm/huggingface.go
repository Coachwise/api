package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"coachwise/src/config"
)

// huggingFace calls an OpenAI-compatible chat-completions endpoint (HF Inference
// / TGI router). Untested against a live model — wired for phase 5.
type huggingFace struct {
	baseURL string
	model   string
	apiKey  string
	client  *http.Client
}

func newHuggingFace() Provider {
	return huggingFace{
		baseURL: config.Config.LLM.BaseURL,
		model:   config.Config.LLM.Model,
		apiKey:  config.Config.LLM.APIKey,
		client:  &http.Client{},
	}
}

func (huggingFace) Name() string { return "huggingface" }

func (h huggingFace) Complete(ctx context.Context, req Request) (Response, error) {
	msgs := []map[string]string{}
	if req.System != "" {
		msgs = append(msgs, map[string]string{"role": "system", "content": req.System})
	}
	for _, m := range req.Messages {
		role := string(m.Role)
		if m.Role == RoleTool {
			role = "user" // fold tool results into the conversation
		}
		msgs = append(msgs, map[string]string{"role": role, "content": m.Content})
	}

	body, _ := json.Marshal(map[string]any{
		"model":       h.model,
		"messages":    msgs,
		"max_tokens":  firstPositive(config.Config.LLM.MaxTokens, 1024),
		"temperature": config.Config.LLM.Temperature,
	})

	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	if h.apiKey != "" {
		hreq.Header.Set("Authorization", "Bearer "+h.apiKey)
	}

	res, err := h.client.Do(hreq)
	if err != nil {
		return Response{}, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return Response{}, fmt.Errorf("llm: huggingface %d: %s", res.StatusCode, string(raw))
	}

	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return Response{}, err
	}
	if len(out.Choices) == 0 {
		return Response{}, errors.New("llm: huggingface returned no choices")
	}
	model := out.Model
	if model == "" {
		model = h.model
	}
	return Response{
		Text:  out.Choices[0].Message.Content,
		Model: model,
		Usage: Usage{out.Usage.PromptTokens, out.Usage.CompletionTokens, out.Usage.TotalTokens},
	}, nil
}

func firstPositive(a, b int) int {
	if a > 0 {
		return a
	}
	return b
}
