package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Credentials represents stored OAuth tokens for the CLI session.
type Credentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	SessionID    string    `json:"session_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	Provider     string    `json:"provider"`
	BaseURL      string    `json:"base_url"`
}

// IsExpired reports whether the access token has expired (with 60s buffer).
func (c *Credentials) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false // no expiry info, assume valid
	}
	return time.Now().After(c.ExpiresAt.Add(-60 * time.Second))
}

// credentialsPath returns the full path to the credentials file.
func credentialsPath() (string, error) {
	dir, err := CredentialsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

// SaveCredentials persists credentials to disk with restricted permissions.
func SaveCredentials(creds *Credentials) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	// Write with owner-only permissions (0600)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing credentials: %w", err)
	}

	return nil
}

// LoadCredentials reads stored credentials from disk.
// Returns nil, nil if no credentials file exists.
func LoadCredentials() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}

	return &creds, nil
}

// ClearCredentials removes stored credentials from disk.
func ClearCredentials() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // already removed
		}
		return fmt.Errorf("removing credentials: %w", err)
	}

	return nil
}

// CredentialsFilePath returns the path to the credentials file for display purposes.
func CredentialsFilePath() string {
	p, err := credentialsPath()
	if err != nil {
		return "(unknown)"
	}
	return p
}
