// Package card defines the Decision Card: the smallest unit of a
// human-in-the-loop decision made by an AI agent.
//
// A card is a value object. It is never mutated in place; every state change
// is appended to the ledger as a new event carrying a full snapshot (see
// package store). This makes the whole history replayable and verifiable.
package card

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Status is the lifecycle state of a decision card.
type Status string

const (
	StatusPending    Status = "pending"
	StatusApproved   Status = "approved"
	StatusRejected   Status = "rejected"
	StatusExpired    Status = "expired"
	StatusSuperseded Status = "superseded"
)

// Option is one selectable answer on a card.
type Option struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Impact    string `json:"impact,omitempty"`
	IsDefault bool   `json:"is_default,omitempty"`
}

// Impact describes the blast radius of the decision.
type Impact struct {
	Files      []string `json:"files,omitempty"`
	Reversible *bool    `json:"reversible,omitempty"`
	Scope      string   `json:"scope,omitempty"`
}

// Resolution records how a card was closed.
type Resolution struct {
	Choice    string `json:"choice,omitempty"`
	DecidedBy string `json:"decided_by,omitempty"`
	Note      string `json:"note,omitempty"`
	DecidedAt string `json:"decided_at,omitempty"`
}

// Card is a single decision point raised by an agent.
type Card struct {
	ID            string      `json:"id"`
	CreatedAt     string      `json:"created_at"`
	Title         string      `json:"title"`
	Question      string      `json:"question"`
	Context       string      `json:"context,omitempty"`
	Options       []Option    `json:"options"`
	Impact        Impact      `json:"impact"`
	DefaultAction string      `json:"default_action"`
	Deadline      string      `json:"deadline,omitempty"`
	Status        Status      `json:"status"`
	Resolution    *Resolution `json:"resolution,omitempty"`
	Tags          []string    `json:"tags,omitempty"`
}

// Issue is a single lint finding produced by Validate.
type Issue struct {
	CardID   string `json:"card_id"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// OptionKeys returns the option keys in declaration order.
func (c Card) OptionKeys() []string {
	keys := make([]string, 0, len(c.Options))
	for _, o := range c.Options {
		keys = append(keys, o.Key)
	}
	return keys
}

// HasOption reports whether key is a declared option.
func (c Card) HasOption(key string) bool {
	for _, o := range c.Options {
		if o.Key == key {
			return true
		}
	}
	return false
}

// IsOpen reports whether the card still awaits a decision.
func (c Card) IsOpen() bool { return c.Status == StatusPending }

// Canonical returns a deterministic JSON encoding of the card. Tags are
// sorted so that two logically identical cards always produce the same bytes.
func (c Card) Canonical() ([]byte, error) {
	clone := c
	clone.Tags = append([]string(nil), c.Tags...)
	sort.Strings(clone.Tags)
	return json.Marshal(clone)
}

// Fingerprint is the SHA-256 of the canonical encoding.
func (c Card) Fingerprint() string {
	b, err := c.Canonical()
	if err != nil {
		// json.Marshal on this struct cannot fail; keep the API total.
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Validate checks structural invariants. It never returns an error: findings
// are returned as a list so callers can render them in any format.
func (c Card) Validate() []Issue {
	var out []Issue
	add := func(rule, severity, msg string) {
		out = append(out, Issue{CardID: c.ID, Rule: rule, Severity: severity, Message: msg})
	}

	if strings.TrimSpace(c.ID) == "" {
		add("RB001", "error", "card id must not be empty")
	}
	if strings.TrimSpace(c.Title) == "" {
		add("RB002", "error", "title must not be empty")
	}
	if strings.TrimSpace(c.Question) == "" {
		add("RB003", "error", "question must describe what the human must decide")
	}
	if len(c.Options) < 2 {
		add("RB004", "error", "at least two options are required for a real decision")
	}
	seen := map[string]bool{}
	for _, o := range c.Options {
		if strings.TrimSpace(o.Key) == "" {
			add("RB005", "error", "option key must not be empty")
		}
		if seen[o.Key] {
			add("RB006", "error", fmt.Sprintf("duplicate option key %q", o.Key))
		}
		seen[o.Key] = true
		if strings.TrimSpace(o.Label) == "" {
			add("RB007", "error", fmt.Sprintf("option %q needs a human-readable label", o.Key))
		}
	}
	if c.DefaultAction == "" {
		add("RB008", "error", "default_action is required so the agent can time out safely")
	} else if !c.HasOption(c.DefaultAction) {
		add("RB009", "error", fmt.Sprintf("default_action %q is not a declared option", c.DefaultAction))
	}
	defaults := 0
	for _, o := range c.Options {
		if o.IsDefault {
			defaults++
		}
	}
	if defaults > 1 {
		add("RB010", "error", "only one option may be marked is_default")
	}
	if c.Deadline != "" {
		if _, err := time.Parse(time.RFC3339, c.Deadline); err != nil {
			add("RB011", "error", "deadline must be an RFC3339 timestamp")
		}
	}
	switch c.Status {
	case StatusPending:
		if c.Resolution != nil {
			add("RB012", "error", "a pending card must not carry a resolution")
		}
	case StatusApproved, StatusRejected, StatusExpired, StatusSuperseded:
		if c.Resolution == nil {
			add("RB013", "error", "a closed card must carry a resolution")
		}
	default:
		add("RB014", "error", fmt.Sprintf("unknown status %q", c.Status))
	}
	if c.Impact.Reversible == nil {
		add("RB015", "warn", "impact.reversible is not declared; reviewers cannot judge rollback risk")
	}
	if len(c.Impact.Files) == 0 && strings.TrimSpace(c.Impact.Scope) == "" {
		add("RB016", "warn", "no impacted files or scope declared")
	}
	return out
}

// ParseOptions parses the compact CLI syntax used by `rubicon card new`:
//
//	"A:Delete the legacy migration|high,B:Keep it|low"
//
// Each option is "KEY:LABEL" and an optional "|impact" suffix. The option
// marked as default is supplied separately via --default.
func ParseOptions(spec, defaultKey string) ([]Option, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, fmt.Errorf("options must not be empty")
	}
	parts := strings.Split(spec, ",")
	opts := make([]Option, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		impact := ""
		if i := strings.LastIndex(p, "|"); i >= 0 {
			impact = strings.TrimSpace(p[i+1:])
			p = strings.TrimSpace(p[:i])
		}
		kv := strings.SplitN(p, ":", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid option %q: expected KEY:LABEL[|impact]", p)
		}
		key := strings.TrimSpace(kv[0])
		label := strings.TrimSpace(kv[1])
		if key == "" || label == "" {
			return nil, fmt.Errorf("invalid option %q: key and label must be non-empty", p)
		}
		if seen[key] {
			return nil, fmt.Errorf("duplicate option key %q", key)
		}
		seen[key] = true
		opts = append(opts, Option{
			Key:       key,
			Label:     label,
			Impact:    impact,
			IsDefault: key == defaultKey,
		})
	}
	if len(opts) < 2 {
		return nil, fmt.Errorf("at least two options are required, got %d", len(opts))
	}
	if defaultKey != "" && !seen[defaultKey] {
		return nil, fmt.Errorf("default option %q is not declared", defaultKey)
	}
	return opts, nil
}
