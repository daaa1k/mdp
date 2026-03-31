#!/usr/bin/env bash
# scripts/setup.sh
#
# Install all development dependencies via mise and prek.
#
# Usage:
#   scripts/setup.sh

set -Eeuo pipefail

trap 'echo "error: setup failed at line $LINENO" >&2' ERR

require_cmd() {
  command -v "$1" >/dev/null || {
    echo "error: missing dependency: $1 (https://mise.jdx.dev/getting-started.html)" >&2
    exit 1
  }
}

require_cmd mise

echo "==> mise trust"
mise trust

echo "==> mise install"
mise install

echo "==> mise lock"
mise lock

echo "==> prek install"
mise exec -- prek install

echo ""
echo "Setup complete!"
