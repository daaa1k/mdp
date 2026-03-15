# mdp - Go implementation

## Overview

mdp reads an image from the clipboard, saves it to a configured backend, and prints a Markdown image link to stdout.

## Development commands

```sh
# Build
go build ./...

# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Lint
go vet ./...

# Format
gofmt -w .

# Install locally
go install .
```

### Nix

```sh
# Enter dev shell (provides go, gopls, golangci-lint)
nix develop

# Build via Nix
nix build

# Run all checks (fmt, vet, tests)
nix flake check

# Update flake inputs
nix flake update
```

### Updating vendorHash after go.mod changes

1. Set `vendorHash = pkgs.lib.fakeHash;` in `flake.nix`
2. Run `nix build .#mdp 2>&1 | grep 'got:'`
3. Replace `vendorHash` with the hash shown in `got:`

### Updating binaryHashes after a release

```sh
# Linux
nix store prefetch-file --hash-type sha256 --json \
  https://github.com/daaa1k/mdp/releases/download/vX.Y.Z/mdp-linux-x86_64

# macOS (aarch64)
nix store prefetch-file --hash-type sha256 --json \
  https://github.com/daaa1k/mdp/releases/download/vX.Y.Z/mdp-macos-aarch64
```

## Project structure

```
main.go                      CLI entry point (cobra)
internal/
  naming/     naming.go      Timestamp-based filename generation
  config/     config.go      YAML config loading (project + global)
  clipboard/  clipboard.go   Cross-platform clipboard image reading
  backend/
    backend.go               Backend interface
    local.go                 Local filesystem backend
    r2.go                    Cloudflare R2 backend (S3-compatible)
    nodebb.go                NodeBB forum backend
```

## Configuration priority

```
CLI flag --backend > .mdp.yaml (walks up from CWD) > config.yaml in OS config dir > local
```

## Environment variables

| Variable | Backend | Purpose |
|---|---|---|
| `R2_ACCESS_KEY_ID` | r2 | R2 access key |
| `R2_SECRET_ACCESS_KEY` | r2 | R2 secret key |
| `NODEBB_USERNAME` | nodebb | Forum username |
| `NODEBB_PASSWORD` | nodebb | Forum password |

## Notes on WebP output

`golang.org/x/image/webp` is a decoder only. The current implementation re-encodes images as PNG (lossless). For true WebP output, link `github.com/chai2010/webp` (requires CGO).
