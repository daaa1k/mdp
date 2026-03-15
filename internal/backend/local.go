package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalBackend saves images to the local filesystem.
type LocalBackend struct {
	Dir string
}

// Save writes data to Dir/filename and returns a relative Markdown-compatible URL.
func (b *LocalBackend) Save(data []byte, filename string) (string, error) {
	if err := os.MkdirAll(b.Dir, 0o755); err != nil {
		return "", fmt.Errorf("create directory %s: %w", b.Dir, err)
	}
	dest := filepath.Join(b.Dir, filename)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", fmt.Errorf("write file %s: %w", dest, err)
	}
	// Use forward slashes for Markdown compatibility across platforms.
	url := "./" + strings.ReplaceAll(filepath.Join(b.Dir, filename), string(filepath.Separator), "/")
	return url, nil
}
