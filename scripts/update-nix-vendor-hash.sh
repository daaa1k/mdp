#!/usr/bin/env bash
# Updates vendorHash in flake.nix after go.mod / go.sum changes (for Nix buildGoModule).
# Run manually from the repo root when dependencies change, e.g.:
#   ./scripts/update-nix-vendor-hash.sh
#
# CI: .github/workflows/nix-vendor-hash.yml runs this on pull requests from
# renovate/* branches when go.mod or go.sum changes.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

extract_got() {
  # grep exits 1 when there is no match; with pipefail + set -e that must not abort the script.
  grep -Eo 'got:[[:space:]]*[^[:space:]]+' "$1" 2>/dev/null | head -1 | sed -E 's/^got:[[:space:]]*//I' | tr -d '\r' || true
}

if ! command -v nix >/dev/null 2>&1; then
  echo "error: nix is not in PATH (required to compute vendorHash)" >&2
  exit 1
fi

COMBINED="$(mktemp)"
trap 'rm -f "$COMBINED"' EXIT

set +e
nix build ".#mdp" --no-link &>"$COMBINED"
nix_rc=$?
set -e

if [[ "$nix_rc" -eq 0 ]]; then
  echo "vendorHash in flake.nix already matches go.mod / go.sum."
  exit 0
fi

got="$(extract_got "$COMBINED")"

if [[ -n "$got" ]]; then
  python3 "$SCRIPT_DIR/update_nix_vendor_hash_lib.py" apply "$got"
  nix build ".#mdp" --no-link
  echo "Verified: nix build .#mdp succeeds."
  exit 0
fi

# No "got:" in the log — often "inconsistent vendoring": go.mod/go.sum changed but vendorHash
# still matches the *old* module FOD, so Nix never reaches a fixed-output hash mismatch.
# Temporarily use fakeHash so the go-modules FOD is recomputed and emits "got:".
echo "First build did not report a FOD hash mismatch; retrying with vendorHash = pkgs.lib.fakeHash ..." >&2
python3 "$SCRIPT_DIR/update_nix_vendor_hash_lib.py" set-fake

set +e
nix build ".#mdp" --no-link &>"$COMBINED"
set -e

got="$(extract_got "$COMBINED")"
if [[ -z "$got" ]]; then
  echo "error: still no 'got:' hash after fakeHash; nix log:" >&2
  cat "$COMBINED" >&2
  exit 1
fi

python3 "$SCRIPT_DIR/update_nix_vendor_hash_lib.py" apply "$got"

nix build ".#mdp" --no-link
echo "Verified: nix build .#mdp succeeds."
