#!/usr/bin/env bash
# CI / native Linux (Wayland only): same E2E cases as macOS / WSL2 / Windows.
# Uses wl-copy / wl-paste with an existing session (WAYLAND_DISPLAY) or starts
# headless Weston when unset (e.g. GitHub Actions ubuntu-latest).
set -euo pipefail

if [[ "$(uname -s)" != Linux ]]; then
  echo "skip: ci-e2e-linux-wayland.sh is Linux-only (this is $(uname -s))" >&2
  exit 0
fi

if [[ -r /proc/version ]] && grep -qiE 'microsoft|wsl' /proc/version; then
  echo "skip: use e2e-wsl-* on WSL2; this task is for native Linux Wayland" >&2
  exit 0
fi

for bin in wl-copy wl-paste; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    echo "error: $bin not found (install wl-clipboard, e.g. apt install wl-clipboard)" >&2
    exit 1
  fi
done

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP=""
WESTON_PID=""
WESTON_STARTED=0

cleanup() {
  if [[ -n "$TMP" ]]; then
    rm -rf "$TMP"
    TMP=""
  fi
  if [[ "$WESTON_STARTED" -eq 1 && -n "$WESTON_PID" ]]; then
    kill "$WESTON_PID" 2>/dev/null || true
    wait "$WESTON_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

start_headless_weston_if_needed() {
  if [[ -n "${WAYLAND_DISPLAY:-}" ]]; then
    return 0
  fi
  if ! command -v weston >/dev/null 2>&1; then
    echo "error: WAYLAND_DISPLAY unset and weston not found (install weston or run inside a Wayland session)" >&2
    exit 1
  fi
  : "${XDG_RUNTIME_DIR:=/tmp/mdp-xdg-runtime-$$}"
  mkdir -p "$XDG_RUNTIME_DIR"
  chmod 700 "$XDG_RUNTIME_DIR"
  export XDG_RUNTIME_DIR

  export WAYLAND_DISPLAY=wayland-mdp-e2e
  weston --no-config --backend=headless --socket="$WAYLAND_DISPLAY" >/tmp/weston-mdp-e2e.log 2>&1 &
  WESTON_PID=$!
  WESTON_STARTED=1

  local sock="${XDG_RUNTIME_DIR}/${WAYLAND_DISPLAY}"
  local i
  for i in $(seq 1 200); do
    if [[ -S "$sock" ]]; then
      return 0
    fi
    sleep 0.05
  done
  echo "error: Wayland socket not ready: $sock (see /tmp/weston-mdp-e2e.log)" >&2
  cat /tmp/weston-mdp-e2e.log >&2 || true
  exit 1
}

start_headless_weston_if_needed

# --- clipboard WebP (PNG on clipboard) ---
FIXTURE_SNAP="$ROOT/e2e/fixtures/clipboard-snapshot"
if [[ ! -f "$FIXTURE_SNAP/KIND" || ! -f "$FIXTURE_SNAP/payload" ]]; then
  echo "error: missing fixture under $FIXTURE_SNAP" >&2
  exit 1
fi

TMP="$(mktemp -d "${TMPDIR:-/tmp}/mdp-ci-e2e-linux.XXXXXX")"

mkdir -p "$TMP/out"
cat >"$TMP/.mdp.yaml" <<'EOF'
backend: local
local:
  dir: ./out
EOF

cp "$FIXTURE_SNAP/payload" "$TMP/clip.png"
wl-copy --type image/png <"$TMP/clip.png"

(cd "$ROOT" && go build -o "$TMP/mdp" .)
(cd "$TMP" && "$TMP/mdp" --backend local)

shopt -s nullglob
webp_files=("$TMP/out"/*)
shopt -u nullglob
if [[ ${#webp_files[@]} -eq 0 ]]; then
  echo "error: no files written under $TMP/out" >&2
  exit 1
fi
OUT_WEBP="$(ls -t "${webp_files[@]}" | head -1)"
if ! file "$OUT_WEBP" | grep -qiE 'Web/P|WEBP|webp'; then
  echo "error: expected WebP output, got:" >&2
  file "$OUT_WEBP" >&2
  exit 1
fi
echo "e2e ok: $OUT_WEBP ($(file -b "$OUT_WEBP"))"

# --- file drop (byte-identical) ---
FIXTURE_FD="$ROOT/e2e/fixtures/filedrop"
for f in one.png one.webp two.png two.webp; do
  if [[ ! -f "$FIXTURE_FD/$f" ]]; then
    echo "error: missing fixture $FIXTURE_FD/$f" >&2
    exit 1
  fi
done

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

set_clipboard_uri_list() {
  local -a paths=("$@")
  {
    for p in "${paths[@]}"; do
      printf 'file://%s\r\n' "$p"
    done
  } | wl-copy --type text/uri-list
}

cp "$FIXTURE_FD/one.png" "$TMP/case1.png"
set_clipboard_uri_list "$TMP/case1.png"
run_mdp
assert_out_count 1
assert_outputs_match_sources "$TMP/case1.png"
echo "e2e ok: single PNG"

cp "$FIXTURE_FD/one.webp" "$TMP/case2.webp"
set_clipboard_uri_list "$TMP/case2.webp"
run_mdp
assert_out_count 1
assert_outputs_match_sources "$TMP/case2.webp"
echo "e2e ok: single WebP"

cp "$FIXTURE_FD/two.png" "$TMP/m_a.png"
cp "$FIXTURE_FD/two.webp" "$TMP/m_b.webp"
set_clipboard_uri_list "$TMP/m_a.png" "$TMP/m_b.webp"
run_mdp
assert_out_count 2
assert_outputs_match_sources "$TMP/m_a.png" "$TMP/m_b.webp"
echo "e2e ok: multiple PNG+WebP"

echo "ci-e2e-linux-wayland: all cases passed"
