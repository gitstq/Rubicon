// Package cli implements the rubicon command line interface.
//
// All commands are pure functions of the ledger plus an injectable clock
// (--now), which makes the tool deterministic and CI-friendly.
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gitstq/Rubicon/internal/card"
	"github.com/gitstq/Rubicon/internal/engine"
	"github.com/gitstq/Rubicon/internal/handoff"
	"github.com/gitstq/Rubicon/internal/store"
)

// Version is injected at build time via -ldflags "-X ...cli.Version=...".
var Version = "1.0.0"

type globals struct {
	dir    string
	format string
	now    string
	rest   []string
}

func parseGlobals(args []string) (globals, error) {
	g := globals{dir: ".rubicon", format: "table"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			g.rest = append(g.rest, a)
			continue
		}
		body := strings.TrimPrefix(a, "--")
		key, val, inline := body, "", false
		if j := strings.IndexByte(body, '='); j >= 0 {
			key, val, inline = body[:j], body[j+1:], true
		}
		switch key {
		case "dir", "format", "now":
			if !inline {
				if i+1 >= len(args) {
					return g, fmt.Errorf("--%s requires a value", key)
				}
				i++
				val = args[i]
			}
			switch key {
			case "dir":
				g.dir = val
			case "format":
				g.format = val
			case "now":
				g.now = val
			}
		default:
			g.rest = append(g.rest, a)
		}
	}
	if g.format != "table" && g.format != "json" && g.format != "md" {
		return g, fmt.Errorf("invalid --format %q (use table, json or md)", g.format)
	}
	return g, nil
}

func clock(g globals) (func() time.Time, error) {
	if g.now == "" {
		return func() time.Time { return time.Now().UTC() }, nil
	}
	t, err := time.Parse(time.RFC3339, g.now)
	if err != nil {
		return nil, fmt.Errorf("invalid --now %q: expected RFC3339", g.now)
	}
	return func() time.Time { return t.UTC() }, nil
}

func openEngine(g globals) (*engine.Engine, error) {
	now, err := clock(g)
	if err != nil {
		return nil, err
	}
	s, err := store.Open(g.dir)
	if err != nil {
		return nil, err
	}
	return engine.New(s, now), nil
}

func newFS(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// parseArgs allows positional arguments to appear before flags, which the
// standard flag package does not support. Every rubicon flag takes a value, so
// a "--key value" pair is kept together and all other bare tokens are moved
// to the end of the argument list before parsing.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(strings.SplitN(a, "=", 2)[0], "-")
		if known := fs.Lookup(name); known != nil && !strings.Contains(a, "=") && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	if err := fs.Parse(append(flags, positional...)); err != nil {
		return nil, err
	}
	return fs.Args(), nil
}

// Run dispatches a command and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	g, err := parseGlobals(args)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	if len(g.rest) == 0 {
		usage(stdout)
		return 2
	}
	cmd := g.rest[0]
	switch cmd {
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "rubicon %s\n", Version)
		return 0
	case "init":
		return cmdInit(g, stdout, stderr)
	case "card":
		return cmdCard(g, stdout, stderr)
	case "decide":
		return cmdDecide(g, stdout, stderr)
	case "reject":
		return cmdReject(g, stdout, stderr)
	case "supersede":
		return cmdSupersede(g, stdout, stderr)
	case "expire":
		return cmdExpire(g, stdout, stderr)
	case "verify":
		return cmdVerify(g, stdout, stderr)
	case "lint":
		return cmdLint(g, stdout, stderr)
	case "report":
		return cmdReport(g, stdout, stderr)
	case "handoff":
		return cmdHandoff(g, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n\n", cmd)
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `rubicon - decision cards & session handoff for AI agents

USAGE
  rubicon [--dir DIR] [--format table|json|md] [--now RFC3339] <command> [flags]

COMMANDS
  init                         Create the .rubicon workspace
  card new [flags]             Raise a new decision card
  card list [--status STATUS]  List cards
  card show <id>               Show one card in full
  decide <id> --choice K       Approve a card with option K
  reject <id>                  Reject a card
  supersede <id>               Mark a card as replaced
  expire                       Close cards past their deadline
  verify                       Verify the hash-chained ledger (CI gate)
  lint                         Lint every card (CI gate)
  report                       Full workspace report
  handoff build [flags]        Build a sealed handoff package
  handoff resume --file F      Resume from a handoff package
  version                      Print version
  help                         Show this help

EXAMPLES
  rubicon init
  rubicon card new --title "Drop legacy migration?" \
      --question "May I delete db/migrate/001.sql?" \
      --options "A:Delete|high,B:Keep|low" --default B \
      --files "db/migrate/001.sql" --reversible false
  rubicon card list --status pending
  rubicon decide dc_0001 --choice A --by alice --note "Confirmed in review"
  rubicon handoff build --summary "Refactor stage 1 done" --out handoff.json
  rubicon handoff resume --file handoff.json
  rubicon verify && rubicon lint

EXIT CODES
  0 success   1 runtime error   2 usage error   3 CI gate failed
`)
}

func cmdInit(g globals, stdout, stderr io.Writer) int {
	s, err := store.Open(g.dir)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	fmt.Fprintf(stdout, "workspace ready: %s\n", s.Path)
	return 0
}

func cmdCard(g globals, stdout, stderr io.Writer) int {
	if len(g.rest) < 2 {
		fmt.Fprintln(stderr, "error: expected `card new|list|show`")
		return 2
	}
	switch g.rest[1] {
	case "new":
		return cmdCardNew(g, stdout, stderr)
	case "list":
		return cmdCardList(g, stdout, stderr)
	case "show":
		return cmdCardShow(g, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown card subcommand %q\n", g.rest[1])
		return 2
	}
}

func cmdCardNew(g globals, stdout, stderr io.Writer) int {
	fs := newFS("card new", stderr)
	id := fs.String("id", "", "card id (auto-generated when omitted)")
	title := fs.String("title", "", "short human-readable title")
	question := fs.String("question", "", "the exact question a human must answer")
	context := fs.String("context", "", "background for the reviewer")
	options := fs.String("options", "", "KEY:LABEL[|impact],KEY:LABEL[|impact]")
	def := fs.String("default", "", "safe default option key")
	deadline := fs.String("deadline", "", "RFC3339 deadline")
	files := fs.String("files", "", "comma-separated impacted files")
	scope := fs.String("scope", "", "human-readable blast radius")
	reversible := fs.String("reversible", "", "true|false (omit for unknown)")
	tags := fs.String("tags", "", "comma-separated tags")
	if _, err := parseArgs(fs, g.rest[2:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	var rev *bool
	if *reversible != "" {
		b, err := strconv.ParseBool(*reversible)
		if err != nil {
			fmt.Fprintln(stderr, "error: --reversible must be true or false")
			return 2
		}
		rev = &b
	}
	e, err := openEngine(g)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	c, err := e.Create(engine.CreateInput{
		ID:            *id,
		Title:         *title,
		Question:      *question,
		Context:       *context,
		OptionsSpec:   *options,
		DefaultAction: *def,
		Deadline:      *deadline,
		Files:         splitCSV(*files),
		Scope:         *scope,
		Reversible:    rev,
		Tags:          splitCSV(*tags),
	})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if g.format == "json" {
		return finish(writeJSON(stdout, c), stderr)
	}
	fmt.Fprintf(stdout, "created %s  %s\n  default: %s  options: %s  fingerprint: %s\n",
		c.ID, c.Title, c.DefaultAction, strings.Join(c.OptionKeys(), "/"), c.Fingerprint()[:12])
	return 0
}

func cmdCardList(g globals, stdout, stderr io.Writer) int {
	fs := newFS("card list", stderr)
	status := fs.String("status", "", "filter by status")
	if _, err := parseArgs(fs, g.rest[2:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	e, err := openEngine(g)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	cards, err := e.List(*status)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return finish(renderCards(stdout, g.format, cards), stderr)
}

func cmdCardShow(g globals, stdout, stderr io.Writer) int {
	if len(g.rest) < 3 {
		fmt.Fprintln(stderr, "error: `card show` requires an id")
		return 2
	}
	e, err := openEngine(g)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	c, err := e.Get(g.rest[2])
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return finish(writeJSON(stdout, c), stderr)
}

func cmdDecide(g globals, stdout, stderr io.Writer) int {
	fs := newFS("decide", stderr)
	choice := fs.String("choice", "", "option key to approve")
	by := fs.String("by", "human", "who decided")
	note := fs.String("note", "", "reason")
	if _, err := parseArgs(fs, g.rest[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, "error: `decide` requires a card id")
		return 2
	}
	if *choice == "" {
		fmt.Fprintln(stderr, "error: --choice is required")
		return 2
	}
	e, err := openEngine(g)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	c, err := e.Decide(fs.Arg(0), *choice, *by, *note)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	fmt.Fprintf(stdout, "approved %s -> %s (by %s)\n", c.ID, *choice, *by)
	return 0
}

func cmdReject(g globals, stdout, stderr io.Writer) int {
	fs := newFS("reject", stderr)
	by := fs.String("by", "human", "who rejected")
	note := fs.String("note", "", "reason")
	if _, err := parseArgs(fs, g.rest[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, "error: `reject` requires a card id")
		return 2
	}
	e, err := openEngine(g)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	c, err := e.Reject(fs.Arg(0), *by, *note)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	fmt.Fprintf(stdout, "rejected %s (by %s)\n", c.ID, *by)
	return 0
}

func cmdSupersede(g globals, stdout, stderr io.Writer) int {
	fs := newFS("supersede", stderr)
	by := fs.String("by", "human", "who superseded")
	note := fs.String("note", "", "reason")
	if _, err := parseArgs(fs, g.rest[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, "error: `supersede` requires a card id")
		return 2
	}
	e, err := openEngine(g)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	c, err := e.Supersede(fs.Arg(0), *by, *note)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	fmt.Fprintf(stdout, "superseded %s\n", c.ID)
	return 0
}

func cmdExpire(g globals, stdout, stderr io.Writer) int {
	e, err := openEngine(g)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	expired, err := e.Expire()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if g.format == "json" {
		return finish(writeJSON(stdout, expired), stderr)
	}
	fmt.Fprintf(stdout, "expired %d card(s)\n", len(expired))
	for _, c := range expired {
		fmt.Fprintf(stdout, "  %s  %s\n", c.ID, c.Title)
	}
	return 0
}

func cmdVerify(g globals, stdout, stderr io.Writer) int {
	s, err := store.Open(g.dir)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	r, err := s.Verify()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if g.format == "json" {
		if err := writeJSON(stdout, r); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
	} else {
		status := "OK"
		if !r.OK {
			status = "BROKEN"
		}
		fmt.Fprintf(stdout, "ledger %s | %d events | head %s\n", status, r.Events, short(r.Head))
		if !r.OK {
			fmt.Fprintf(stdout, "  first error: %s (event %d)\n", r.FirstError, r.BrokenAt)
		}
	}
	if !r.OK {
		return 3
	}
	return 0
}

func cmdLint(g globals, stdout, stderr io.Writer) int {
	e, err := openEngine(g)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	issues, err := e.Lint()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if err := renderIssues(stdout, g.format, issues); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	for _, i := range issues {
		if i.Severity == "error" {
			return 3
		}
	}
	return 0
}

func cmdReport(g globals, stdout, stderr io.Writer) int {
	e, err := openEngine(g)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	s, err := e.Report()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	switch g.format {
	case "json":
		return finish(writeJSON(stdout, s), stderr)
	case "md":
		fmt.Fprintf(stdout, "# Rubicon report\n\n- Generated: %s\n- Cards: %d\n- Events: %d\n- Chain: %t\n- Lint errors: %d\n- Lint warnings: %d\n- Head: `%s`\n\n## By status\n\n",
			s.Generated, s.Total, s.Events, s.ChainOK, s.Errors, s.Warnings, s.Head)
		keys := sortedKeys(s.ByStatus)
		for _, k := range keys {
			fmt.Fprintf(stdout, "- **%s**: %d\n", k, s.ByStatus[k])
		}
		return 0
	default:
		fmt.Fprintf(stdout, "cards=%d events=%d chain_ok=%t errors=%d warnings=%d head=%s\n",
			s.Total, s.Events, s.ChainOK, s.Errors, s.Warnings, short(s.Head))
		for _, k := range sortedKeys(s.ByStatus) {
			fmt.Fprintf(stdout, "  %-12s %d\n", k, s.ByStatus[k])
		}
		return 0
	}
}

func cmdHandoff(g globals, stdout, stderr io.Writer) int {
	if len(g.rest) < 2 {
		fmt.Fprintln(stderr, "error: expected `handoff build|resume`")
		return 2
	}
	switch g.rest[1] {
	case "build":
		return cmdHandoffBuild(g, stdout, stderr)
	case "resume":
		return cmdHandoffResume(g, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown handoff subcommand %q\n", g.rest[1])
		return 2
	}
}

func cmdHandoffBuild(g globals, stdout, stderr io.Writer) int {
	fs := newFS("handoff build", stderr)
	out := fs.String("out", "handoff.json", "output path")
	summary := fs.String("summary", "", "one-line session summary")
	var steps, questions, files, next multiFlag
	fs.Var(&steps, "step", "completed step (repeatable), LABEL[|note]")
	fs.Var(&questions, "question", "open question (repeatable)")
	fs.Var(&files, "file", "touched file (repeatable)")
	fs.Var(&next, "next", "next action (repeatable)")
	if _, err := parseArgs(fs, g.rest[2:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	e, err := openEngine(g)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	parsedSteps := make([]handoff.Step, 0, len(steps))
	for _, raw := range steps {
		label, note := raw, ""
		if i := strings.IndexByte(raw, '|'); i >= 0 {
			label, note = strings.TrimSpace(raw[:i]), strings.TrimSpace(raw[i+1:])
		}
		parsedSteps = append(parsedSteps, handoff.Step{Label: strings.TrimSpace(label), Done: true, Note: note})
	}
	p, err := e.BuildHandoff(*summary, parsedSteps, []string(questions), []string(files), []string(next))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if err := handoff.WriteFile(*out, &p); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if g.format == "json" {
		return finish(writeJSON(stdout, p), stderr)
	}
	fmt.Fprintf(stdout, "handoff written: %s\n  %d unresolved card(s), %d step(s), digest %s\n",
		*out, len(p.PendingCards), len(p.CompletedSteps), short(p.Digest))
	return 0
}

func cmdHandoffResume(g globals, stdout, stderr io.Writer) int {
	fs := newFS("handoff resume", stderr)
	file := fs.String("file", "handoff.json", "handoff package path")
	if _, err := parseArgs(fs, g.rest[2:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	p, err := handoff.ReadFile(*file)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if g.format == "json" {
		return finish(writeJSON(stdout, p), stderr)
	}
	fmt.Fprint(stdout, p.Resume())
	return 0
}

// ---- rendering helpers -------------------------------------------------

func writeJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func renderCards(w io.Writer, format string, cards []card.Card) error {
	switch format {
	case "json":
		return writeJSON(w, cards)
	case "md":
		fmt.Fprintln(w, "| ID | Status | Title | Default | Deadline | Options |")
		fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- |")
		for _, c := range cards {
			fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s |\n",
				c.ID, c.Status, escapeMD(c.Title), c.DefaultAction, orDash(c.Deadline), strings.Join(c.OptionKeys(), "/"))
		}
		return nil
	default:
		tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tSTATUS\tTITLE\tDEFAULT\tDEADLINE\tOPTIONS")
		for _, c := range cards {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
				c.ID, c.Status, c.Title, c.DefaultAction, orDash(c.Deadline), strings.Join(c.OptionKeys(), "/"))
		}
		return tw.Flush()
	}
}

func renderIssues(w io.Writer, format string, issues []card.Issue) error {
	switch format {
	case "json":
		return writeJSON(w, issues)
	case "md":
		if len(issues) == 0 {
			fmt.Fprintln(w, "No findings. All cards are well-formed.")
			return nil
		}
		fmt.Fprintln(w, "| Rule | Severity | Card | Message |")
		fmt.Fprintln(w, "| --- | --- | --- | --- |")
		for _, i := range issues {
			fmt.Fprintf(w, "| %s | %s | %s | %s |\n", i.Rule, i.Severity, i.CardID, escapeMD(i.Message))
		}
		return nil
	default:
		if len(issues) == 0 {
			fmt.Fprintln(w, "No findings. All cards are well-formed.")
			return nil
		}
		tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "RULE\tSEVERITY\tCARD\tMESSAGE")
		for _, i := range issues {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", i.Rule, i.Severity, i.CardID, i.Message)
		}
		return tw.Flush()
	}
}

func escapeMD(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12] + "..."
	}
	return h
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func finish(err error, stderr io.Writer) int {
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}
