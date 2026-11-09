// Package engine implements the decision workflow on top of the ledger.
//
// State is never stored directly. Rebuild() replays the event stream and
// derives the current set of cards, which means the ledger is the single
// source of truth and every state transition is auditable.
package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gitstq/Rubicon/internal/card"
	"github.com/gitstq/Rubicon/internal/handoff"
	"github.com/gitstq/Rubicon/internal/store"
)

// Event kinds recorded in the ledger.
const (
	EventCreated    = "card.created"
	EventDecided    = "card.decided"
	EventRejected   = "card.rejected"
	EventExpired    = "card.expired"
	EventSuperseded = "card.superseded"
)

// Engine drives a workspace. Now is injectable so CI runs are deterministic.
type Engine struct {
	Store *store.Store
	Now   func() time.Time
}

// New wraps a store. If now is nil, the system clock in UTC is used.
func New(s *store.Store, now func() time.Time) *Engine {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Engine{Store: s, Now: now}
}

func (e *Engine) ts() string { return e.Now().UTC().Format(time.RFC3339) }

// CreateInput is the typed input of `rubicon card new`.
type CreateInput struct {
	ID            string
	Title         string
	Question      string
	Context       string
	OptionsSpec   string
	DefaultAction string
	Deadline      string
	Files         []string
	Scope         string
	Reversible    *bool
	Tags          []string
}

// Create validates and appends a new pending decision card.
func (e *Engine) Create(in CreateInput) (card.Card, error) {
	opts, err := card.ParseOptions(in.OptionsSpec, in.DefaultAction)
	if err != nil {
		return card.Card{}, err
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = e.nextID()
	}
	c := card.Card{
		ID:            id,
		CreatedAt:     e.ts(),
		Title:         strings.TrimSpace(in.Title),
		Question:      strings.TrimSpace(in.Question),
		Context:       strings.TrimSpace(in.Context),
		Options:       opts,
		Impact:        card.Impact{Files: in.Files, Scope: in.Scope, Reversible: in.Reversible},
		DefaultAction: in.DefaultAction,
		Deadline:      in.Deadline,
		Status:        card.StatusPending,
		Tags:          in.Tags,
	}
	if problems := fatal(c.Validate()); len(problems) > 0 {
		return card.Card{}, fmt.Errorf("invalid card: %s", strings.Join(problems, "; "))
	}
	if state, err := e.Rebuild(); err == nil {
		if _, exists := state[id]; exists {
			return card.Card{}, fmt.Errorf("card id %q already exists", id)
		}
	}
	if _, err := e.Store.Append(EventCreated, e.ts(), c); err != nil {
		return card.Card{}, err
	}
	return c, nil
}

func (e *Engine) nextID() string {
	state, err := e.Rebuild()
	if err != nil {
		return "dc_0001"
	}
	max := 0
	for id := range state {
		var n int
		if _, err := fmt.Sscanf(id, "dc_%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("dc_%04d", max+1)
}

// Rebuild replays every event and returns the current card set keyed by id.
func (e *Engine) Rebuild() (map[string]card.Card, error) {
	events, err := e.Store.Load()
	if err != nil {
		return nil, err
	}
	state := make(map[string]card.Card, len(events))
	for _, ev := range events {
		var c card.Card
		if err := json.Unmarshal(ev.Payload, &c); err != nil {
			return nil, fmt.Errorf("event %d: corrupt card payload: %w", ev.Seq, err)
		}
		state[c.ID] = c
	}
	return state, nil
}

// List returns cards sorted by id, optionally filtered by status.
func (e *Engine) List(status string) ([]card.Card, error) {
	state, err := e.Rebuild()
	if err != nil {
		return nil, err
	}
	out := make([]card.Card, 0, len(state))
	for _, c := range state {
		if status == "" || string(c.Status) == status {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Get returns a single card by id.
func (e *Engine) Get(id string) (card.Card, error) {
	state, err := e.Rebuild()
	if err != nil {
		return card.Card{}, err
	}
	c, ok := state[id]
	if !ok {
		return card.Card{}, fmt.Errorf("card %q not found", id)
	}
	return c, nil
}

func (e *Engine) close(id string, status card.Status, choice, by, note, kind string) (card.Card, error) {
	c, err := e.Get(id)
	if err != nil {
		return card.Card{}, err
	}
	if !c.IsOpen() {
		return card.Card{}, fmt.Errorf("card %q is already %s", id, c.Status)
	}
	if c.Deadline != "" {
		dl, err := time.Parse(time.RFC3339, c.Deadline)
		if err != nil {
			return card.Card{}, fmt.Errorf("card %q has an invalid deadline: %w", id, err)
		}
		if e.Now().After(dl) {
			return card.Card{}, fmt.Errorf("card %q passed its deadline at %s; run `rubicon expire` first", id, c.Deadline)
		}
	}
	if choice != "" && !c.HasOption(choice) {
		return card.Card{}, fmt.Errorf("choice %q is not a declared option (available: %s)", choice, strings.Join(c.OptionKeys(), ", "))
	}
	if by == "" {
		by = "human"
	}
	c.Status = status
	c.Resolution = &card.Resolution{
		Choice:    choice,
		DecidedBy: by,
		Note:      note,
		DecidedAt: e.ts(),
	}
	if _, err := e.Store.Append(kind, e.ts(), c); err != nil {
		return card.Card{}, err
	}
	return c, nil
}

// Decide approves a card with a concrete option.
func (e *Engine) Decide(id, choice, by, note string) (card.Card, error) {
	return e.close(id, card.StatusApproved, choice, by, note, EventDecided)
}

// Reject closes a card without selecting an option.
func (e *Engine) Reject(id, by, note string) (card.Card, error) {
	return e.close(id, card.StatusRejected, "", by, note, EventRejected)
}

// Supersede marks a card as replaced by a later decision.
func (e *Engine) Supersede(id, by, note string) (card.Card, error) {
	return e.close(id, card.StatusSuperseded, "", by, note, EventSuperseded)
}

// Expire closes every pending card whose deadline has passed. It is
// idempotent: running it twice produces no additional events.
func (e *Engine) Expire() ([]card.Card, error) {
	state, err := e.Rebuild()
	if err != nil {
		return nil, err
	}
	now := e.Now()
	var out []card.Card
	for _, c := range state {
		if !c.IsOpen() || c.Deadline == "" {
			continue
		}
		dl, err := time.Parse(time.RFC3339, c.Deadline)
		if err != nil {
			return nil, fmt.Errorf("card %q has an invalid deadline: %w", c.ID, err)
		}
		if !now.After(dl) {
			continue
		}
		c.Status = card.StatusExpired
		c.Resolution = &card.Resolution{
			DecidedBy: "system",
			Note:      "deadline reached without a human decision",
			DecidedAt: e.ts(),
		}
		if _, err := e.Store.Append(EventExpired, e.ts(), c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Lint returns all structural findings across all cards.
func (e *Engine) Lint() ([]card.Issue, error) {
	cards, err := e.List("")
	if err != nil {
		return nil, err
	}
	var issues []card.Issue
	for _, c := range cards {
		issues = append(issues, c.Validate()...)
	}
	return issues, nil
}

// Stats summarizes the workspace, used by `rubicon report`.
type Stats struct {
	Total     int            `json:"total"`
	ByStatus  map[string]int `json:"by_status"`
	Events    int            `json:"events"`
	Head      string         `json:"head"`
	ChainOK   bool           `json:"chain_ok"`
	Errors    int            `json:"lint_errors"`
	Warnings  int            `json:"lint_warnings"`
	Generated string         `json:"generated_at"`
}

// Report builds a full workspace report.
func (e *Engine) Report() (Stats, error) {
	cards, err := e.List("")
	if err != nil {
		return Stats{}, err
	}
	issues, err := e.Lint()
	if err != nil {
		return Stats{}, err
	}
	vr, err := e.Store.Verify()
	if err != nil {
		return Stats{}, err
	}
	s := Stats{
		Total:     len(cards),
		ByStatus:  map[string]int{},
		Events:    vr.Events,
		Head:      vr.Head,
		ChainOK:   vr.OK,
		Generated: e.ts(),
	}
	for _, c := range cards {
		s.ByStatus[string(c.Status)]++
	}
	for _, i := range issues {
		if i.Severity == "error" {
			s.Errors++
		} else {
			s.Warnings++
		}
	}
	return s, nil
}

// BuildHandoff creates a sealed handoff package from the current state.
func (e *Engine) BuildHandoff(summary string, steps []handoff.Step, open, files, next []string) (handoff.Package, error) {
	pending, err := e.List(string(card.StatusPending))
	if err != nil {
		return handoff.Package{}, err
	}
	vr, err := e.Store.Verify()
	if err != nil {
		return handoff.Package{}, err
	}
	p := handoff.Package{
		Schema:         handoff.Schema,
		GeneratedAt:    e.ts(),
		Summary:        summary,
		LedgerHead:     vr.Head,
		LedgerEvents:   vr.Events,
		PendingCards:   pending,
		CompletedSteps: steps,
		OpenQuestions:  open,
		FilesTouched:   files,
		NextActions:    next,
	}
	if err := p.Seal(); err != nil {
		return handoff.Package{}, err
	}
	return p, nil
}

func fatal(issues []card.Issue) []string {
	var out []string
	for _, i := range issues {
		if i.Severity == "error" {
			out = append(out, fmt.Sprintf("%s %s", i.Rule, i.Message))
		}
	}
	return out
}
