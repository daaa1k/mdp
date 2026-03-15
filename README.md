# mdp

Paste clipboard image as Markdown link.

Reads an image from the clipboard, saves it to the configured backend, and prints a Markdown image link to stdout.

## Installation

### Homebrew (macOS / Linux)

```sh
brew install daaa1k/tap/mdp
```

### Go install

```sh
go install github.com/daaa1k/mdp@latest
```

### Download binary

Download a pre-built binary from [GitHub Releases](https://github.com/daaa1k/mdp/releases).

### Nix / Home Manager

Add mdp as a flake input and import the Home Manager module:

```nix
# flake.nix
inputs.mdp.url = "github:daaa1k/mdp";
```

```nix
# home.nix
{ inputs, pkgs, ... }: {
  imports = [ inputs.mdp.homeManagerModules.default ];

  programs.mdp = {
    enable = true;

    # Optional: use the pre-built binary instead of compiling from source
    # package = inputs.mdp.packages.${pkgs.system}.mdp-bin;

    settings = {
      backend = "r2";
      r2 = {
        bucket = "my-bucket";
        public_url = "https://cdn.example.com";
        endpoint = "https://<account-id>.r2.cloudflarestorage.com";
        prefix = "images";
        # R2 credentials via R2_ACCESS_KEY_ID / R2_SECRET_ACCESS_KEY env vars.
      };
    };
  };
}
```

This writes the configuration to `$XDG_CONFIG_HOME/mdp/config.yaml` automatically.
Credentials (`R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, etc.) must still be provided
via environment variables.

**Package variants:**

| Attribute | Description |
|---|---|
| `packages.${system}.default` | Built from source via `buildGoModule` |
| `packages.${system}.mdp-bin` | Pre-built binary from GitHub Releases (x86_64-linux, aarch64-darwin) |

## Usage

```sh
mdp
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
mdp

# Override backend via flag
mdp --backend r2

# Debug mode
mdp --debug
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

Configuration files use YAML format. Backend selection follows this priority chain:

```
CLI flag > project config (.mdp.yaml) > global config > local (fallback)
```

### Project config (`.mdp.yaml`)

Place this file in your project root (mdp walks up from the current directory to find it):

```yaml
backend: r2

local:
  dir: images

r2:
  bucket: my-bucket
  public_url: https://cdn.example.com
  endpoint: https://<account-id>.r2.cloudflarestorage.com
  prefix: images

nodebb:
  url: https://forum.example.com
```

### Global config

- **Unix**: `~/.config/mdp/config.yaml` (or `$XDG_CONFIG_HOME/mdp/config.yaml`)
- **macOS**: `~/Library/Application Support/mdp/config.yaml`
- **Windows**: `%APPDATA%\mdp\config.yaml`

```yaml
backend: local

local:
  dir: images
```

#### WSL2: custom PowerShell path

```yaml
powershell_path: /mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe
```

## Backends

### local (default)

Saves images to a local directory and returns a relative path.

```yaml
local:
  dir: images   # default
```

### r2

Uploads to Cloudflare R2 via the S3-compatible API.

```yaml
r2:
  bucket: my-bucket
  public_url: https://cdn.example.com
  endpoint: https://<account-id>.r2.cloudflarestorage.com
  prefix: images   # optional key prefix
```

Required environment variables:

```sh
export R2_ACCESS_KEY_ID=...
export R2_SECRET_ACCESS_KEY=...
```

### nodebb

Uploads to a NodeBB forum instance.

```yaml
nodebb:
  url: https://forum.example.com
```

Required environment variables:

```sh
export NODEBB_USERNAME=...
export NODEBB_PASSWORD=...
```

Session cookies are cached so you only need to log in once:
- **Unix**: `~/.cache/mdp/nodebb_cookies.json`
- **macOS**: `~/Library/Caches/mdp/nodebb_cookies.json`
- **Windows**: `%LOCALAPPDATA%\mdp\nodebb_cookies.json`

## License

MIT
