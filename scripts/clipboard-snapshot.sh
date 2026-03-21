#!/usr/bin/env bash
# scripts/clipboard-snapshot.sh
#
# Snapshot the clipboard to disk once, then restore the same content as many times as needed.
# (capture saves → restore puts the same data back on the clipboard each time)
#
# Platform: macOS (PNG / TIFF / picture images, plain text)
#
# Usage:
#   scripts/clipboard-snapshot.sh capture [--dir DIR]   # save current clipboard to snapshot (overwrite)
#   scripts/clipboard-snapshot.sh restore [--dir DIR]   # restore saved snapshot to clipboard (same content every time)
#
# Storage (overwritten each capture):
#   $CLIPBOARD_SNAPSHOT_DIR or $XDG_STATE_HOME/mdp/clipboard-snapshot (default: ~/.local/state/mdp/clipboard-snapshot)
#
# Example:
#   CLIPBOARD_SNAPSHOT_DIR="$HOME/.mdp-clip" scripts/clipboard-snapshot.sh capture

set -euo pipefail

usage() {
  sed -n '1,25p' "$0" | sed 's/^# \{0,1\}//' | head -n 20
}

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "error: this script is for macOS only." >&2
  exit 1
fi

: "${XDG_STATE_HOME:=$HOME/.local/state}"
SNAP_DIR="${CLIPBOARD_SNAPSHOT_DIR:-$XDG_STATE_HOME/mdp/clipboard-snapshot}"

cmd="${1:-}"
shift || true

case "$cmd" in
  capture | restore | help | -h | --help) ;;
  "")
    echo "error: subcommand required: capture | restore" >&2
    usage >&2
    exit 1
    ;;
  *)
    echo "error: unknown subcommand: $cmd" >&2
    usage >&2
    exit 1
    ;;
esac

if [[ "$cmd" == "help" || "$cmd" == "-h" || "$cmd" == "--help" ]]; then
  usage
  exit 0
fi

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir)
      if [[ -z "${2:-}" ]]; then
        echo "error: --dir requires a path" >&2
        exit 1
      fi
      SNAP_DIR="$2"
      shift 2
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

mkdir -p "$SNAP_DIR"

export CLIPBOARD_SNAPSHOT_DIR="$SNAP_DIR"

capture() {
  osascript <<'APPLESCRIPT'
on writeTextFile(pathStr, contentsStr)
  set f to POSIX file pathStr
  set fd to open for access f with write permission
  set eof fd to 0
  write contentsStr to fd as «class utf8»
  close access fd
end writeTextFile

on writeBinaryFile(pathStr, binData)
  set f to POSIX file pathStr
  set fd to open for access f with write permission
  set eof fd to 0
  write binData to fd
  close access fd
end writeBinaryFile

on run
  set snapDir to (system attribute "CLIPBOARD_SNAPSHOT_DIR")
  if snapDir is missing value or snapDir is "" then error "CLIPBOARD_SNAPSHOT_DIR is not set"
  set kindPath to snapDir & "/KIND"
  set payloadPath to snapDir & "/payload"
  try
    set imgData to the clipboard as «class PNGf»
    writeTextFile(kindPath, "png")
    writeBinaryFile(payloadPath, imgData)
    return
  end try
  try
    set imgData to the clipboard as TIFF picture
    writeTextFile(kindPath, "tiff")
    writeBinaryFile(payloadPath, imgData)
    return
  end try
  try
    set imgData to the clipboard as picture
    writeTextFile(kindPath, "tiff")
    writeBinaryFile(payloadPath, imgData)
    return
  end try
  try
    set txt to the clipboard as string
    writeTextFile(kindPath, "text")
    writeTextFile(payloadPath, txt)
    return
  end try
  error "clipboard has no PNG/TIFF/picture image or text."
end run
APPLESCRIPT
  echo "Snapshot saved: $SNAP_DIR"
}

restore() {
  if [[ ! -f "$SNAP_DIR/KIND" || ! -f "$SNAP_DIR/payload" ]]; then
    echo "error: no snapshot found. Run capture first. (${SNAP_DIR})" >&2
    exit 1
  fi
  kind="$(tr -d '\r' <"$SNAP_DIR/KIND" | head -n 1)"
  case "$kind" in
    png)
      osascript <<'APPLESCRIPT'
set snapDir to (system attribute "CLIPBOARD_SNAPSHOT_DIR")
set p to snapDir & "/payload"
set the clipboard to (read (POSIX file p) as «class PNGf»)
APPLESCRIPT
      ;;
    tiff)
      osascript <<'APPLESCRIPT'
set snapDir to (system attribute "CLIPBOARD_SNAPSHOT_DIR")
set p to snapDir & "/payload"
set the clipboard to (read (POSIX file p) as TIFF picture)
APPLESCRIPT
      ;;
    text)
      osascript <<'APPLESCRIPT'
set snapDir to (system attribute "CLIPBOARD_SNAPSHOT_DIR")
set p to snapDir & "/payload"
set t to read (POSIX file p) as «class utf8»
set the clipboard to t
APPLESCRIPT
      ;;
    *)
      echo "error: unknown KIND: $kind" >&2
      exit 1
      ;;
  esac
  echo "Clipboard restored (${kind})"
}

case "$cmd" in
  capture) capture ;;
  restore) restore ;;
esac
