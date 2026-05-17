//go:build !darwin && !windows

package auth

import (
	"fmt"
	"os"
	"path/filepath"
)

// CredentialsDir returns the XDG-compliant directory for storing credentials.
// Linux/FreeBSD: $XDG_CONFIG_HOME/vectrade (defaults to ~/.config/vectrade)
func CredentialsDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "vectrade"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".config", "vectrade"), nil
}
