#!/usr/bin/env bash
# E2E (macOS only): restore a committed clipboard snapshot, run mdp with local backend,
# and assert the saved file is WebP. Intended for CI (macos-latest) and local verification.
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "skip: e2e-macos-clipboard-webp.sh is macOS-only (this is $(uname -s))" >&2
  exit 0
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FIXTURE="$ROOT/e2e/fixtures/clipboard-snapshot"

if [[ ! -f "$FIXTURE/KIND" || ! -f "$FIXTURE/payload" ]]; then
  echo "error: missing fixture under $FIXTURE" >&2
  exit 1
fi

TMP="$(mktemp -d "${TMPDIR:-/tmp}/mdp-e2e.XXXXXX")"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$TMP/out"
cat >"$TMP/.mdp.yaml" <<'EOF'
backend: local
local:
  dir: ./out
EOF

export CLIPBOARD_SNAPSHOT_DIR="$FIXTURE"
"$ROOT/scripts/clipboard-snapshot.sh" restore

# Build once from the module root, then run from $TMP so os.Getwd() matches our .mdp.yaml
# (go run -C would leave the process cwd at the module root and pick up the repo's .mdp.yaml).
(cd "$ROOT" && go build -o "$TMP/mdp" .)
(cd "$TMP" && "$TMP/mdp" --backend local)

shopt -s nullglob
files=("$TMP/out"/*)
shopt -u nullglob
if [[ ${#files[@]} -eq 0 ]]; then
  echo "error: no files written under $TMP/out" >&2
  exit 1
fi

# Newest file (timestamp in name; ls -t is fine)
OUT="$(ls -t "${files[@]}" | head -1)"

if ! file "$OUT" | grep -qiE 'Web/P|WEBP|webp'; then
  echo "error: expected WebP output, got:" >&2
  file "$OUT" >&2
  exit 1
fi

echo "e2e ok: $OUT ($(file -b "$OUT"))"
