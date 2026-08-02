package llm

import "testing"

func TestParseProseAndActions(t *testing.T) {
	raw := "let me look:\n```action\n" +
		`{"name":"search_exercises","args":{"q":"pull up"}}` + "\n```\ndone."
	p := Parse(raw)

	if len(p.Actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(p.Actions))
	}
	if p.Actions[0].Name != "search_exercises" {
		t.Errorf("name = %q, want search_exercises", p.Actions[0].Name)
	}
	if got := string(p.Actions[0].Args); got != `{"q":"pull up"}` {
		t.Errorf("args = %s", got)
	}
	// The fenced block must be stripped from the user-facing prose.
	if want := "let me look:\n\ndone."; p.Text != want {
		t.Errorf("text = %q, want %q", p.Text, want)
	}
}

func TestParseMalformedBlockSkipped(t *testing.T) {
	raw := "hello\n```action\nnot json\n```\nmore"
	p := Parse(raw)
	if len(p.Actions) != 0 {
		t.Fatalf("got %d actions, want 0 (malformed skipped)", len(p.Actions))
	}
	if p.Text == "" {
		t.Error("prose should survive a malformed block")
	}
}

func TestParseMultipleActions(t *testing.T) {
	raw := "```action\n" + `{"name":"a"}` + "\n```\n```action\n" + `{"name":"b"}` + "\n```"
	p := Parse(raw)
	if len(p.Actions) != 2 || p.Actions[0].Name != "a" || p.Actions[1].Name != "b" {
		t.Fatalf("got %+v, want [a b] in order", p.Actions)
	}
}
