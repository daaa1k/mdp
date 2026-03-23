#!/usr/bin/env bash
# E2E (WSL2 only): load the committed PNG fixture onto the Windows clipboard, run mdp with
# the local backend, and assert the saved file is WebP. Skips on non-WSL Linux and macOS.
set -euo pipefail

if ! [[ -r /proc/version ]] || ! grep -qiE 'microsoft|wsl' /proc/version; then
  echo "skip: e2e-wsl-clipboard-webp.sh is WSL2-only (this is $(uname -s))" >&2
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
FIXTURE="$ROOT/e2e/fixtures/clipboard-snapshot"

if [[ ! -f "$FIXTURE/KIND" || ! -f "$FIXTURE/payload" ]]; then
  echo "error: missing fixture under $FIXTURE" >&2
  exit 1
fi

TMP="$(mktemp -d "${TMPDIR:-/tmp}/mdp-e2e-wsl.XXXXXX")"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$TMP/out"
cat >"$TMP/.mdp.yaml" <<'EOF'
backend: local
local:
  dir: ./out
EOF

cp "$FIXTURE/payload" "$TMP/clip.png"

PS1_WIN=$(wslpath -w "$ROOT/scripts/wsl-clipboard-set-image.ps1")
CLIP_WIN=$(wslpath -w "$TMP/clip.png")

"$POWERSHELL_EXE" -Sta -NoProfile -ExecutionPolicy Bypass -File "$PS1_WIN" -ImagePath "$CLIP_WIN"

(cd "$ROOT" && go build -o "$TMP/mdp" .)
(cd "$TMP" && "$TMP/mdp" --backend local)

shopt -s nullglob
files=("$TMP/out"/*)
shopt -u nullglob
if [[ ${#files[@]} -eq 0 ]]; then
  echo "error: no files written under $TMP/out" >&2
  exit 1
fi

OUT="$(ls -t "${files[@]}" | head -1)"

if ! file "$OUT" | grep -qiE 'Web/P|WEBP|webp'; then
  echo "error: expected WebP output, got:" >&2
  file "$OUT" >&2
  exit 1
fi

echo "e2e ok: $OUT ($(file -b "$OUT"))"
