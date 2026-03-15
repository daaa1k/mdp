package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEffectiveBackend_CLIOverridesAll(t *testing.T) {
	cfg := &Config{
		CLIBackend: BackendR2,
		Project:    &ProjectConfig{Backend: BackendNodeBB},
		Global:     &GlobalConfig{Backend: BackendLocal},
	}
	if got := cfg.EffectiveBackend(); got != BackendR2 {
		t.Errorf("want %s, got %s", BackendR2, got)
	}
}

func TestEffectiveBackend_ProjectOverridesGlobal(t *testing.T) {
	cfg := &Config{
		Project: &ProjectConfig{Backend: BackendNodeBB},
		Global:  &GlobalConfig{Backend: BackendLocal},
	}
	if got := cfg.EffectiveBackend(); got != BackendNodeBB {
		t.Errorf("want %s, got %s", BackendNodeBB, got)
	}
}

func TestEffectiveBackend_FallbackToLocal(t *testing.T) {
	cfg := &Config{}
	if got := cfg.EffectiveBackend(); got != BackendLocal {
		t.Errorf("want %s, got %s", BackendLocal, got)
	}
}

func TestEffectiveLocalDir_Default(t *testing.T) {
	cfg := &Config{}
	if got := cfg.EffectiveLocalDir(); got != "images" {
		t.Errorf("want 'images', got %s", got)
	}
}

func TestEffectiveLocalDir_FromProject(t *testing.T) {
	cfg := &Config{
		Project: &ProjectConfig{Local: LocalConfig{Dir: "assets"}},
	}
	if got := cfg.EffectiveLocalDir(); got != "assets" {
		t.Errorf("want 'assets', got %s", got)
	}
}

func TestLoadProjectConfig_FindsFile(t *testing.T) {
	dir := t.TempDir()
	content := "backend: r2\nr2:\n  bucket: my-bucket\n  public_url: https://cdn.example.com\n"
	if err := os.WriteFile(filepath.Join(dir, ".mdp.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// Change to temp dir for loading.
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	cfg, err := loadProjectConfig()
	if err != nil {
		t.Fatalf("loadProjectConfig: %v", err)
	}
	if cfg.Backend != BackendR2 {
		t.Errorf("want r2, got %s", cfg.Backend)
	}
	if cfg.R2.Bucket != "my-bucket" {
		t.Errorf("want my-bucket, got %s", cfg.R2.Bucket)
	}
}
