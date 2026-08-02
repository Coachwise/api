package llm

import (
	"encoding/json"
	"regexp"
	"strings"
)

// The model emits proposed actions as fenced ```action blocks holding one JSON
// object {"name": ..., "args": {...}}. Everything outside the fences is prose
// shown to the user. A malformed block is skipped, not fatal — we degrade to
// plain chat rather than crash.

// ProposedAction is one action the model wants run. The backend executes reads;
// writes go to the client for approval. Args are validated against the catalog.
type ProposedAction struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// Parsed is the result of reading a model response.
type Parsed struct {
	Text    string           // prose for the user, action blocks stripped
	Actions []ProposedAction // proposed, in order
}

var actionBlock = regexp.MustCompile("(?s)```action\\s*(.*?)```")

// Parse splits a raw model response into user-facing prose and proposed actions.
func Parse(raw string) Parsed {
	var actions []ProposedAction
	for _, m := range actionBlock.FindAllStringSubmatch(raw, -1) {
		var a ProposedAction
		if err := json.Unmarshal([]byte(strings.TrimSpace(m[1])), &a); err != nil || a.Name == "" {
			continue // skip malformed block
		}
		actions = append(actions, a)
	}
	text := strings.TrimSpace(actionBlock.ReplaceAllString(raw, ""))
	return Parsed{Text: text, Actions: actions}
}
