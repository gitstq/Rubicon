// Package handoff builds a self-contained, verifiable session handoff package.
//
// A handoff answers the question "what should the next agent session know?"
// without shipping a chat log. It contains only the unresolved decisions,
// the completed steps, the touched files and a digest of the ledger head.
package handoff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gitstq/Rubicon/internal/card"
)

// Schema is the versioned handoff format identifier.
const Schema = "rubicon.handoff/v1"

// Step is one item of work in the session plan.
type Step struct {
	Label string `json:"label"`
	Done  bool   `json:"done"`
	Note  string `json:"note,omitempty"`
}

// Package is a complete handoff document.
type Package struct {
	Schema         string      `json:"schema"`
	GeneratedAt    string      `json:"generated_at"`
	Summary        string      `json:"summary"`
	LedgerHead     string      `json:"ledger_head"`
	LedgerEvents   int         `json:"ledger_events"`
	PendingCards   []card.Card `json:"pending_cards"`
	CompletedSteps []Step      `json:"completed_steps"`
	OpenQuestions  []string    `json:"open_questions"`
	FilesTouched   []string    `json:"files_touched"`
	NextActions    []string    `json:"next_actions"`
	Digest         string      `json:"digest"`
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// canonical returns the digest input. All slice fields are normalized so the
// digest is stable regardless of the order in which the caller supplied data.
func (p Package) canonical() ([]byte, error) {
	clone := p
	clone.Digest = ""
	clone.FilesTouched = sortedCopy(p.FilesTouched)
	clone.OpenQuestions = sortedCopy(p.OpenQuestions)
	clone.NextActions = sortedCopy(p.NextActions)
	clone.PendingCards = append([]card.Card(nil), p.PendingCards...)
	sort.Slice(clone.PendingCards, func(i, j int) bool {
		return clone.PendingCards[i].ID < clone.PendingCards[j].ID
	})
	return json.Marshal(clone)
}

// Seal computes and stores the SHA-256 digest of the canonical package.
func (p *Package) Seal() error {
	b, err := p.canonical()
	if err != nil {
		return fmt.Errorf("encode handoff: %w", err)
	}
	sum := sha256.Sum256(b)
	p.Digest = hex.EncodeToString(sum[:])
	return nil
}

// Verify checks the schema and recomputes the digest. Any edit to the package —
// including reordering a list — is detected.
func (p Package) Verify() error {
	if p.Schema != Schema {
		return fmt.Errorf("unsupported handoff schema %q (expected %q)", p.Schema, Schema)
	}
	if p.Digest == "" {
		return fmt.Errorf("handoff is not sealed: digest is empty")
	}
	b, err := p.canonical()
	if err != nil {
		return fmt.Errorf("encode handoff: %w", err)
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != p.Digest {
		return fmt.Errorf("handoff digest mismatch: the package was modified after sealing")
	}
	return nil
}

// Resume renders a compact, human-readable briefing for the next session.
func (p Package) Resume() string {
	pending := len(p.PendingCards)
	done := 0
	for _, s := range p.CompletedSteps {
		if s.Done {
			done++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Handoff %s\n", p.Schema)
	fmt.Fprintf(&b, "  Generated   : %s\n", p.GeneratedAt)
	fmt.Fprintf(&b, "  Ledger head : %s (%d events)\n", shortHash(p.LedgerHead), p.LedgerEvents)
	fmt.Fprintf(&b, "  Summary     : %s\n", p.Summary)
	fmt.Fprintf(&b, "  Progress    : %d/%d steps completed\n", done, len(p.CompletedSteps))
	fmt.Fprintf(&b, "  Unresolved  : %d decision card(s) await a human\n", pending)
	for _, c := range p.PendingCards {
		fmt.Fprintf(&b, "    - %s  %s  (default: %s)\n", c.ID, c.Title, c.DefaultAction)
	}
	if len(p.OpenQuestions) > 0 {
		fmt.Fprintf(&b, "  Open questions:\n")
		for _, q := range p.OpenQuestions {
			fmt.Fprintf(&b, "    ? %s\n", q)
		}
	}
	if len(p.NextActions) > 0 {
		fmt.Fprintf(&b, "  Next actions:\n")
		for _, a := range p.NextActions {
			fmt.Fprintf(&b, "    > %s\n", a)
		}
	}
	return b.String()
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12] + "..."
	}
	return h
}

// WriteFile seals (if needed) and writes the package as indented JSON.
func WriteFile(path string, p *Package) error {
	if p.Digest == "" {
		if err := p.Seal(); err != nil {
			return err
		}
	}
	if err := p.Verify(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal handoff: %w", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("write handoff: %w", err)
	}
	return nil
}

// ReadFile loads and verifies a handoff package.
func ReadFile(path string) (Package, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Package{}, fmt.Errorf("read handoff: %w", err)
	}
	var p Package
	if err := json.Unmarshal(b, &p); err != nil {
		return Package{}, fmt.Errorf("parse handoff: %w", err)
	}
	if err := p.Verify(); err != nil {
		return Package{}, err
	}
	return p, nil
}
