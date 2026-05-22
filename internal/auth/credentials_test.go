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

func TestSaveCredentials_CreatesDirectoryAndFile(t *testing.T) {
	tmp := t.TempDir()
	// Override HOME so CredentialsDir resolves under tmp
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	creds := &Credentials{
		AccessToken:  "at_save_test",
		RefreshToken: "rt_save_test",
		SessionID:    "sess_save",
		ExpiresAt:    time.Now().Add(1 * time.Hour).Truncate(time.Second),
		Provider:     "google",
		BaseURL:      "https://api.vectrade.io/v1",
	}

	if err := SaveCredentials(creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	// Verify file exists
	path, _ := credentialsPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("credentials file not created: %v", err)
	}
	// Verify permissions are 0600
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("permissions = %o, want 0600", perm)
	}

	// Verify content
	data, _ := os.ReadFile(path)
	var loaded Credentials
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loaded.AccessToken != "at_save_test" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "at_save_test")
	}
}

func TestLoadCredentials_ReturnsNilWhenNoFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if creds != nil {
		t.Errorf("expected nil credentials, got %+v", creds)
	}
}

func TestLoadCredentials_ReadsFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	original := &Credentials{
		AccessToken:  "at_load_test",
		RefreshToken: "rt_load_test",
		SessionID:    "sess_load",
		ExpiresAt:    time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		Provider:     "microsoft",
		BaseURL:      "https://api.vectrade.io/v1",
	}

	// Save first
	if err := SaveCredentials(original); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	// Load
	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected credentials, got nil")
	}
	if loaded.AccessToken != "at_load_test" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "at_load_test")
	}
	if loaded.Provider != "microsoft" {
		t.Errorf("Provider = %q, want %q", loaded.Provider, "microsoft")
	}
	if !loaded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt mismatch")
	}
}

func TestLoadCredentials_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	// Create credentials file with invalid JSON at the platform-specific path
	dir, _ := CredentialsDir()
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "credentials.json"), []byte("not valid json{"), 0600)

	_, err := LoadCredentials()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestClearCredentials_RemovesFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	// Save credentials first
	creds := &Credentials{AccessToken: "at_clear", Provider: "google", BaseURL: "https://api.vectrade.io/v1"}
	if err := SaveCredentials(creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	// Verify file exists
	path, _ := credentialsPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist: %v", err)
	}

	// Clear
	if err := ClearCredentials(); err != nil {
		t.Fatalf("ClearCredentials: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("credentials file should have been removed")
	}
}

func TestClearCredentials_NoFileIsOK(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	// Ensure dir exists but no creds file
	dir, _ := CredentialsDir()
	os.MkdirAll(dir, 0700)

	if err := ClearCredentials(); err != nil {
		t.Errorf("ClearCredentials with no file should succeed: %v", err)
	}
}

func TestCredentialsDir_ContainsVectrade(t *testing.T) {
	dir, err := CredentialsDir()
	if err != nil {
		t.Fatalf("CredentialsDir: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %q", dir)
	}
	if !filepath.IsAbs(dir) || dir == "" {
		t.Errorf("CredentialsDir() = %q, expected non-empty absolute path", dir)
	}
}

func TestCredentialsPath_EndsWithJSON(t *testing.T) {
	path, err := credentialsPath()
	if err != nil {
		t.Fatalf("credentialsPath: %v", err)
	}
	if filepath.Base(path) != "credentials.json" {
		t.Errorf("expected credentials.json, got %q", filepath.Base(path))
	}
}

func TestSaveAndLoadCredentials_FullCycle(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))

	original := &Credentials{
		AccessToken:  "cycle_at",
		RefreshToken: "cycle_rt",
		SessionID:    "cycle_sess",
		ExpiresAt:    time.Now().Add(2 * time.Hour).Truncate(time.Second),
		Provider:     "apple",
		BaseURL:      "https://sandbox.api.vectrade.io/v1",
	}

	// Save
	if err := SaveCredentials(original); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Load
	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Verify all fields
	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken mismatch")
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken mismatch")
	}
	if loaded.SessionID != original.SessionID {
		t.Errorf("SessionID mismatch")
	}
	if loaded.Provider != original.Provider {
		t.Errorf("Provider mismatch")
	}
	if loaded.BaseURL != original.BaseURL {
		t.Errorf("BaseURL mismatch")
	}

	// Clear
	if err := ClearCredentials(); err != nil {
		t.Fatalf("clear: %v", err)
	}

	// Verify gone
	afterClear, err := LoadCredentials()
	if err != nil {
		t.Fatalf("load after clear: %v", err)
	}
	if afterClear != nil {
		t.Error("expected nil after clear")
	}
}
