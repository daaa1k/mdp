package main

import (
	"fmt"
	"os"

	"github.com/daaa1k/mdp/internal/backend"
	"github.com/daaa1k/mdp/internal/clipboard"
	"github.com/daaa1k/mdp/internal/config"
	"github.com/daaa1k/mdp/internal/markdown"
	"github.com/daaa1k/mdp/internal/naming"
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3"
var version = "dev"

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var backendFlag string
	var debug bool

	cmd := &cobra.Command{
		Use:     "mdp",
		Short:   "Paste clipboard image as Markdown link",
		Long:    "Reads an image from the clipboard, saves it to the configured backend, and prints a Markdown image link to stdout.",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(backendFlag, debug)
		},
	}

	cmd.Flags().StringVar(&backendFlag, "backend", "", "storage backend: local, r2, nodebb")
	cmd.Flags().BoolVar(&debug, "debug", false, "enable debug output to stderr")

	return cmd
}

func run(backendFlag string, debug bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if backendFlag != "" {
		cfg.CLIBackend = config.BackendType(backendFlag)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[debug] effective backend: %s\n", cfg.EffectiveBackend())
	}

	images, err := clipboard.GetImages(cfg.EffectivePowerShellPath())
	if err != nil {
		return fmt.Errorf("clipboard: %w", err)
	}
	if len(images) == 0 {
		return fmt.Errorf("no images found in clipboard")
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[debug] found %d image(s) in clipboard\n", len(images))
	}

	for i, img := range images {
		filename := filenameFor(i+1, len(images), img.Ext)

		url, err := saveImage(cfg, img.Data, filename, debug)
		if err != nil {
			return fmt.Errorf("save image: %w", err)
		}

		fmt.Println(markdown.Generate(url))
	}

	return nil
}

// filenameFor returns a filename using plain timestamp for single uploads,
// or timestamp+index for multi-image uploads.
func filenameFor(i, total int, ext string) string {
	if total == 1 {
		return naming.Generate(ext)
	}
	return naming.GenerateN(i, ext)
}

// saveImage routes to the appropriate backend and returns the resulting URL.
func saveImage(cfg *config.Config, data []byte, filename string, debug bool) (string, error) {
	switch cfg.EffectiveBackend() {
	case config.BackendR2:
		r2cfg := cfg.EffectiveR2()
		if debug {
			fmt.Fprintf(os.Stderr, "[debug] uploading to R2 bucket=%s prefix=%s\n", r2cfg.Bucket, r2cfg.Prefix)
		}
		b, err := backend.NewR2Backend(r2cfg.Bucket, r2cfg.PublicURL, r2cfg.Endpoint, r2cfg.Prefix)
		if err != nil {
			return "", err
		}
		return b.Save(data, filename)

	case config.BackendNodeBB:
		nbcfg := cfg.EffectiveNodeBB()
		if debug {
			fmt.Fprintf(os.Stderr, "[debug] uploading to NodeBB url=%s\n", nbcfg.URL)
		}
		b, err := backend.NewNodeBBBackend(nbcfg.URL)
		if err != nil {
			return "", err
		}
		return b.Save(data, filename)

	default: // local
		dir := cfg.EffectiveLocalDir()
		if debug {
			fmt.Fprintf(os.Stderr, "[debug] saving locally to dir=%s\n", dir)
		}
		b := &backend.LocalBackend{Dir: dir}
		return b.Save(data, filename)
	}
}
