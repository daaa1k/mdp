package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalBackend saves images to the local filesystem.
type LocalBackend struct {
	dir string
}

// NewLocalBackend creates a LocalBackend that saves images under dir.
func NewLocalBackend(dir string) *LocalBackend {
	return &LocalBackend{dir: dir}
}

// Save writes data to dir/filename and returns a relative Markdown-compatible URL.
func (b *LocalBackend) Save(_ context.Context, data []byte, filename string) (string, error) {
	if err := os.MkdirAll(b.dir, 0o755); err != nil {
		return "", fmt.Errorf("create directory %s: %w", b.dir, err)
	}
	dest := filepath.Join(b.dir, filename)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", fmt.Errorf("write file %s: %w", dest, err)
	}
	// Use forward slashes for Markdown compatibility across platforms.
	url := "./" + strings.ReplaceAll(filepath.Join(b.dir, filename), string(filepath.Separator), "/")
	return url, nil
}
