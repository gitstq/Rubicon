#!/usr/bin/env bash
# A fully deterministic end-to-end demo. Every command is pinned to the same
# --now value, so the output is byte-for-byte reproducible on any machine.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$ROOT/dist/rubicon"
WS="$(mktemp -d)"
trap 'rm -rf "$WS"' EXIT

if [[ ! -x "$BIN" ]]; then
  mkdir -p "$ROOT/dist"
  (cd "$ROOT" && go build -trimpath -o "$BIN" ./cmd/rubicon)
fi

NOW="2026-11-09T10:00:00Z"
R=("$BIN" --dir "$WS/.rubicon" --now "$NOW")

step() { printf '\n\033[1;36m==> %s\033[0m\n' "$1"; }

step "1. Initialize the workspace"
"${R[@]}" init

step "2. Raise two decision cards"
"${R[@]}" card new \
  --title "Ship v1.0.0?" \
  --question "May I tag and publish the release?" \
  --context "All CI jobs are green on every platform." \
  --options "A:Ship|high,B:Wait for one more review|low" \
  --default B --files "CHANGELOG.md" --reversible true --tags "release,review"

"${R[@]}" card new \
  --id dc_0002 \
  --title "Delete the legacy migration?" \
  --question "May I remove db/migrate/001.sql?" \
  --options "A:Delete|high,B:Keep it|low" \
  --default B --deadline "2026-11-08T10:00:00Z" \
  --scope "database" --reversible false

"${R[@]}" card new \
  --id dc_0003 \
  --title "Add a v1.1 milestone?" \
  --question "Should we open the next milestone now?" \
  --options "A:Open it|low,B:Wait until v1 ships|low" \
  --default B --scope "roadmap" --reversible true

step "3. List pending cards (Markdown)"
"${R[@]}" --format md card list --status pending

step "4. A human decides the first card"
"${R[@]}" decide dc_0001 --choice A --by alice --note "Confirmed in release review"

step "5. Expire the overdue card (dc_0003 stays pending), verify the chain and lint all cards"
"${R[@]}" expire
"${R[@]}" verify
"${R[@]}" lint

step "6. Full report"
"${R[@]}" --format md report

step "7. Build and resume a session handoff"
"${R[@]}" handoff build \
  --out "$WS/handoff.json" \
  --summary "Refactor stage 1 complete, release approved" \
  --step "Extract parser|no regressions" \
  --step "Add unit tests|race detector clean" \
  --question "Should we support Go 1.20 as well?" \
  --file "internal/store/store.go" \
  --next "Run make release" --next "Publish v1.0.0"
"${R[@]}" handoff resume --file "$WS/handoff.json"

step "8. Show that the handoff detects tampering"
python3 - "$WS/handoff.json" <<'PY'
import json, sys
p = sys.argv[1]
d = json.load(open(p, encoding="utf-8"))
d["summary"] = "tampered"
json.dump(d, open(p + ".tampered.json", "w", encoding="utf-8"), ensure_ascii=False)
PY
if "$BIN" handoff resume --file "$WS/handoff.json.tampered.json" >/dev/null 2>&1; then
  echo "ERROR: tampering was not detected" >&2
  exit 1
else
  echo "OK: modified handoff rejected (digest mismatch)"
fi

printf '\n\033[1;32mDemo complete. Workspace: %s\033[0m\n' "$WS/.rubicon"
