package config

import (
	"os"
	"path/filepath"

	"github.com/daaa1k/mdp/internal/xdg"
	"gopkg.in/yaml.v3"
)

// BackendType represents the storage backend choice.
type BackendType string

const (
	BackendLocal  BackendType = "local"
	BackendR2     BackendType = "r2"
	BackendNodeBB BackendType = "nodebb"
)

// R2Config holds Cloudflare R2 settings.
type R2Config struct {
	Bucket    string `yaml:"bucket"`
	PublicURL string `yaml:"public_url"`
	Endpoint  string `yaml:"endpoint"`
	Prefix    string `yaml:"prefix"`
}

// NodeBBConfig holds NodeBB settings.
type NodeBBConfig struct {
	URL string `yaml:"url"`
}

// LocalConfig holds local backend settings.
type LocalConfig struct {
	Dir string `yaml:"dir"`
}

// ProjectConfig is loaded from .mdp.yaml in the project or its parents.
type ProjectConfig struct {
	Backend BackendType  `yaml:"backend"`
	Local   LocalConfig  `yaml:"local"`
	R2      R2Config     `yaml:"r2"`
	NodeBB  NodeBBConfig `yaml:"nodebb"`
}

// GlobalConfig is loaded from the user's config directory.
type GlobalConfig struct {
	Backend BackendType  `yaml:"backend"`
	Local   LocalConfig  `yaml:"local"`
	R2      R2Config     `yaml:"r2"`
	NodeBB  NodeBBConfig `yaml:"nodebb"`

	// WSL2: optional path to powershell.exe
	PowerShellPath string `yaml:"powershell_path"`
}

// Config is the merged configuration.
type Config struct {
	Project *ProjectConfig
	Global  *GlobalConfig

	// CLIBackend overrides all config file settings when non-empty.
	CLIBackend BackendType
}

// EffectiveBackend resolves the backend priority: CLI > project > global > local.
func (c *Config) EffectiveBackend() BackendType {
	if c.CLIBackend != "" {
		return c.CLIBackend
	}
	if c.Project != nil && c.Project.Backend != "" {
		return c.Project.Backend
	}
	if c.Global != nil && c.Global.Backend != "" {
		return c.Global.Backend
	}
	return BackendLocal
}

// EffectiveLocalDir returns the local directory, defaulting to "images".
func (c *Config) EffectiveLocalDir() string {
	if c.Project != nil && c.Project.Local.Dir != "" {
		return c.Project.Local.Dir
	}
	if c.Global != nil && c.Global.Local.Dir != "" {
		return c.Global.Local.Dir
	}
	return "images"
}

// EffectiveR2 returns the merged R2 config (project overrides global).
func (c *Config) EffectiveR2() R2Config {
	var r2 R2Config
	if c.Global != nil {
		r2 = c.Global.R2
	}
	if c.Project != nil {
		if c.Project.R2.Bucket != "" {
			r2.Bucket = c.Project.R2.Bucket
		}
		if c.Project.R2.PublicURL != "" {
			r2.PublicURL = c.Project.R2.PublicURL
		}
		if c.Project.R2.Endpoint != "" {
			r2.Endpoint = c.Project.R2.Endpoint
		}
		if c.Project.R2.Prefix != "" {
			r2.Prefix = c.Project.R2.Prefix
		}
	}
	return r2
}

// EffectiveNodeBB returns the merged NodeBB config.
func (c *Config) EffectiveNodeBB() NodeBBConfig {
	var nb NodeBBConfig
	if c.Global != nil {
		nb = c.Global.NodeBB
	}
	if c.Project != nil && c.Project.NodeBB.URL != "" {
		nb.URL = c.Project.NodeBB.URL
	}
	return nb
}

// EffectivePowerShellPath returns the configured powershell path or empty string.
func (c *Config) EffectivePowerShellPath() string {
	if c.Global != nil {
		return c.Global.PowerShellPath
	}
	return ""
}

// Load reads project and global configs from disk.
func Load() (*Config, error) {
	cfg := &Config{}

	if proj, err := loadProjectConfig(); err == nil {
		cfg.Project = proj
	}
	if global, err := loadGlobalConfig(); err == nil {
		cfg.Global = global
	}

	return cfg, nil
}

// loadProjectConfig walks up from CWD looking for .mdp.yaml.
func loadProjectConfig() (*ProjectConfig, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		path := filepath.Join(dir, ".mdp.yaml")
		if data, err := os.ReadFile(path); err == nil {
			var cfg ProjectConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, err
			}
			return &cfg, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, os.ErrNotExist
}

// loadGlobalConfig reads the global config file from the XDG config directory.
func loadGlobalConfig() (*GlobalConfig, error) {
	dir, err := xdg.ConfigDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "mdp", "config.yaml"))
	if err != nil {
		return nil, err
	}
	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
