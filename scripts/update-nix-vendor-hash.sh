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

if ! command -v nix >/dev/null 2>&1; then
  echo "error: nix is not in PATH (required to compute vendorHash)" >&2
  exit 1
fi

ERRFILE="$(mktemp)"
trap 'rm -f "$ERRFILE"' EXIT

if nix build ".#mdp" --no-link 2>"$ERRFILE"; then
  echo "vendorHash in flake.nix already matches go.mod / go.sum."
  exit 0
fi

got="$(sed -n 's/.*got:[[:space:]]*//p' "$ERRFILE" | head -1 | tr -d '\r')"
if [[ -z "$got" ]]; then
  echo "nix build failed and no 'got:' hash was found in the log:" >&2
  cat "$ERRFILE" >&2
  exit 1
fi

python3 -c "
import pathlib, re, sys
got = sys.argv[1].strip()
path = pathlib.Path('flake.nix')
text = path.read_text()

def repl(m):
    return m.group(1) + '\"' + got + '\"'

new, n = re.subn(r'(vendorHash = )\"sha256-[^\"]+\"', repl, text, count=1)
if n != 1:
    sys.exit('expected exactly one vendorHash line in flake.nix')
path.write_text(new)
print('Updated vendorHash to', got)
" "$got"

nix build ".#mdp" --no-link
echo "Verified: nix build .#mdp succeeds."
