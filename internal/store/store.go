// Package store implements the append-only, hash-chained ledger that backs
// every Rubicon workspace. Nothing is ever overwritten: a decision is a stream
// of immutable events, and the current state is obtained by replaying them.
package store

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// LedgerDir is the workspace directory created by `rubicon init`.
	LedgerDir = ".rubicon"
	// LedgerFile is the JSONL event log.
	LedgerFile = "ledger.jsonl"
	// GenesisHash is the prev_hash of the very first event.
	GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"
)

// Event is one immutable ledger entry.
type Event struct {
	Seq      int             `json:"seq"`
	PrevHash string          `json:"prev_hash"`
	Hash     string          `json:"hash"`
	Kind     string          `json:"kind"`
	TS       string          `json:"ts"`
	Payload  json.RawMessage `json:"payload"`
}

// Store owns a single ledger file.
type Store struct {
	Dir  string
	Path string
}

// Open creates the workspace (if needed) and returns a Store. It never
// truncates an existing ledger.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	p := filepath.Join(dir, LedgerFile)
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			return nil, fmt.Errorf("init ledger: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("stat ledger: %w", err)
	}
	return &Store{Dir: dir, Path: p}, nil
}

// ComputeHash is the chain rule: sha256(prev|kind|ts|payload).
func ComputeHash(prev, kind, ts string, payload []byte) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%s|", prev, kind, ts)
	_, _ = h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// Load reads all events in file order. Blank lines are ignored so a ledger
// that ends with a stray newline still parses cleanly.
func (s *Store) Load() ([]Event, error) {
	f, err := os.Open(s.Path)
	if err != nil {
		return nil, fmt.Errorf("open ledger: %w", err)
	}
	defer f.Close()

	var events []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, fmt.Errorf("line %d: invalid event JSON: %w", line, err)
		}
		events = append(events, e)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan ledger: %w", err)
	}
	return events, nil
}

// Head returns the hash and sequence number of the last event.
func (s *Store) Head() (string, int, error) {
	events, err := s.Load()
	if err != nil {
		return "", 0, err
	}
	if len(events) == 0 {
		return GenesisHash, 0, nil
	}
	last := events[len(events)-1]
	return last.Hash, last.Seq, nil
}

// Append writes one event and returns it. The payload is marshalled once and
// reused for both hashing and storage, guaranteeing byte-for-byte agreement.
func (s *Store) Append(kind, ts string, payload any) (Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal payload: %w", err)
	}
	prev, seq, err := s.Head()
	if err != nil {
		return Event{}, err
	}
	e := Event{
		Seq:      seq + 1,
		PrevHash: prev,
		Kind:     kind,
		TS:       ts,
		Payload:  json.RawMessage(raw),
	}
	e.Hash = ComputeHash(prev, kind, ts, raw)

	f, err := os.OpenFile(s.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return Event{}, fmt.Errorf("append ledger: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(e); err != nil {
		return Event{}, fmt.Errorf("write event: %w", err)
	}
	return e, nil
}

// VerifyReport summarizes a full chain verification.
type VerifyReport struct {
	OK         bool   `json:"ok"`
	Events     int    `json:"events"`
	Head       string `json:"head"`
	BrokenAt   int    `json:"broken_at,omitempty"`
	FirstError string `json:"first_error,omitempty"`
}

// Verify replays the chain, checking sequence continuity, prev_hash linkage,
// and every event hash. It is deliberately read-only.
func (s *Store) Verify() (VerifyReport, error) {
	events, err := s.Load()
	if err != nil {
		return VerifyReport{}, err
	}
	report := VerifyReport{OK: true, Events: len(events), Head: GenesisHash}
	prev := GenesisHash
	for i, e := range events {
		if e.Seq != i+1 {
			report.OK = false
			report.BrokenAt = e.Seq
			report.FirstError = fmt.Sprintf("sequence gap: expected %d, got %d", i+1, e.Seq)
			return report, nil
		}
		if e.PrevHash != prev {
			report.OK = false
			report.BrokenAt = e.Seq
			report.FirstError = fmt.Sprintf("prev_hash mismatch at event %d", e.Seq)
			return report, nil
		}
		if got := ComputeHash(prev, e.Kind, e.TS, []byte(e.Payload)); got != e.Hash {
			report.OK = false
			report.BrokenAt = e.Seq
			report.FirstError = fmt.Sprintf("hash mismatch at event %d (ledger was modified)", e.Seq)
			return report, nil
		}
		prev = e.Hash
	}
	report.Head = prev
	return report, nil
}
