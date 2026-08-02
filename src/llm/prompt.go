package llm

import (
	"fmt"
	"sort"
	"strings"
)

// BuildSystem renders the system prompt from the action catalog. It teaches the
// model the action protocol and lists the actions grouped by domain.
func BuildSystem() string {
	var b strings.Builder
	b.WriteString(`You are Coachwise's AI coaching assistant. You help build training plans and exercises and run app actions on the user's behalf.

Reply to the user in their language (default Persian). Be concise.

You never touch the database. To act, emit a fenced block:
` + "```action" + `
{"name": "<action>", "args": { ... }}
` + "```" + `
Emit one block per action. Prose outside blocks is shown to the user.
Reads run automatically and return results for you to use next. Writes are only proposed — the user must approve each one, so never assume a write happened.

Never invent ids. Use only ids that came from a read result or a previous action's result. To build a plan with exercises: first create or find each exercise to get its id, then propose create_plan (optionally with those exercise ids) or add_plan_exercise. If you don't have real exercise ids yet, create the plan empty and add exercises in later steps.

Actions:
`)

	byDomain := map[string][]Action{}
	for _, a := range catalog {
		byDomain[a.Domain] = append(byDomain[a.Domain], a)
	}
	domains := make([]string, 0, len(byDomain))
	for d := range byDomain {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	for _, d := range domains {
		fmt.Fprintf(&b, "\n[%s]\n", d)
		for _, a := range byDomain[d] {
			fmt.Fprintf(&b, "- %s (%s): %s args=%s\n", a.Name, a.Kind, a.Description, a.Params)
		}
	}
	return b.String()
}
