package naming

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	name := Generate("webp")
	if !strings.HasSuffix(name, ".webp") {
		t.Errorf("expected .webp suffix, got %s", name)
	}
	// YYYYMMDD_HHMMSS is 15 chars
	parts := strings.SplitN(name, ".", 2)
	if len(parts[0]) != 15 {
		t.Errorf("expected 15-char timestamp, got %d: %s", len(parts[0]), parts[0])
	}
}

func TestGenerateN_single(t *testing.T) {
	name := GenerateN(1, "webp")
	if !strings.HasSuffix(name, ".webp") {
		t.Errorf("expected .webp suffix, got %s", name)
	}
	parts := strings.SplitN(name, ".", 2)
	if len(parts[0]) != 15 {
		t.Errorf("expected 15-char timestamp for n=1, got %s", parts[0])
	}
}

func TestGenerateN_multi(t *testing.T) {
	name := GenerateN(2, "png")
	if !strings.HasSuffix(name, ".png") {
		t.Errorf("expected .png suffix, got %s", name)
	}
	if !strings.Contains(name, "_2.") {
		t.Errorf("expected _2 in filename, got %s", name)
	}
}
