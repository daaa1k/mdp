# mdpaste - Go implementation

## Overview

mdpaste reads an image from the clipboard, saves it to a configured backend, and prints a Markdown image link to stdout.

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

## Project structure

```
main.go                      CLI entry point (cobra)
internal/
  naming/     naming.go      Timestamp-based filename generation
  markdown/   markdown.go    Markdown image link generation
  config/     config.go      TOML config loading (project + global)
  clipboard/  clipboard.go   Cross-platform clipboard image reading
  backend/
    local.go                 Local filesystem backend
    r2.go                    Cloudflare R2 backend (S3-compatible)
    nodebb.go                NodeBB forum backend
```

## Configuration priority

```
CLI flag --backend > .mdpaste.toml (walks up from CWD) > ~/.config/mdpaste/config.toml > local
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
