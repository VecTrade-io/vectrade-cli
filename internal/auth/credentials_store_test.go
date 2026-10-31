package auth

import (
	"os"
	"path/filepath"
	"testing"
)

// isolateCredentials points the credentials directory at a fresh temp dir on every platform.
func isolateCredentials(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", home)
	dir, err := CredentialsDir()
	if err != nil {
		t.Fatalf("CredentialsDir: %v", err)
	}
	return dir
}

func TestCredentialsStore_RoundTripAndClear(t *testing.T) {
	dir := isolateCredentials(t)

	if creds, err := LoadCredentials(); err != nil || creds != nil {
		t.Fatalf("expected no credentials yet, got %v, %v", creds, err)
	}
	if err := SaveCredentials(&Credentials{AccessToken: "at", Provider: "google"}); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}
	if got := CredentialsFilePath(); got != filepath.Join(dir, "credentials.json") {
		t.Errorf("CredentialsFilePath = %q", got)
	}
	info, err := os.Stat(filepath.Join(dir, "credentials.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 && os.PathSeparator == '/' {
		t.Errorf("credentials must be owner-only, got %o", perm)
	}

	creds, err := LoadCredentials()
	if err != nil || creds == nil || creds.AccessToken != "at" {
		t.Fatalf("LoadCredentials = %v, %v", creds, err)
	}
	if err := ClearCredentials(); err != nil {
		t.Fatalf("ClearCredentials: %v", err)
	}
	if err := ClearCredentials(); err != nil {
		t.Fatalf("clearing twice should be a no-op: %v", err)
	}
}

func TestLoadCredentials_RejectsCorruptFile(t *testing.T) {
	dir := isolateCredentials(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "credentials.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCredentials(); err == nil {
		t.Fatal("expected a parse error")
	}
}

func TestCredentialsStore_ReportsFilesystemErrors(t *testing.T) {
	dir := isolateCredentials(t)
	// A directory where the credentials file should be makes read, write and remove fail.
	if err := os.MkdirAll(filepath.Join(dir, "credentials.json", "child"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCredentials(); err == nil {
		t.Error("expected LoadCredentials to fail")
	}
	if err := SaveCredentials(&Credentials{AccessToken: "at"}); err == nil {
		t.Error("expected SaveCredentials to fail")
	}
	if err := ClearCredentials(); err == nil {
		t.Error("expected ClearCredentials to fail")
	}
}

func TestSaveCredentials_FailsWhenConfigDirIsAFile(t *testing.T) {
	dir := isolateCredentials(t)
	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveCredentials(&Credentials{AccessToken: "at"}); err == nil {
		t.Fatal("expected SaveCredentials to fail")
	}
}
