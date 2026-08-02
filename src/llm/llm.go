// Package llm is the text-generation backend for the AI assistant, behind a
// provider registry (like payments/sms/storage). The model only ever returns
// text; it never touches the DB. Real providers: huggingface. Unconfigured =
// AI disabled (endpoints degrade). Tests inject a double via SetProvider.
package llm

import (
	"context"
	"errors"
	"strings"

	"coachwise/src/config"
	"coachwise/src/logger"
)

var ErrNoProvider = errors.New("llm: no provider configured")

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	// RoleTool feeds executed-action results back into the prompt. It's a
	// prompt-only role — never persisted (DB rows are only user | assistant).
	RoleTool Role = "tool"
)

type Message struct {
	Role    Role
	Content string
}

type Request struct {
	System   string
	Messages []Message
}

// Usage is token accounting for one completion (0 when the provider omits it).
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Response struct {
	Text  string
	Model string
	Usage Usage
}

// Provider turns a prompt into text. Concrete backends live in their own files.
type Provider interface {
	Name() string
	Complete(ctx context.Context, req Request) (Response, error)
}

var active Provider

// Init builds the configured provider. Empty provider = AI disabled (not fatal),
// so the app runs without a model like it runs without NATS.
func Init() {
	switch strings.ToLower(config.Config.LLM.Provider) {
	case "huggingface":
		active = newHuggingFace()
	case "":
		active = nil
		logger.Info("llm: no provider configured — AI assistant disabled")
		return
	default:
		logger.Fatalf("llm: unknown provider %q", config.Config.LLM.Provider)
	}
	logger.Infof("llm: using %s", active.Name())
}

// SetProvider injects a provider — for tests only (deterministic double).
func SetProvider(p Provider) { active = p }

// Enabled reports whether a provider is configured.
func Enabled() bool { return active != nil }

// Complete runs the active provider.
func Complete(ctx context.Context, req Request) (Response, error) {
	if active == nil {
		return Response{}, ErrNoProvider
	}
	return active.Complete(ctx, req)
}

// MaxIterations is the read-loop cap per user turn (default 4).
func MaxIterations() int {
	if n := config.Config.LLM.MaxIterations; n > 0 {
		return n
	}
	return 4
}
