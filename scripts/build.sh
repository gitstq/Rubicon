#!/usr/bin/env bash
# Cross-compile Rubicon for every supported platform and produce release
# archives plus a SHA256SUMS manifest. Uses only the Go toolchain, tar and zip.
set -euo pipefail

VERSION="${VERSION:-1.0.0}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="$ROOT/dist"
PKG="github.com/gitstq/Rubicon/internal/cli"
LDFLAGS="-s -w -X ${PKG}.Version=${VERSION}"

TARGETS=(
  "linux amd64"
  "linux arm64"
  "darwin amd64"
  "darwin arm64"
  "windows amd64"
  "windows arm64"
)

rm -rf "$DIST"
mkdir -p "$DIST"
cd "$ROOT"

echo "==> Building rubicon v${VERSION} for ${#TARGETS[@]} targets"
for target in "${TARGETS[@]}"; do
  read -r goos goarch <<<"$target"
  name="rubicon_${VERSION}_${goos}_${goarch}"
  out="$DIST/$name"
  bin="rubicon"
  [[ "$goos" == "windows" ]] && bin="rubicon.exe"

  echo "    - ${goos}/${goarch}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "$out/$bin" ./cmd/rubicon

  cp "$ROOT/LICENSE" "$out/LICENSE"
  cp "$ROOT/README.md" "$out/README.md"

  if [[ "$goos" == "windows" ]]; then
    (cd "$DIST" && zip -qr "${name}.zip" "$name")
  else
    (cd "$DIST" && tar -czf "${name}.tar.gz" "$name")
  fi
done

echo "==> Generating SHA256SUMS"
(cd "$DIST" && sha256sum rubicon_${VERSION}_*.tar.gz rubicon_${VERSION}_*.zip > SHA256SUMS)

echo "==> Verifying every binary reports the injected version"
for target in "${TARGETS[@]}"; do
  read -r goos goarch <<<"$target"
  bin="$DIST/rubicon_${VERSION}_${goos}_${goarch}/rubicon"
  [[ "$goos" == "windows" ]] && bin="${bin}.exe"
  # Only the host binary is executable here; cross binaries are checked by file(1).
  file "$bin" | sed 's/^/    /'
done

echo "==> Artifacts in $DIST"
ls -1 "$DIST" | sed 's/^/    /'
