package backend

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalBackend_Save(t *testing.T) {
	dir := t.TempDir()
	b := &LocalBackend{Dir: filepath.Join(dir, "images")}
	data := []byte("fake image data")

	url, err := b.Save(context.Background(), data, "test.webp")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasPrefix(url, "./") {
		t.Errorf("expected URL to start with './': %s", url)
	}
	// Verify file was written.
	dest := filepath.Join(dir, "images", "test.webp")
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("file content mismatch")
	}
}

func TestLocalBackend_SaveNestedDir(t *testing.T) {
	dir := t.TempDir()
	b := &LocalBackend{Dir: filepath.Join(dir, "a", "b", "c")}
	_, err := b.Save(context.Background(), []byte("x"), "img.png")
	if err != nil {
		t.Fatalf("Save nested: %v", err)
	}
}

func TestLocalBackend_URLForwardSlashes(t *testing.T) {
	dir := t.TempDir()
	b := &LocalBackend{Dir: filepath.Join(dir, "images")}
	url, err := b.Save(context.Background(), []byte("x"), "img.webp")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if strings.Contains(url, "\\") {
		t.Errorf("URL should use forward slashes, got: %s", url)
	}
}
