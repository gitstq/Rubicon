# testdata

Unit tests create their ledgers and handoff packages in `t.TempDir()` so the
suite is hermetic and race-safe. This directory holds small, human-readable
reference fixtures used by the documentation and by `examples/demo.sh`:

- `../examples/decision-card.json` — a canonical Decision Card, useful when
  wiring Rubicon into an agent that emits cards as JSON.
- `../examples/demo.sh` — the deterministic end-to-end walkthrough.

No binary fixtures, no credentials and no network access are required to run
`go test ./...`.
