package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, want int, args ...string) string {
	t.Helper()
	var out, errb bytes.Buffer
	got := Run(args, &out, &errb)
	if got != want {
		t.Fatalf("Run(%v) exit=%d (want %d)\nstdout:\n%s\nstderr:\n%s", args, got, want, out.String(), errb.String())
	}
	return out.String()
}

func TestParseGlobals(t *testing.T) {
	g, err := parseGlobals([]string{"--dir=/tmp/ws", "--format", "md", "--now", "2026-11-09T10:00:00Z", "card", "list"})
	if err != nil {
		t.Fatal(err)
	}
	if g.dir != "/tmp/ws" || g.format != "md" || g.now != "2026-11-09T10:00:00Z" {
		t.Fatalf("globals parsed incorrectly: %+v", g)
	}
	if strings.Join(g.rest, " ") != "card list" {
		t.Fatalf("subcommand mangled: %+v", g.rest)
	}
	if _, err := parseGlobals([]string{"--format", "xml"}); err == nil {
		t.Fatal("invalid format must be rejected")
	}
}

func TestEndToEndWorkflow(t *testing.T) {
	dir := t.TempDir()
	ws := filepath.Join(dir, "workspace")
	now := "2026-11-09T10:00:00Z"

	run(t, 0, "--dir", ws, "init")
	run(t, 0, "--dir", ws, "--now", now, "card", "new",
		"--title", "Ship v1.0.0?", "--question", "May I tag and publish the release?",
		"--options", "A:Ship|high,B:Wait|low", "--default", "B",
		"--files", "CHANGELOG.md", "--reversible", "true", "--tags", "release")
	run(t, 0, "--dir", ws, "--now", now, "card", "new",
		"--id", "dc_0002", "--title", "Delete old migration?",
		"--question", "May I remove db/migrate/001.sql?",
		"--options", "A:Delete|high,B:Keep|low", "--default", "B",
		"--deadline", "2026-11-08T10:00:00Z", "--scope", "database")

	listJSON := run(t, 0, "--dir", ws, "--format", "json", "card", "list")
	var cards []map[string]any
	if err := json.Unmarshal([]byte(listJSON), &cards); err != nil {
		t.Fatalf("card list is not valid JSON: %v\n%s", err, listJSON)
	}
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}

	run(t, 0, "--dir", ws, "--now", now, "decide", "dc_0001", "--choice", "A", "--by", "alice", "--note", "LGTM")
	if !strings.Contains(run(t, 0, "--dir", ws, "--format", "md", "card", "list"), "dc_0001") {
		t.Fatal("markdown listing should contain the card")
	}
	run(t, 0, "--dir", ws, "verify")
	run(t, 0, "--dir", ws, "lint")
	report := run(t, 0, "--dir", ws, "--format", "md", "report")
	for _, want := range []string{"# Rubicon report", "Cards: 2", "Chain: true"} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q:\n%s", want, report)
		}
	}

	ho := filepath.Join(dir, "handoff.json")
	run(t, 0, "--dir", ws, "--now", now, "handoff", "build",
		"--out", ho, "--summary", "Stage 1 complete",
		"--step", "Implement parser|done", "--question", "Any blocker?",
		"--file", "main.go", "--next", "Cut the release")
	if _, err := os.Stat(ho); err != nil {
		t.Fatal("handoff file was not written:", err)
	}
	resume := run(t, 0, "--dir", ws, "handoff", "resume", "--file", ho)
	for _, want := range []string{"Handoff", "dc_0002", "Stage 1 complete"} {
		if !strings.Contains(resume, want) {
			t.Errorf("resume missing %q:\n%s", want, resume)
		}
	}

	expired := run(t, 0, "--dir", ws, "--now", now, "expire")
	if !strings.Contains(expired, "dc_0002") {
		t.Fatalf("expected dc_0002 to expire:\n%s", expired)
	}
	run(t, 0, "--dir", ws, "verify")
	if !strings.Contains(run(t, 0, "version"), "rubicon") {
		t.Fatal("version output is wrong")
	}
}

func TestTamperedLedgerFailsVerifyGate(t *testing.T) {
	dir := t.TempDir()
	run(t, 0, "--dir", dir, "--now", "2026-11-09T10:00:00Z", "card", "new",
		"--title", "T", "--question", "Q?", "--options", "A:One,B:Two", "--default", "B")
	path := filepath.Join(dir, "ledger.jsonl")
	raw, _ := os.ReadFile(path)
	tampered := strings.Replace(string(raw), `"title":"T"`, `"title":"HACKED"`, 1)
	if tampered == string(raw) {
		t.Fatal("test setup failed")
	}
	_ = os.WriteFile(path, []byte(tampered), 0o644)
	run(t, 3, "--dir", dir, "verify") // CI gate must fail
}

func TestUsageErrors(t *testing.T) {
	run(t, 2, "not-a-command")
	run(t, 2, "decide")       // missing id
	run(t, 2, "card", "show") // missing id
	run(t, 2, "--format", "xml", "version")
}
