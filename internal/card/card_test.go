package card

import (
	"strings"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func baseCard() Card {
	return Card{
		ID:        "dc_0001",
		CreatedAt: "2026-11-09T10:00:00Z",
		Title:     "Ship v1.0.0?",
		Question:  "May I tag and publish the release?",
		Options: []Option{
			{Key: "A", Label: "Ship", Impact: "high", IsDefault: false},
			{Key: "B", Label: "Wait", Impact: "low", IsDefault: true},
		},
		Impact: Impact{
			Files:      []string{"CHANGELOG.md"},
			Reversible: boolPtr(true),
			Scope:      "release tag",
		},
		DefaultAction: "B",
		Deadline:      "2026-11-10T10:00:00Z",
		Status:        StatusPending,
		Tags:          []string{"release", "review"},
	}
}

func fatalCount(c Card) int {
	n := 0
	for _, i := range c.Validate() {
		if i.Severity == "error" {
			n++
		}
	}
	return n
}

func TestValidateAcceptsWellFormedCard(t *testing.T) {
	if n := fatalCount(baseCard()); n != 0 {
		t.Fatalf("expected a valid card, got %d errors: %+v", n, baseCard().Validate())
	}
}

func TestValidateRejectsBrokenCards(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Card)
		rule   string
	}{
		{"empty title", func(c *Card) { c.Title = "" }, "RB002"},
		{"single option", func(c *Card) { c.Options = c.Options[:1] }, "RB004"},
		{"duplicate keys", func(c *Card) { c.Options[1].Key = "A" }, "RB006"},
		{"unknown default", func(c *Card) { c.DefaultAction = "Z" }, "RB009"},
		{"missing default", func(c *Card) { c.DefaultAction = "" }, "RB008"},
		{"bad deadline", func(c *Card) { c.Deadline = "tomorrow" }, "RB011"},
		{"pending with resolution", func(c *Card) {
			c.Resolution = &Resolution{Choice: "A"}
		}, "RB012"},
		{"closed without resolution", func(c *Card) { c.Status = StatusApproved }, "RB013"},
		{"unknown status", func(c *Card) { c.Status = "weird" }, "RB014"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := baseCard()
			tc.mutate(&c)
			found := false
			for _, i := range c.Validate() {
				if i.Rule == tc.rule && i.Severity == "error" {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected rule %s, got %+v", tc.rule, c.Validate())
			}
		})
	}
}

func TestValidateWarnsOnMissingRollbackInfo(t *testing.T) {
	c := baseCard()
	c.Impact.Reversible = nil
	found := false
	for _, i := range c.Validate() {
		if i.Rule == "RB015" && i.Severity == "warn" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected RB015 warning when reversible is unset")
	}
}

func TestFingerprintIgnoresTagOrder(t *testing.T) {
	a := baseCard()
	b := baseCard()
	b.Tags = []string{"review", "release"} // same set, different order
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatalf("fingerprint must be order-independent: %s != %s", a.Fingerprint(), b.Fingerprint())
	}
	changed := baseCard()
	changed.Title = "Ship v1.0.1?"
	if a.Fingerprint() == changed.Fingerprint() {
		t.Fatal("fingerprint must change when content changes")
	}
	if len(a.Fingerprint()) != 64 {
		t.Fatalf("expected a 64-char sha256 hex, got %q", a.Fingerprint())
	}
}

func TestParseOptions(t *testing.T) {
	opts, err := ParseOptions("A:Delete the legacy migration|high,B:Keep it|low", "B")
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
	if opts[0].Key != "A" || opts[0].Label != "Delete the legacy migration" || opts[0].Impact != "high" {
		t.Fatalf("unexpected first option: %+v", opts[0])
	}
	if !opts[1].IsDefault || opts[0].IsDefault {
		t.Fatalf("is_default should only be set on B: %+v", opts)
	}
}

func TestParseOptionsRejectsInvalidInput(t *testing.T) {
	bad := []string{
		"",
		"A:Only one",
		"A:One,B",
		"A:One,A:Two",
		"A:,B:Two",
	}
	for _, spec := range bad {
		if _, err := ParseOptions(spec, ""); err == nil {
			t.Errorf("expected error for %q", spec)
		}
	}
	if _, err := ParseOptions("A:One,B:Two", "Z"); err == nil {
		t.Error("expected error when default key is not declared")
	}
}

func TestOptionKeysAndHasOption(t *testing.T) {
	c := baseCard()
	if got := strings.Join(c.OptionKeys(), ","); got != "A,B" {
		t.Fatalf("unexpected keys %q", got)
	}
	if !c.HasOption("A") || c.HasOption("Z") {
		t.Fatal("HasOption is wrong")
	}
	if !c.IsOpen() {
		t.Fatal("pending card should be open")
	}
	c.Status = StatusApproved
	if c.IsOpen() {
		t.Fatal("approved card should not be open")
	}
}
