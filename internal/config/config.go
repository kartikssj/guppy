// Package config resolves guppy's on-disk configuration locations.
package config

import (
	"os"
	"path/filepath"
)

const appName = "guppy"

// Dir returns guppy's configuration directory:
// $XDG_CONFIG_HOME/guppy, or ~/.config/guppy when XDG_CONFIG_HOME is unset.
func Dir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return appName // fall back to a cwd-relative directory
	}
	return filepath.Join(home, ".config", appName)
}

// EnsureDir creates the config directory with owner-only permissions.
func EnsureDir() error {
	return os.MkdirAll(Dir(), 0o700)
}

// TokenPath is where the cached OAuth token lives.
func TokenPath() string {
	return filepath.Join(Dir(), "token.json")
}

// CredentialsPath is where an optional user-supplied OAuth client
// configuration (downloaded from Google Cloud Console) lives.
func CredentialsPath() string {
	return filepath.Join(Dir(), "credentials.json")
}
