#!/usr/bin/env bash
# Compatibility wrapper: restores a saved snapshot to the clipboard (same as clipboard-snapshot.sh restore).
set -euo pipefail
exec "$(dirname "$0")/clipboard-snapshot.sh" restore "$@"
