#!/usr/bin/env bash
# E2E (macOS only): put PNG/WebP fixture paths on the clipboard as a file drop (Finder-style),
# run mdp with the local backend, and assert saved files are byte-identical to the sources.
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "skip: e2e-macos-clipboard-filedrop.sh is macOS-only (this is $(uname -s))" >&2
  exit 0
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FIXTURE="$ROOT/e2e/fixtures/filedrop"
for f in one.png one.webp two.png two.webp; do
  if [[ ! -f "$FIXTURE/$f" ]]; then
    echo "error: missing fixture $FIXTURE/$f" >&2
    exit 1
  fi
done

TMP="$(mktemp -d "${TMPDIR:-/tmp}/mdp-e2e-filedrop.XXXXXX")"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$TMP/out"
cat >"$TMP/.mdp.yaml" <<'EOF'
backend: local
local:
  dir: ./out
EOF

(cd "$ROOT" && go build -o "$TMP/mdp" .)

set_clipboard_files() {
  swift "$ROOT/scripts/macos-set-clipboard-filedrop.swift" -- "$@"
}

assert_out_count() {
  local want="$1"
  shopt -s nullglob
  local outs=("$TMP/out"/*)
  shopt -u nullglob
  if [[ ${#outs[@]} -ne "$want" ]]; then
    echo "error: expected $want file(s) in $TMP/out, got ${#outs[@]}" >&2
    ls -la "$TMP/out" >&2 || true
    exit 1
  fi
}

# Every source file must match exactly one output file (byte-for-byte).
assert_outputs_match_sources() {
  local -a sources=("$@")
  shopt -s nullglob
  local -a outs=("$TMP/out"/*)
  shopt -u nullglob
  if [[ ${#outs[@]} -ne ${#sources[@]} ]]; then
    echo "error: output count ${#outs[@]} != source count ${#sources[@]}" >&2
    exit 1
  fi
  for src in "${sources[@]}"; do
    local found=0
    for out in "${outs[@]}"; do
      if cmp -s "$src" "$out"; then
        found=1
        break
      fi
    done
    if [[ $found -eq 0 ]]; then
      echo "error: no output matches source $(basename "$src")" >&2
      exit 1
    fi
  done
}

run_mdp() {
  rm -rf "$TMP/out"
  mkdir -p "$TMP/out"
  (cd "$TMP" && "$TMP/mdp" --backend local)
}

# --- Single PNG ---
cp "$FIXTURE/one.png" "$TMP/case1.png"
set_clipboard_files "$TMP/case1.png"
run_mdp
assert_out_count 1
assert_outputs_match_sources "$TMP/case1.png"
echo "e2e ok: single PNG"

# --- Single WebP ---
cp "$FIXTURE/one.webp" "$TMP/case2.webp"
set_clipboard_files "$TMP/case2.webp"
run_mdp
assert_out_count 1
assert_outputs_match_sources "$TMP/case2.webp"
echo "e2e ok: single WebP"

# --- Multiple (PNG + WebP) ---
cp "$FIXTURE/two.png" "$TMP/m_a.png"
cp "$FIXTURE/two.webp" "$TMP/m_b.webp"
set_clipboard_files "$TMP/m_a.png" "$TMP/m_b.webp"
run_mdp
assert_out_count 2
assert_outputs_match_sources "$TMP/m_a.png" "$TMP/m_b.webp"
echo "e2e ok: multiple PNG+WebP"

echo "e2e-macos-clipboard-filedrop: all cases passed"
