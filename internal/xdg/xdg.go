// Package xdg provides XDG Base Directory helpers.
//
// On Unix and macOS, XDG_CONFIG_HOME / XDG_CACHE_HOME env vars are respected,
// falling back to ~/.config and ~/.cache. On Windows, the standard OS paths
// (%APPDATA%, %LOCALAPPDATA%) are used unchanged.
package xdg

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the base directory for user-specific configuration files.
//   - Windows: %APPDATA% (via os.UserConfigDir)
//   - Others:  $XDG_CONFIG_HOME, falling back to ~/.config
func ConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		return os.UserConfigDir()
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

// CacheDir returns the base directory for user-specific cache files.
//   - Windows: %LOCALAPPDATA% (via os.UserCacheDir)
//   - Others:  $XDG_CACHE_HOME, falling back to ~/.cache
func CacheDir() (string, error) {
	if runtime.GOOS == "windows" {
		return os.UserCacheDir()
	}
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache"), nil
}
