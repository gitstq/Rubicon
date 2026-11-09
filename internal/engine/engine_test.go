package engine

import (
	"testing"
	"time"

	"github.com/gitstq/Rubicon/internal/card"
	"github.com/gitstq/Rubicon/internal/handoff"
	"github.com/gitstq/Rubicon/internal/store"
)

const fixedNow = "2026-11-09T10:00:00Z"

func boolPtr(b bool) *bool { return &b }

func newTestEngine(t *testing.T) (*Engine, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ts, _ := time.Parse(time.RFC3339, fixedNow)
	e := New(s, func() time.Time { return ts })
	return e, dir
}

func createInput(id string) CreateInput {
	return CreateInput{
		ID:            id,
		Title:         "Ship v1.0.0?",
		Question:      "May I tag and publish the release?",
		Context:       "All CI jobs are green.",
		OptionsSpec:   "A:Ship|high,B:Wait|low",
		DefaultAction: "B",
		Deadline:      "2026-11-10T10:00:00Z",
		Files:         []string{"CHANGELOG.md"},
		Scope:         "release tag",
		Reversible:    boolPtr(true),
		Tags:          []string{"release"},
	}
}

func TestCreateAndRebuild(t *testing.T) {
	e, _ := newTestEngine(t)
	c, err := e.Create(createInput("dc_0001"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != card.StatusPending || c.DefaultAction != "B" {
		t.Fatalf("unexpected card: %+v", c)
	}
	state, err := e.Rebuild()
	if err != nil {
		t.Fatal(err)
	}
	if len(state) != 1 || state["dc_0001"].Title != "Ship v1.0.0?" {
		t.Fatalf("rebuild failed: %+v", state)
	}
	if _, err := e.Create(createInput("dc_0001")); err == nil {
		t.Fatal("duplicate ids must be rejected")
	}
}

func TestAutoID(t *testing.T) {
	e, _ := newTestEngine(t)
	in := createInput("")
	if _, err := e.Create(in); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Create(in); err != nil {
		t.Fatal(err)
	}
	cards, _ := e.List("")
	if len(cards) != 2 || cards[0].ID != "dc_0001" || cards[1].ID != "dc_0002" {
		t.Fatalf("auto ids are wrong: %+v", cards)
	}
}

func TestDecideFlow(t *testing.T) {
	e, _ := newTestEngine(t)
	_, _ = e.Create(createInput("dc_0001"))
	c, err := e.Decide("dc_0001", "A", "alice", "Confirmed in review")
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != card.StatusApproved || c.Resolution == nil || c.Resolution.Choice != "A" {
		t.Fatalf("decision not recorded: %+v", c)
	}
	state, _ := e.Rebuild()
	if state["dc_0001"].Status != card.StatusApproved {
		t.Fatal("replayed state must reflect the decision")
	}
	if _, err := e.Decide("dc_0001", "A", "bob", ""); err == nil {
		t.Fatal("a closed card must not be decided twice")
	}
}

func TestDecideRejectsInvalidChoice(t *testing.T) {
	e, _ := newTestEngine(t)
	_, _ = e.Create(createInput("dc_0001"))
	if _, err := e.Decide("dc_0001", "Z", "alice", ""); err == nil {
		t.Fatal("undeclared option must be rejected")
	}
	if _, err := e.Decide("dc_9999", "A", "alice", ""); err == nil {
		t.Fatal("unknown card must be rejected")
	}
}

func TestRejectAndSupersede(t *testing.T) {
	e, _ := newTestEngine(t)
	_, _ = e.Create(createInput("dc_0001"))
	_, _ = e.Create(createInput("dc_0002"))
	if _, err := e.Reject("dc_0001", "bob", "out of scope"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Supersede("dc_0002", "bob", "replaced by dc_0003"); err != nil {
		t.Fatal(err)
	}
	cards, _ := e.List("")
	if cards[0].Status != card.StatusRejected || cards[1].Status != card.StatusSuperseded {
		t.Fatalf("unexpected statuses: %+v", cards)
	}
}

func TestExpireIsIdempotent(t *testing.T) {
	e, _ := newTestEngine(t)
	in := createInput("dc_0001")
	in.Deadline = "2026-11-08T10:00:00Z" // already past the fixed clock
	if _, err := e.Create(in); err != nil {
		t.Fatal(err)
	}
	expired, err := e.Expire()
	if err != nil {
		t.Fatal(err)
	}
	if len(expired) != 1 || expired[0].Status != card.StatusExpired {
		t.Fatalf("expected one expired card: %+v", expired)
	}
	again, err := e.Expire()
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatal("expire must be idempotent")
	}
	if _, err := e.Decide("dc_0001", "A", "alice", ""); err == nil {
		t.Fatal("a card past its deadline must not be decidable")
	}
}

func TestLintAndReport(t *testing.T) {
	e, _ := newTestEngine(t)
	_, _ = e.Create(createInput("dc_0001"))
	issues, err := e.Lint()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("well-formed cards should produce no findings: %+v", issues)
	}
	s, err := e.Report()
	if err != nil {
		t.Fatal(err)
	}
	if s.Total != 1 || s.Events != 1 || !s.ChainOK || s.Errors != 0 {
		t.Fatalf("unexpected report: %+v", s)
	}
}

func TestBuildHandoff(t *testing.T) {
	e, _ := newTestEngine(t)
	_, _ = e.Create(createInput("dc_0001"))
	_, _ = e.Create(createInput("dc_0002"))
	_, _ = e.Decide("dc_0001", "A", "alice", "")
	p, err := e.BuildHandoff("Stage 1 done", []handoff.Step{{Label: "Implement", Done: true}},
		[]string{"Any blocker?"}, []string{"main.go"}, []string{"Cut the release"})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Verify(); err != nil {
		t.Fatal(err)
	}
	if len(p.PendingCards) != 1 || p.PendingCards[0].ID != "dc_0002" {
		t.Fatalf("handoff must carry only unresolved cards: %+v", p.PendingCards)
	}
	if p.LedgerEvents != 3 || p.LedgerHead == "" {
		t.Fatalf("handoff ledger metadata is wrong: %+v", p)
	}
}
