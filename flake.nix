{
  description = "mdp — Paste clipboard image as Markdown link";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      # Read version from VERSION file to keep it in sync automatically.
      version = builtins.replaceStrings [ "\n" ] [ "" ] (builtins.readFile ./VERSION);

      # Pre-built binary hashes for each supported platform.
      # Update these whenever a new version is released:
      #   nix store prefetch-file --hash-type sha256 --json <url>
      binaryHashes = {
        "x86_64-linux"   = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
        "aarch64-darwin" = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
      };

      # Map Nix system strings to GitHub Release artifact names.
      binaryArtifacts = {
        "x86_64-linux"   = "mdp-linux-x86_64";
        "aarch64-darwin" = "mdp-macos-aarch64";
      };

      # Build a package wrapping the pre-built GitHub Release binary.
      #
      # On Linux (including WSL2 + NixOS), autoPatchelfHook rewrites the ELF
      # interpreter and RPATH so the binary works under the Nix store layout.
      mkBinaryPackage = pkgs:
        let
          system   = pkgs.stdenv.hostPlatform.system;
          artifact = binaryArtifacts.${system}
            or (throw "mdp-bin: no pre-built binary for ${system}");
          hash     = binaryHashes.${system};
          src = pkgs.fetchurl {
            url = "https://github.com/daaa1k/mdp/releases/download/v${version}/${artifact}";
            inherit hash;
          };
        in
        pkgs.stdenv.mkDerivation {
          pname = "mdp-bin";
          inherit version src;

          dontUnpack = true;

          # autoPatchelfHook is only needed on Linux (including WSL2/NixOS).
          nativeBuildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux [
            pkgs.autoPatchelfHook
          ];

          # Runtime libraries required by the Linux binary.
          buildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux [
            pkgs.glibc
          ];

          installPhase = ''
            install -Dm755 $src $out/bin/mdp
          '';
        };

      # Home Manager module — system-agnostic, exported at the top level.
      #
      # Usage in a Home Manager configuration:
      #
      #   inputs.mdp.url = "github:daaa1k/mdp";
      #
      #   { inputs, ... }: {
      #     imports = [ inputs.mdp.homeManagerModules.default ];
      #     programs.mdp = {
      #       enable = true;
      #       # Use the pre-built binary instead of building from source:
      #       # package = inputs.mdp.packages.${pkgs.system}.mdp-bin;
      #       settings = {
      #         backend = "r2";
      #         r2 = {
      #           bucket = "my-bucket";
      #           public_url = "https://cdn.example.com";
      #           endpoint = "https://<account-id>.r2.cloudflarestorage.com";
      #           # R2 credentials via R2_ACCESS_KEY_ID / R2_SECRET_ACCESS_KEY env vars.
      #         };
      #       };
      #     };
      #   }
      hmModule = { config, lib, pkgs, ... }:
        let
          cfg = config.programs.mdp;
          tomlFormat = pkgs.formats.toml { };
        in
        {
          options.programs.mdp = {
            enable = lib.mkEnableOption "mdp clipboard image to Markdown link tool";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.system}.default;
              defaultText = lib.literalExpression "mdp.packages.${pkgs.system}.default";
              description = ''
                The mdp package to install.

                Two variants are available:
                - `mdp.packages.''${pkgs.system}.default` — built from source via buildGoModule (default)
                - `mdp.packages.''${pkgs.system}.mdp-bin` — pre-built binary from GitHub Releases
                  (faster setup; no Go compilation required; supports x86_64-linux and aarch64-darwin)
              '';
            };

            settings = lib.mkOption {
              type = tomlFormat.type;
              default = { };
              description = ''
                Global configuration for mdp written to
                {file}`$XDG_CONFIG_HOME/mdp/config.toml`.

                Top-level keys:
                - `backend` — default backend (`"r2"`, `"nodebb"`, or `"local"`)
                - `r2` — Cloudflare R2 settings (`bucket`, `public_url`, `endpoint`, `prefix`)
                - `nodebb` — NodeBB settings (`url`)
                - `local` — local backend settings (`dir`)
                - `powershell_path` — WSL2: path to powershell.exe

                Credentials for R2 (`R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`) and
                NodeBB (`NODEBB_USERNAME`, `NODEBB_PASSWORD`) are read from environment
                variables and must not be placed in the config file.

                See the mdp README for the full schema reference.
              '';
              example = lib.literalExpression ''
                {
                  backend = "r2";
                  r2 = {
                    bucket = "my-bucket";
                    public_url = "https://cdn.example.com";
                    endpoint = "https://your-account-id.r2.cloudflarestorage.com";
                    prefix = "images";
                  };
                  # powershell_path = "/mnt/c/Program Files/PowerShell/7/pwsh.exe";
                }
              '';
            };
          };

          config = lib.mkIf cfg.enable {
            home.packages = [ cfg.package ];

            xdg.configFile."mdp/config.toml" = lib.mkIf (cfg.settings != { }) {
              source = tomlFormat.generate "mdp-config.toml" cfg.settings;
            };
          };
        };
    in
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        mdp = pkgs.buildGoModule {
          pname = "mdp";
          inherit version;

          src = pkgs.lib.cleanSource ./.;

          # Hash of the Go vendor directory produced by `go mod vendor`.
          # To update after changing go.mod / go.sum:
          #   1. Set vendorHash to pkgs.lib.fakeHash
          #   2. Run: nix build .#mdp 2>&1 | grep 'got:'
          #   3. Replace the value below with the hash shown in 'got:'
          vendorHash = "sha256-XYlS4P6n6azpZv+4try/tkxBTIlGUW5H8g/9xsEnQes=";

          ldflags = [ "-s" "-w" "-X main.version=${version}" ];

          meta = with pkgs.lib; {
            description = "Paste clipboard image as Markdown link";
            homepage    = "https://github.com/daaa1k/mdp";
            license     = licenses.mit;
            mainProgram = "mdp";
          };
        };

        # Format check: `gofmt -l .` must produce no output.
        fmtCheck = pkgs.runCommandLocal "mdp-fmt" { } ''
          diff <(cd ${pkgs.lib.cleanSource ./.} && ${pkgs.go}/bin/gofmt -l .) /dev/null \
            || (echo "gofmt found unformatted files — run: gofmt -w ." && exit 1)
          touch $out
        '';

        # Vet check: runs `go vet ./...` against the source.
        # Uses mdp.goModules (the vendor dir from buildGoModule) so that
        # network access is not required inside the Nix sandbox.
        vetCheck = pkgs.runCommandLocal "mdp-vet"
          { nativeBuildInputs = [ pkgs.go ]; }
          ''
            cp -r ${pkgs.lib.cleanSource ./.} src
            chmod -R u+w src
            ln -sf ${mdp.goModules} src/vendor
            cd src
            HOME=$TMPDIR CGO_ENABLED=0 GOFLAGS=-mod=vendor go vet ./...
            touch $out
          '';
      in
      {
        # --- packages ---------------------------------------------------
        packages = {
          default = mdp;
          inherit mdp;
        } // pkgs.lib.optionalAttrs (binaryArtifacts ? ${system}) {
          # mdp-bin is only exposed on platforms that have a pre-built binary.
          mdp-bin = mkBinaryPackage pkgs;
        };

        # --- checks (run by `nix flake check`) --------------------------
        checks = {
          # Build the package itself (also runs `go test`).
          inherit mdp;

          # Formatting check.
          mdp-fmt = fmtCheck;

          # Vet check.
          mdp-vet = vetCheck;
        };

        # --- devShell ---------------------------------------------------
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools   # goimports, etc.
            golangci-lint
          ];
        };
      }
    ) // {
      # --- Home Manager module (system-agnostic) ----------------------
      homeManagerModules.default = hmModule;
    };
}
