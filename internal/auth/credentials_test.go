package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCredentials_SaveLoadCycle(t *testing.T) {
	// Use a temp dir to avoid touching real credentials
	tmp := t.TempDir()
	path := filepath.Join(tmp, "credentials.json")

	creds := &Credentials{
		AccessToken:  "at_test_123",
		RefreshToken: "rt_test_456",
		SessionID:    "sess_789",
		ExpiresAt:    time.Now().Add(24 * time.Hour).Truncate(time.Second),
		Provider:     "google",
		BaseURL:      "https://api.vectrade.io/v1",
	}

	// Write
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.MkdirAll(tmp, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read back
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var loaded Credentials
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.AccessToken != creds.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, creds.AccessToken)
	}
	if loaded.RefreshToken != creds.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, creds.RefreshToken)
	}
	if loaded.SessionID != creds.SessionID {
		t.Errorf("SessionID = %q, want %q", loaded.SessionID, creds.SessionID)
	}
	if loaded.Provider != creds.Provider {
		t.Errorf("Provider = %q, want %q", loaded.Provider, creds.Provider)
	}
	if loaded.BaseURL != creds.BaseURL {
		t.Errorf("BaseURL = %q, want %q", loaded.BaseURL, creds.BaseURL)
	}
	if !loaded.ExpiresAt.Equal(creds.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", loaded.ExpiresAt, creds.ExpiresAt)
	}
}

func TestCredentials_FilePermissions(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "credentials.json")

	data := []byte(`{"access_token":"test"}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("permissions = %o, want 0600", perm)
	}
}

func TestIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "zero value (no expiry) -> not expired",
			expiresAt: time.Time{},
			want:      false,
		},
		{
			name:      "far future -> not expired",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "already past -> expired",
			expiresAt: time.Now().Add(-1 * time.Minute),
			want:      true,
		},
		{
			name:      "within 60s buffer -> expired",
			expiresAt: time.Now().Add(30 * time.Second),
			want:      true,
		},
		{
			name:      "just outside 60s buffer -> not expired",
			expiresAt: time.Now().Add(90 * time.Second),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Credentials{ExpiresAt: tt.expiresAt}
			if got := c.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v (expiresAt=%v, now=%v)",
					got, tt.want, tt.expiresAt, time.Now())
			}
		})
	}
}

func TestClearCredentials_NonExistent(t *testing.T) {
	// Removing a file that does not exist should not error
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nope.json")

	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCredentialsFilePath_NotEmpty(t *testing.T) {
	p := CredentialsFilePath()
	if p == "" {
		t.Error("CredentialsFilePath() returned empty string")
	}
	if p == "(unknown)" {
		t.Error("CredentialsFilePath() returned (unknown)")
	}
}

func TestCredentials_JSONRoundTrip(t *testing.T) {
	original := &Credentials{
		AccessToken:  "access_tok",
		RefreshToken: "refresh_tok",
		SessionID:    "session_123",
		ExpiresAt:    time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		Provider:     "microsoft",
		BaseURL:      "https://api.vectrade.io/v1",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Credentials
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken mismatch")
	}
	if decoded.Provider != original.Provider {
		t.Errorf("Provider mismatch")
	}
	if !decoded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt mismatch: got %v, want %v", decoded.ExpiresAt, original.ExpiresAt)
	}
}
