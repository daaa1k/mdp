# mdpaste

Paste clipboard image as Markdown link.

Reads an image from the clipboard, saves it to the configured backend, and prints a Markdown image link to stdout.

## Installation

### Homebrew (macOS / Linux)

```sh
brew install daaa1k/tap/mdpaste
```

### Go install

```sh
go install github.com/daaa1k/mdpaste@latest
```

### Download binary

Download a pre-built binary from [GitHub Releases](https://github.com/daaa1k/mdpaste/releases).

## Usage

```sh
mdpaste
```

Reads an image from the clipboard, saves it using the configured backend, and prints the Markdown image syntax to stdout.

```
![](./images/20260312_120233.webp)
```

### Options

```
--backend string   storage backend: local, r2, nodebb
--debug            enable debug output to stderr
```

### Examples

```sh
# Use configured backend
mdpaste

# Override backend via flag
mdpaste --backend r2

# Debug mode
mdpaste --debug
```

## Platform support

| Platform | Clipboard method |
|---|---|
| macOS | pngpaste (recommended), AppleScript fallback |
| Linux (Wayland) | wl-paste |
| Linux (X11) | xclip |
| WSL2 | PowerShell via interop |
| Windows | PowerShell |

> **macOS tip**: Install `pngpaste` to reliably capture screenshot clipboard images:
> ```sh
> brew install pngpaste
> ```

## Configuration

Configuration files use TOML format. Backend selection follows this priority chain:

```
CLI flag > project config (.mdpaste.toml) > global config > local (fallback)
```

### Project config (`.mdpaste.toml`)

Place this file in your project root (mdpaste walks up from the current directory to find it):

```toml
backend = "r2"

[local]
dir = "images"

[r2]
bucket = "my-bucket"
public_url = "https://cdn.example.com"
endpoint = "https://<account-id>.r2.cloudflarestorage.com"
prefix = "images"

[nodebb]
url = "https://forum.example.com"
```

### Global config

- **Unix**: `~/.config/mdpaste/config.toml` (or `$XDG_CONFIG_HOME/mdpaste/config.toml`)
- **Windows**: `%APPDATA%\mdpaste\config.toml`

```toml
backend = "local"

[local]
dir = "images"
```

#### WSL2: custom PowerShell path

```toml
powershell_path = "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"
```

## Backends

### local (default)

Saves images to a local directory and returns a relative path.

```toml
[local]
dir = "images"   # default
```

### r2

Uploads to Cloudflare R2 via the S3-compatible API.

```toml
[r2]
bucket = "my-bucket"
public_url = "https://cdn.example.com"
endpoint = "https://<account-id>.r2.cloudflarestorage.com"
prefix = "images"   # optional key prefix
```

Required environment variables:

```sh
export R2_ACCESS_KEY_ID=...
export R2_SECRET_ACCESS_KEY=...
```

### nodebb

Uploads to a NodeBB forum instance.

```toml
[nodebb]
url = "https://forum.example.com"
```

Required environment variables:

```sh
export NODEBB_USERNAME=...
export NODEBB_PASSWORD=...
```

Session cookies are cached so you only need to log in once:
- **Unix**: `~/.cache/mdpaste/nodebb_cookies.json`
- **Windows**: `%LOCALAPPDATA%\mdpaste\nodebb_cookies.json`

## License

MIT
