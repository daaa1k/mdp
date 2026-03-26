{
  description = "mdp — Paste clipboard image as Markdown link";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }: let
    # Read version from VERSION file to keep it in sync automatically.
    version = builtins.replaceStrings ["\n"] [""] (builtins.readFile ./VERSION);

    # Pre-built binary hashes for each supported platform.
    # Update these whenever a new version is released:
    #   nix store prefetch-file --hash-type sha256 --json <url>
    # Leave as "" to disable the mdp-bin package for that platform.
    binaryHashes = {
      "x86_64-linux" = "sha256-C4v+Vjzay9ZBxyKv2E5Q8ESF8uw59JO+eZLfjOIp0M8=";
      "aarch64-darwin" = "sha256-rEA+avjulMnQX5WU4UU/leiUpRwVbTRu6J8JeiPPt48=";
    };

    # Map Nix system strings to GitHub Release artifact names.
    binaryArtifacts = {
      "x86_64-linux" = "mdp-linux-amd64";
      "aarch64-darwin" = "mdp-darwin-arm64";
    };

    # Build a package wrapping the pre-built GitHub Release binary.
    #
    # On Linux (including WSL2 + NixOS), autoPatchelfHook rewrites the ELF
    # interpreter and RPATH so the binary works under the Nix store layout.
    mkBinaryPackage = pkgs: let
      system = pkgs.stdenv.hostPlatform.system;
      artifact = binaryArtifacts.${system}
            or (throw "mdp-bin: no pre-built binary for ${system}");
      hash = binaryHashes.${system};
      src = pkgs.fetchurl {
        url = "https://github.com/daaa1k/mdp/releases/download/v${version}/${artifact}";
        inherit hash;
      };
    in
      pkgs.stdenv.mkDerivation {
        pname = "mdp-bin";
        inherit version src;

        dontUnpack = true;

        nativeBuildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux [
          pkgs.autoPatchelfHook
        ];

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
    #     };
    #   }
    hmModule = {
      config,
      lib,
      pkgs,
      ...
    }: let
      cfg = config.programs.mdp;
      settingsFormat = pkgs.formats.yaml {};
    in {
      options.programs.mdp = {
        enable = lib.mkEnableOption "mdp";

        package = lib.mkOption {
          type = lib.types.package;
          default = self.packages.${pkgs.system}.default;
          defaultText = lib.literalExpression "mdp.packages.\${pkgs.system}.default";
          description = ''
            The mdp package to install.

            Two variants are available:
            - `mdp.packages.''${pkgs.system}.default` — built from source via buildGoModule (default)
            - `mdp.packages.''${pkgs.system}.mdp-bin` — pre-built binary from GitHub Releases
              (faster setup; no Go compilation required; supports x86_64-linux and aarch64-darwin)
          '';
        };

        settings = lib.mkOption {
          type = settingsFormat.type;
          default = {};
          description = ''
            Configuration written to <filename>$XDG_CONFIG_HOME/mdp/config.yaml</filename>.

            Available top-level keys mirror the YAML config format:
            - `backend` — storage backend: `"local"`, `"r2"`, or `"nodebb"`
            - `local.dir` — directory for local backend
            - `r2.bucket`, `r2.public_url`, `r2.endpoint`, `r2.account_id`, `r2.prefix` — R2 settings
            - `nodebb.url` — NodeBB forum URL
            - `powershell_path` — WSL2: path to powershell.exe
          '';
          example = lib.literalExpression ''
            {
              backend = "r2";
              r2 = {
                bucket = "my-bucket";
                public_url = "https://cdn.example.com";
                account_id = "abc123";
              };
            }
          '';
        };
      };

      config = lib.mkIf cfg.enable (lib.mkMerge [
        { home.packages = [cfg.package]; }
        (lib.mkIf (cfg.settings != {}) {
          xdg.configFile."mdp/config.yaml".source =
            settingsFormat.generate "mdp-config.yaml" cfg.settings;
        })
      ]);
    };
  in
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};

        # Use Go 1.26 to match go.mod requirement.
        go = pkgs.go_1_26;

        mdp = (pkgs.buildGoModule.override {inherit go;}) {
          pname = "mdp";
          inherit version;

          src = pkgs.lib.cleanSource ./.;

          # Hash of the Go vendor directory fetched by Nix.
          # vendor/ is not committed; Nix fetches modules via go.sum.
          # To update after changing go.mod / go.sum:
          #   1. Set vendorHash to pkgs.lib.fakeHash
          #   2. Run: nix build .#mdp 2>&1 | grep 'got:'
          #   3. Replace the value below with the hash shown in 'got:'
          vendorHash = "sha256-oMjqMwQZalpZmv7PPGIpREvXL/8kin5okHk4TUPS+b4=";

          ldflags = ["-s" "-w" "-X main.version=${version}"];

          meta = with pkgs.lib; {
            description = "Paste clipboard image as Markdown link";
            homepage = "https://github.com/daaa1k/mdp";
            license = licenses.mit;
            mainProgram = "mdp";
          };
        };

        # Format check: `gofmt -l` on non-vendor Go files must produce no output.
        fmtCheck = pkgs.runCommandLocal "mdp-fmt" {} ''
          src=${pkgs.lib.cleanSource ./.}
          unformatted=$(find "$src" -name '*.go' -not -path "*/vendor/*" \
            | xargs ${go}/bin/gofmt -l)
          if [ -n "$unformatted" ]; then
            echo "gofmt found unformatted files — run: gofmt -w ."
            echo "$unformatted"
            exit 1
          fi
          touch $out
        '';

        # Vet check: runs `go vet ./...` against the source.
        vetCheck =
          pkgs.runCommandLocal "mdp-vet"
          {nativeBuildInputs = [go];}
          ''
            cp -r ${pkgs.lib.cleanSource ./.} src
            chmod -R u+w src
            ln -sf ${mdp.goModules} src/vendor
            cd src
            HOME=$TMPDIR CGO_ENABLED=0 GOFLAGS=-mod=vendor go vet ./...
            touch $out
          '';
      in {
        # --- packages ---------------------------------------------------
        packages =
          {
            default = mdp;
            inherit mdp;
          }
          // pkgs.lib.optionalAttrs (binaryArtifacts ? ${system} && binaryHashes.${system} != "") {
            mdp-bin = mkBinaryPackage pkgs;
          };

        # --- checks (run by `nix flake check`) --------------------------
        checks = {
          inherit mdp;
          mdp-fmt = fmtCheck;
          mdp-vet = vetCheck;
        };

        # --- devShell ---------------------------------------------------
        devShells.default = pkgs.mkShell {
          packages = [
            go
            pkgs.gopls
            pkgs.gotools
            pkgs.golangci-lint
          ];
        };
      }
    )
    // {
      # --- Home Manager module (system-agnostic) ----------------------
      homeManagerModules.default = hmModule;
    };
}
