package handoff

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitstq/Rubicon/internal/card"
)

func samplePackage() Package {
	return Package{
		Schema:       Schema,
		GeneratedAt:  "2026-11-09T12:00:00Z",
		Summary:      "Refactor stage 1 complete",
		LedgerHead:   strings.Repeat("a", 64),
		LedgerEvents: 7,
		PendingCards: []card.Card{
			{ID: "dc_0002", Title: "Drop legacy migration?", DefaultAction: "B", Status: card.StatusPending},
		},
		CompletedSteps: []Step{{Label: "Extract parser", Done: true}, {Label: "Add tests", Done: true}},
		OpenQuestions:  []string{"Which Go version to support?"},
		FilesTouched:   []string{"internal/parser.go", "go.mod"},
		NextActions:    []string{"Run cross-platform build"},
	}
}

func TestSealAndVerify(t *testing.T) {
	p := samplePackage()
	if err := p.Seal(); err != nil {
		t.Fatal(err)
	}
	if len(p.Digest) != 64 {
		t.Fatalf("expected sha256 hex digest, got %q", p.Digest)
	}
	if err := p.Verify(); err != nil {
		t.Fatalf("sealed package must verify: %v", err)
	}
}

func TestDigestIsOrderIndependent(t *testing.T) {
	a := samplePackage()
	b := samplePackage()
	b.FilesTouched = []string{"go.mod", "internal/parser.go"}
	b.NextActions = []string{"Run cross-platform build"}
	_ = a.Seal()
	_ = b.Seal()
	if a.Digest != b.Digest {
		t.Fatalf("digest must not depend on slice order: %s != %s", a.Digest, b.Digest)
	}
}

func TestVerifyRejectsTampering(t *testing.T) {
	p := samplePackage()
	_ = p.Seal()
	p.Summary = "Refactor stage 1 NOT complete"
	if err := p.Verify(); err == nil {
		t.Fatal("modifying the summary must invalidate the digest")
	}
}

func TestVerifyRejectsUnsealedAndWrongSchema(t *testing.T) {
	p := samplePackage()
	if err := p.Verify(); err == nil {
		t.Fatal("unsealed package must be rejected")
	}
	_ = p.Seal()
	p.Schema = "rubicon.handoff/v0"
	if err := p.Verify(); err == nil {
		t.Fatal("wrong schema must be rejected")
	}
}

func TestWriteAndReadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "handoff.json")
	p := samplePackage() // deliberately unsealed: WriteFile must seal it
	if err := WriteFile(path, &p); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != p.Summary || len(got.PendingCards) != 1 {
		t.Fatalf("round trip lost data: %+v", got)
	}
	if err := got.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestResumeBriefing(t *testing.T) {
	p := samplePackage()
	_ = p.Seal()
	text := p.Resume()
	for _, want := range []string{"Handoff", "1 decision card(s)", "dc_0002", "2/2 steps", "Next actions"} {
		if !strings.Contains(text, want) {
			t.Errorf("resume briefing missing %q:\n%s", want, text)
		}
	}
}
