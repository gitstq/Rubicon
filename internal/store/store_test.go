package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendAndVerify(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for i, payload := range []map[string]any{
		{"id": "dc_0001", "status": "pending"},
		{"id": "dc_0001", "status": "approved"},
		{"id": "dc_0002", "status": "pending"},
	} {
		e, err := s.Append("card.created", "2026-11-09T10:00:0"+string(rune('0'+i))+"Z", payload)
		if err != nil {
			t.Fatal(err)
		}
		if e.Seq != i+1 {
			t.Fatalf("expected seq %d, got %d", i+1, e.Seq)
		}
		if len(e.Hash) != 64 {
			t.Fatalf("expected sha256 hex, got %q", e.Hash)
		}
	}
	events, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	r, err := s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK || r.Events != 3 {
		t.Fatalf("chain should verify: %+v", r)
	}
	head, seq, err := s.Head()
	if err != nil {
		t.Fatal(err)
	}
	if seq != 3 || head != r.Head {
		t.Fatalf("head mismatch: %s/%d vs %s", head, seq, r.Head)
	}
}

func TestVerifyDetectsTamperedPayload(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Append("card.created", "2026-11-09T10:00:00Z", map[string]string{"note": "original"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Append("card.decided", "2026-11-09T10:01:00Z", map[string]string{"note": "approved"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, LedgerFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(raw), `"note":"original"`, `"note":"tampered"`, 1)
	if tampered == string(raw) {
		t.Fatal("test setup failed: payload string not found")
	}
	if err := os.WriteFile(path, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if r.OK {
		t.Fatal("tampering must be detected")
	}
	if r.BrokenAt != 1 || !strings.Contains(r.FirstError, "hash mismatch") {
		t.Fatalf("unexpected report: %+v", r)
	}
}

func TestVerifyDetectsDeletedMiddleEvent(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	for i := 0; i < 3; i++ {
		_, _ = s.Append("evt", "2026-11-09T10:00:00Z", map[string]int{"n": i})
	}
	path := filepath.Join(dir, LedgerFile)
	raw, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	kept := append([]string{}, lines[0], lines[2])
	_ = os.WriteFile(path, []byte(strings.Join(kept, "\n")+"\n"), 0o644)
	r, err := s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if r.OK {
		t.Fatal("removing a middle event must break the chain")
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	s1, _ := Open(dir)
	_, _ = s1.Append("evt", "2026-11-09T10:00:00Z", map[string]string{"a": "b"})
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	events, err := s2.Load()
	if err != nil || len(events) != 1 {
		t.Fatalf("reopening must not truncate the ledger: %v %d", err, len(events))
	}
}

func TestComputeHashIsDeterministic(t *testing.T) {
	a := ComputeHash(GenesisHash, "evt", "2026-11-09T10:00:00Z", []byte(`{"x":1}`))
	b := ComputeHash(GenesisHash, "evt", "2026-11-09T10:00:00Z", []byte(`{"x":1}`))
	c := ComputeHash(GenesisHash, "evt", "2026-11-09T10:00:01Z", []byte(`{"x":1}`))
	if a != b || a == c {
		t.Fatal("hash must be stable for identical input and sensitive to input")
	}
}
