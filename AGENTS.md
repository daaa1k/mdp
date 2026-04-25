<<<<<<< before updating
# Agent notes

## Language

All comments and documentation in this repository must be written in English.

## Cursor Cloud specific instructions

### Service overview

mdp is a single Go CLI binary with no runtime service dependencies. All development commands (`go build`, `go test`, `go vet`, `gofmt`) work with just the Go toolchain. See `CLAUDE.md` for the full command list.

### Running and testing

- `go test -race ./...` runs the full test suite. Clipboard integration tests auto-skip when `WAYLAND_DISPLAY`/`DISPLAY` is not set or `wl-paste`/`xclip` is unavailable; this is expected in headless environments.
- `go vet ./...` is the primary lint check. `golangci-lint` (via `mise.toml`) is optional for deeper analysis.
- The CLI itself requires a system clipboard to do useful work (`mdp --backend local`). In headless Cloud VMs, test the build with `go build -o mdp . && ./mdp --help` and validate correctness via the unit/integration test suite.

### Non-obvious notes

- No CGO is required; the WebP encoder is pure Go.
- `mise.toml` defines convenience tasks (`mise run ci`, `mise run test`, etc.) but all underlying commands are plain `go` and can be run directly.
- The `.pre-commit-config.yaml` hooks call `mise run fmt` and `mise run lint`. These are not installed in Cloud VMs by default; the equivalent direct commands are `gofmt -w .` and `go vet ./...`.
=======
# Agent instructions

## Architecture Decision Records (ADR)

### Before making technical decisions

- Always review existing ADRs under `./docs/adr/` before choosing technologies, architecture, APIs, or other long‑impact technical directions.
- Treat merged ADRs as the project’s recorded conclusions; align new work with them unless you intentionally supersede or deprecate them (see `./docs/adr/README.md`).

### When recording new decisions

- When a new meaningful technical decision is made (or when alternatives were seriously considered), add an ADR as a new Markdown file in `./docs/adr/`, following `./docs/adr/template.md` and the rules in `./docs/adr/README.md` (naming, status, alternatives, PR flow).

### References

- `./docs/adr/README.md` — operating rules and lifecycle
- `./docs/adr/template.md` — ADR structure
>>>>>>> after updating
