#!/usr/bin/env bash
# E2E (WSL2 only): put PNG/WebP fixture paths on the Windows clipboard as a file drop,
# run mdp with the local backend, and assert saved files are byte-identical to the sources.
set -euo pipefail

if ! [[ -r /proc/version ]] || ! grep -qiE 'microsoft|wsl' /proc/version; then
  echo "skip: e2e-wsl-clipboard-filedrop.sh is WSL2-only (this is $(uname -s))" >&2
  exit 0
fi

if ! command -v wslpath >/dev/null 2>&1; then
  echo "skip: wslpath not found (not WSL?)" >&2
  exit 0
fi

POWERSHELL_EXE=""
if command -v powershell.exe >/dev/null 2>&1; then
  POWERSHELL_EXE="powershell.exe"
elif [[ -x /mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe ]]; then
  POWERSHELL_EXE="/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"
fi
if [[ -z "$POWERSHELL_EXE" ]]; then
  echo "error: Windows PowerShell not found (WSL interop or /mnt/c/... required)" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FIXTURE="$ROOT/e2e/fixtures/filedrop"
for f in one.png one.webp two.png two.webp; do
  if [[ ! -f "$FIXTURE/$f" ]]; then
    echo "error: missing fixture $FIXTURE/$f" >&2
    exit 1
  fi
done

TMP="$(mktemp -d "${TMPDIR:-/tmp}/mdp-e2e-wsl-filedrop.XXXXXX")"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$TMP/out"
cat >"$TMP/.mdp.yaml" <<'EOF'
backend: local
local:
  dir: ./out
EOF

(cd "$ROOT" && go build -o "$TMP/mdp" .)

PS1_WIN=$(wslpath -w "$ROOT/scripts/wsl-clipboard-set-filedrop.ps1")

set_clipboard_files() {
  local list="$TMP/filedrop-paths-win.txt"
  : >"$list"
  for f in "$@"; do
    wslpath -w "$f" >>"$list"
  done
  local list_win
  list_win=$(wslpath -w "$list")
  "$POWERSHELL_EXE" -Sta -NoProfile -ExecutionPolicy Bypass -File "$PS1_WIN" -PathsFile "$list_win"
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

echo "e2e-wsl-clipboard-filedrop: all cases passed"
