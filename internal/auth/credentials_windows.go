package auth

import (
	"errors"
	"os"
	"path/filepath"
)

// CredentialsDir returns the Windows-specific directory for storing credentials.
// Windows: %APPDATA%/vectrade
func CredentialsDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", errors.New("APPDATA environment variable not set")
	}
	return filepath.Join(appData, "vectrade"), nil
}
