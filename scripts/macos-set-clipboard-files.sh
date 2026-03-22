#!/usr/bin/env bash
# Sets the general pasteboard to a Finder-style file copy for the given POSIX
# path. macOS only. One file per invocation (E2E calls once per file for
# multi-format coverage; multi-file NSPasteboard.writeObjects is unreliable on
# GitHub macOS runners).
#
# Try several JXA strategies: writeObjects(NSURL) is ideal; if that fails (seen
# on GHA), set NSFilenamesPboardType + public.file-url so mdp can read paths.
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "error: macos-set-clipboard-files.sh is for macOS only." >&2
  exit 1
fi

if [[ $# -ne 1 ]]; then
  echo "usage: $0 FILE" >&2
  exit 1
fi

if [[ ! -f "$1" ]]; then
  echo "error: not a file: $1" >&2
  exit 1
fi

ab="$(python3 -c 'import os,sys; print(os.path.realpath(sys.argv[1]))' "$1")"

clear_clipboard() {
  printf '' | pbcopy 2>/dev/null || true
  osascript -e 'set the clipboard to ""' 2>/dev/null || true
}

set_clipboard_jxa() {
  osascript -l JavaScript - "$1" <<'JXA'
function run(argv) {
  ObjC.import("AppKit");
  var p = argv[0];
  if (!p) {
    return "0";
  }
  var pb = $.NSPasteboard.generalPasteboard;
  var url = $.NSURL.fileURLWithPath(p);
  if (pb.writeObjects([url])) {
    return "1";
  }
  var arr = $.NSArray.arrayWithObject(p);
  if (pb.setPropertyList_forType(arr, "NSFilenamesPboardType")) {
    pb.setString_forType(url.absoluteString, "public.file-url");
    return "1";
  }
  return "0";
}
JXA
}

set_clipboard_applescript() {
  osascript - "$1" <<'APPLESCRIPT'
on run argv
	set n to count argv
	if n is 0 then error "need exactly one file path"
	set the clipboard to POSIX file (item 1 of argv)
end run
APPLESCRIPT
}

clear_clipboard
jxa_out="$(set_clipboard_jxa "$ab" 2>/dev/null || printf '0')"
if [[ "$(echo "$jxa_out" | tr -d '\n\r')" != "1" ]]; then
  clear_clipboard
  set_clipboard_applescript "$ab"
fi
