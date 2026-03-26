"""Helpers for scripts/update-nix-vendor-hash.sh (flake.nix vendorHash edits)."""

from __future__ import annotations

import pathlib
import re
import sys


def set_fake() -> None:
    path = pathlib.Path("flake.nix")
    text = path.read_text()
    new, n = re.subn(
        r"^(\s*)vendorHash = \"sha256-[^\"]+\";",
        r"\1vendorHash = pkgs.lib.fakeHash;",
        text,
        count=1,
        flags=re.M,
    )
    if n != 1:
        sys.exit('expected exactly one vendorHash = "sha256-..." line in flake.nix')
    path.write_text(new)


def apply(got: str) -> None:
    got = got.strip()
    path = pathlib.Path("flake.nix")
    text = path.read_text()

    if "pkgs.lib.fakeHash" in text:
        new, n = re.subn(
            r"^(\s*)vendorHash = pkgs\.lib\.fakeHash;",
            rf'\1vendorHash = "{got}";',
            text,
            count=1,
            flags=re.M,
        )
    else:

        def repl(m: re.Match[str]) -> str:
            return m.group(1) + '"' + got + '"'

        new, n = re.subn(r'(vendorHash = )"sha256-[^"]+"', repl, text, count=1)

    if n != 1:
        sys.exit("expected exactly one vendorHash line to replace in flake.nix")
    path.write_text(new)
    print("Updated vendorHash to", got)


def main() -> None:
    if len(sys.argv) < 2:
        sys.exit("usage: update_nix_vendor_hash_lib.py set-fake | apply <got>")
    cmd = sys.argv[1]
    if cmd == "set-fake":
        set_fake()
        print("Set vendorHash to pkgs.lib.fakeHash")
    elif cmd == "apply":
        if len(sys.argv) != 3:
            sys.exit("usage: update_nix_vendor_hash_lib.py apply <got>")
        apply(sys.argv[2])
    else:
        sys.exit(f"unknown command: {cmd}")


if __name__ == "__main__":
    main()
