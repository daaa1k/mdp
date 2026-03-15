package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
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
	Bucket    string `toml:"bucket"`
	PublicURL string `toml:"public_url"`
	Endpoint  string `toml:"endpoint"`
	Prefix    string `toml:"prefix"`
}

// NodeBBConfig holds NodeBB settings.
type NodeBBConfig struct {
	URL string `toml:"url"`
}

// LocalConfig holds local backend settings.
type LocalConfig struct {
	Dir string `toml:"dir"`
}

// ProjectConfig is loaded from .mdp.toml in the project or its parents.
type ProjectConfig struct {
	Backend BackendType  `toml:"backend"`
	Local   LocalConfig  `toml:"local"`
	R2      R2Config     `toml:"r2"`
	NodeBB  NodeBBConfig `toml:"nodebb"`
}

// GlobalConfig is loaded from the user's config directory.
type GlobalConfig struct {
	Backend BackendType  `toml:"backend"`
	Local   LocalConfig  `toml:"local"`
	R2      R2Config     `toml:"r2"`
	NodeBB  NodeBBConfig `toml:"nodebb"`

	// WSL2: optional path to powershell.exe
	PowerShellPath string `toml:"powershell_path"`
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

// loadProjectConfig walks up from CWD looking for .mdp.toml.
func loadProjectConfig() (*ProjectConfig, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		path := filepath.Join(dir, ".mdp.toml")
		if _, err := os.Stat(path); err == nil {
			var cfg ProjectConfig
			if _, err := toml.DecodeFile(path, &cfg); err != nil {
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

// globalConfigPath returns the platform-specific global config path.
func globalConfigPath() string {
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "mdp", "config.toml")
		}
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "mdp", "config.toml")
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".config", "mdp", "config.toml")
	}
	return ""
}

// loadGlobalConfig reads the global config file.
func loadGlobalConfig() (*GlobalConfig, error) {
	path := globalConfigPath()
	if path == "" {
		return nil, os.ErrNotExist
	}
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	var cfg GlobalConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
