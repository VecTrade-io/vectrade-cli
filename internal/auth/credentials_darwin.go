package auth

import (
	"fmt"
	"os"
	"path/filepath"
)

// CredentialsDir returns the macOS-specific directory for storing credentials.
// macOS: ~/Library/Application Support/vectrade
func CredentialsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, "Library", "Application Support", "vectrade"), nil
}
