// Package config provides types and I/O for gh-identity's YAML configuration
// files (profiles.yml, bindings.yml).
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultConfigDir is the subdirectory under XDG_CONFIG_HOME / ~/.config.
	DefaultConfigDir = "gh-identity"
)

// Dir returns the configuration directory for gh-identity.
// It respects GH_IDENTITY_CONFIG_DIR, then XDG_CONFIG_HOME, then ~/.config.
func Dir() (string, error) {
	if d := os.Getenv("GH_IDENTITY_CONFIG_DIR"); d != "" {
		return d, nil
	}

	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, DefaultConfigDir), nil
}

// EnsureDir creates the config directory (and parents) if it does not exist.
func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}
	return dir, nil
}

// GitConfigDir returns the directory where per-profile gitconfig fragments live.
func GitConfigDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "git"), nil
}

// EnsureGitConfigDir creates the git config fragment directory if needed.
func EnsureGitConfigDir() (string, error) {
	dir, err := GitConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating git config directory: %w", err)
	}
	return dir, nil
}

// BinDir returns the directory where the hook binary is installed.
func BinDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bin"), nil
}
