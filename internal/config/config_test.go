package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear env vars
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvSandbox, "")

	cfg, err := Load("", false, "/nonexistent/path.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("expected default base URL, got %s", cfg.BaseURL)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Errorf("expected timeout %d, got %d", DefaultTimeout, cfg.Timeout)
	}
	if cfg.Output != "table" {
		t.Errorf("expected output 'table', got %s", cfg.Output)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv(EnvAPIKey, "vq_test_from_env")
	t.Setenv(EnvBaseURL, "https://custom.api.io/v1")

	cfg, err := Load("", false, "/nonexistent/path.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "vq_test_from_env" {
		t.Errorf("expected env API key, got %s", cfg.APIKey)
	}
	if cfg.BaseURL != "https://custom.api.io/v1" {
		t.Errorf("expected custom URL, got %s", cfg.BaseURL)
	}
}

func TestLoad_FlagOverridesEnv(t *testing.T) {
	t.Setenv(EnvAPIKey, "vq_test_env_key")

	cfg, err := Load("vq_test_flag_key", false, "/nonexistent/path.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "vq_test_flag_key" {
		t.Errorf("expected flag API key, got %s", cfg.APIKey)
	}
}

func TestLoad_SandboxFromEnv(t *testing.T) {
	t.Setenv(EnvSandbox, "true")
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvBaseURL, "")

	cfg, err := Load("", false, "/nonexistent/path.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Sandbox {
		t.Error("expected sandbox=true from env")
	}
	if cfg.BaseURL != SandboxBaseURL {
		t.Errorf("expected sandbox URL, got %s", cfg.BaseURL)
	}
}

func TestLoad_SandboxFromFlag(t *testing.T) {
	t.Setenv(EnvSandbox, "")
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvAPIKey, "")

	cfg, err := Load("", true, "/nonexistent/path.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != SandboxBaseURL {
		t.Errorf("expected sandbox URL, got %s", cfg.BaseURL)
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := []byte("api_key: vq_test_from_file\ntimeout: 60\noutput: json\n")
	if err := os.WriteFile(cfgPath, content, 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvSandbox, "")

	cfg, err := Load("", false, cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "vq_test_from_file" {
		t.Errorf("expected file API key, got %s", cfg.APIKey)
	}
	if cfg.Timeout != 60 {
		t.Errorf("expected timeout 60, got %d", cfg.Timeout)
	}
	if cfg.Output != "json" {
		t.Errorf("expected output 'json', got %s", cfg.Output)
	}
}

func TestValidate_NoKey(t *testing.T) {
	cfg := &Config{BaseURL: DefaultBaseURL}
	if err := cfg.Validate(); err != ErrNoAPIKey {
		t.Errorf("expected ErrNoAPIKey, got %v", err)
	}
}

func TestValidate_WithKey(t *testing.T) {
	cfg := &Config{APIKey: "vq_test_key", BaseURL: DefaultBaseURL}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoad_TrailingSlashStripped(t *testing.T) {
	t.Setenv(EnvBaseURL, "https://api.example.io/v1/")
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvSandbox, "")

	cfg, err := Load("", false, "/nonexistent/path.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "https://api.example.io/v1" {
		t.Errorf("expected trailing slash stripped, got %s", cfg.BaseURL)
	}
}

func TestValidate_HTTPSRequired(t *testing.T) {
	cfg := &Config{APIKey: "vq_test_key", BaseURL: "http://evil.com"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for non-HTTPS URL")
	}
}

func TestValidate_LocalhostHTTPAllowed(t *testing.T) {
	tests := []string{
		"http://localhost:8080",
		"http://127.0.0.1:3000",
	}
	for _, url := range tests {
		cfg := &Config{APIKey: "vq_test_key", BaseURL: url}
		if err := cfg.Validate(); err != nil {
			t.Errorf("localhost URL %q should be allowed, got: %v", url, err)
		}
	}
}

func TestValidate_JWTTokenSufficient(t *testing.T) {
	cfg := &Config{JWTToken: "jwt_token", BaseURL: DefaultBaseURL}
	if err := cfg.Validate(); err != nil {
		t.Errorf("JWT should be sufficient for validation: %v", err)
	}
}

func TestAuthHeader_PrefersAPIKey(t *testing.T) {
	cfg := &Config{APIKey: "vq_key", JWTToken: "jwt_tok"}
	if got := cfg.AuthHeader(); got != "Bearer vq_key" {
		t.Errorf("expected 'Bearer vq_key', got %q", got)
	}
}

func TestAuthHeader_FallsBackToJWT(t *testing.T) {
	cfg := &Config{JWTToken: "jwt_tok"}
	if got := cfg.AuthHeader(); got != "Bearer jwt_tok" {
		t.Errorf("expected 'Bearer jwt_tok', got %q", got)
	}
}

func TestIsTruthy(t *testing.T) {
	for _, v := range []string{"true", "TRUE", "1", "yes", "YES", " true "} {
		if !isTruthy(v) {
			t.Errorf("isTruthy(%q) should be true", v)
		}
	}
	for _, v := range []string{"false", "0", "no", "", "maybe"} {
		if isTruthy(v) {
			t.Errorf("isTruthy(%q) should be false", v)
		}
	}
}

func TestLoad_MalformedConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("{{{{invalid yaml"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Setenv(EnvAPIKey, "vq_test_key")
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvSandbox, "")

	// Should not error — malformed config is silently ignored, env takes over
	cfg, err := Load("", false, cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "vq_test_key" {
		t.Errorf("expected env key, got %s", cfg.APIKey)
	}
}
