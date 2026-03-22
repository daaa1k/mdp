#!/usr/bin/env bash
# E2E (macOS only): put image files on the clipboard as a file drop (Finder-style),
# run mdp with the local backend, and assert extensions and bytes are preserved.
# Single-file and multi-file cases. Intended for CI (macos-latest).
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "skip: e2e-macos-file-drop.sh is macOS-only (this is $(uname -s))" >&2
  exit 0
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MAC_CLIP="$ROOT/scripts/macos-set-clipboard-files.sh"

# Minimal valid images (same generator as internal/clipboard tests; PNG + JPEG differ by format).
B64_PNG='iVBORw0KGgoAAAANSUhEUgAAAAQAAAAECAIAAAAmkwkpAAAAGElEQVR4nGJhaPjPAANMDEgANwcQAAD//0tvAYpcKvb+AAAAAElFTkSuQmCC'
B64_JPEG='/9j/2wCEAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMDAsKCwsNDhIQDQ4RDgsLEBYQERMUFRUVDA8XGBYUGBIUFRQBAwQEBQQFCQUFCRQNCw0UFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFP/AABEIAAQABAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/APP6KKK/uM/jo//Z'

TMP="$(mktemp -d "${TMPDIR:-/tmp}/mdp-e2e-filedrop.XXXXXX")"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$TMP/fixtures"
b64decode() { printf '%s' "$1" | base64 -d; }
b64decode "$B64_PNG" >"$TMP/fixtures/one.png"
b64decode "$B64_PNG" >"$TMP/fixtures/a.png"
b64decode "$B64_JPEG" >"$TMP/fixtures/b.jpg"

(cd "$ROOT" && go build -o "$TMP/mdp" .)
(cd "$ROOT" && go run ./scripts/e2e-gen-webp >"$TMP/fixtures/pixel.webp")
write_config() {
  local dir="$1"
  mkdir -p "$dir/out"
  cat >"$dir/.mdp.yaml" <<'EOF'
backend: local
local:
  dir: ./out
EOF
}

assert_file_desc() {
  local path="$1"
  local want_kind="$2"
  case "$want_kind" in
    png)
      file "$path" | grep -qiE 'PNG|image/png' || {
        echo "error: expected PNG data in $path, got: $(file -b "$path")" >&2
        return 1
      }
      ;;
    jpeg)
      file "$path" | grep -qiE 'JPEG|image/jpeg' || {
        echo "error: expected JPEG data in $path, got: $(file -b "$path")" >&2
        return 1
      }
      ;;
    webp)
      file "$path" | grep -qiE 'Web/P|WEBP|webp' || {
        echo "error: expected WebP data in $path, got: $(file -b "$path")" >&2
        return 1
      }
      ;;
    *)
      echo "assert_file_desc: unknown kind $want_kind" >&2
      return 1
      ;;
  esac
}

# --- Single file: WebP first (clean pasteboard), extension and bytes preserved ---
printf '' | pbcopy 2>/dev/null || true
osascript -e 'set the clipboard to ""' 2>/dev/null || true

SINGLE_WEBP="$TMP/case-single-webp"
write_config "$SINGLE_WEBP"
"$MAC_CLIP" "$TMP/fixtures/pixel.webp"
( cd "$SINGLE_WEBP" && "$TMP/mdp" --backend local >/dev/null )

shopt -s nullglob
sw_files=("$SINGLE_WEBP/out"/*)
shopt -u nullglob
if [[ ${#sw_files[@]} -ne 1 ]]; then
  echo "error: single WebP file drop: expected 1 output file, got ${#sw_files[@]}" >&2
  ls -la "$SINGLE_WEBP/out" >&2 || true
  exit 1
fi
SW_OUT="${sw_files[0]}"
case "$(basename "$SW_OUT")" in
  *.webp) ;;
  *)
    echo "error: expected .webp output name, got $SW_OUT" >&2
    exit 1
    ;;
esac
assert_file_desc "$SW_OUT" webp
if ! cmp -s "$TMP/fixtures/pixel.webp" "$SW_OUT"; then
  echo "error: single WebP file drop: output bytes differ from source" >&2
  exit 1
fi
echo "e2e ok: single WebP file drop -> $(basename "$SW_OUT") ($(file -b "$SW_OUT"))"

# --- Single file: one PNG, extension and bytes preserved ---
SINGLE="$TMP/case-single"
write_config "$SINGLE"
"$MAC_CLIP" "$TMP/fixtures/one.png"
( cd "$SINGLE" && "$TMP/mdp" --backend local >/dev/null )

shopt -s nullglob
s_files=("$SINGLE/out"/*)
shopt -u nullglob
if [[ ${#s_files[@]} -ne 1 ]]; then
  echo "error: single file drop: expected 1 output file, got ${#s_files[@]}" >&2
  ls -la "$SINGLE/out" >&2 || true
  exit 1
fi
S_OUT="${s_files[0]}"
case "$(basename "$S_OUT")" in
  *.png) ;;
  *)
    echo "error: expected .png output name, got $S_OUT" >&2
    exit 1
    ;;
esac
assert_file_desc "$S_OUT" png
if ! cmp -s "$TMP/fixtures/one.png" "$S_OUT"; then
  echo "error: single file drop: output bytes differ from source" >&2
  exit 1
fi
echo "e2e ok: single file drop -> $(basename "$S_OUT") ($(file -b "$S_OUT"))"

# --- Two formats from file drop: one clipboard file per mdp run (multi-file
# NSPasteboard.writeObjects is unreliable on GitHub macOS; this still checks
# PNG and JPEG bytes and extensions).
MULTI="$TMP/case-multi"
write_config "$MULTI"
printf '' | pbcopy 2>/dev/null || true
osascript -e 'set the clipboard to ""' 2>/dev/null || true
"$MAC_CLIP" "$TMP/fixtures/a.png"
( cd "$MULTI" && "$TMP/mdp" --backend local >/dev/null )
printf '' | pbcopy 2>/dev/null || true
osascript -e 'set the clipboard to ""' 2>/dev/null || true
"$MAC_CLIP" "$TMP/fixtures/b.jpg"
( cd "$MULTI" && "$TMP/mdp" --backend local >/dev/null )

shopt -s nullglob
m_files=("$MULTI/out"/*)
shopt -u nullglob
if [[ ${#m_files[@]} -ne 2 ]]; then
  echo "error: multi file drop: expected 2 output files, got ${#m_files[@]}" >&2
  ls -la "$MULTI/out" >&2 || true
  exit 1
fi

png_out=""
jpg_out=""
for f in "${m_files[@]}"; do
  case "$(basename "$f")" in
    *.png) png_out="$f" ;;
    *.jpg) jpg_out="$f" ;;
    *.jpeg) jpg_out="$f" ;;
    *)
      echo "error: unexpected output name: $f" >&2
      exit 1
      ;;
  esac
done
if [[ -z "$png_out" || -z "$jpg_out" ]]; then
  echo "error: expected one .png and one .jpg output, got:" >&2
  printf '%s\n' "${m_files[@]}" >&2
  exit 1
fi

assert_file_desc "$png_out" png
assert_file_desc "$jpg_out" jpeg
if ! cmp -s "$TMP/fixtures/a.png" "$png_out"; then
  echo "error: multi file drop: PNG output bytes differ from source" >&2
  exit 1
fi
if ! cmp -s "$TMP/fixtures/b.jpg" "$jpg_out"; then
  echo "error: multi file drop: JPEG output bytes differ from source" >&2
  exit 1
fi
echo "e2e ok: multi file drop -> $(basename "$png_out"), $(basename "$jpg_out")"
