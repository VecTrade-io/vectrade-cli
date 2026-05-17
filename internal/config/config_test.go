package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/VecTrade-io/vectrade-cli/internal/auth"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear env vars
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvSandbox, "")

	cfg := Load("", false, "/nonexistent/path.yaml")
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

	cfg := Load("", false, "/nonexistent/path.yaml")
	if cfg.APIKey != "vq_test_from_env" {
		t.Errorf("expected env API key, got %s", cfg.APIKey)
	}
	if cfg.BaseURL != "https://custom.api.io/v1" {
		t.Errorf("expected custom URL, got %s", cfg.BaseURL)
	}
}

func TestLoad_FlagOverridesEnv(t *testing.T) {
	t.Setenv(EnvAPIKey, "vq_test_env_key")

	cfg := Load("vq_test_flag_key", false, "/nonexistent/path.yaml")
	if cfg.APIKey != "vq_test_flag_key" {
		t.Errorf("expected flag API key, got %s", cfg.APIKey)
	}
}

func TestLoad_SandboxFromEnv(t *testing.T) {
	t.Setenv(EnvSandbox, "true")
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvBaseURL, "")

	cfg := Load("", false, "/nonexistent/path.yaml")
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

	cfg := Load("", true, "/nonexistent/path.yaml")
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

	cfg := Load("", false, cfgPath)
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

	cfg := Load("", false, "/nonexistent/path.yaml")
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

	// Malformed config is silently ignored, env takes over
	cfg := Load("", false, cfgPath)
	if cfg.APIKey != "vq_test_key" {
		t.Errorf("expected env key, got %s", cfg.APIKey)
	}
}

func TestDefaultConfigPath_ReturnsVectradePath(t *testing.T) {
	path := defaultConfigPath()
	if path == "" {
		t.Fatal("defaultConfigPath() returned empty")
	}
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("expected config.yaml, got %s", filepath.Base(path))
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
	// Should contain .vectrade
	if dir := filepath.Dir(path); filepath.Base(dir) != ".vectrade" {
		t.Errorf("expected parent dir .vectrade, got %s", dir)
	}
}

func TestLoadStoredJWT_NoFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	jwt, err := loadStoredJWT()
	if err != nil {
		t.Fatalf("loadStoredJWT: %v", err)
	}
	if jwt != "" {
		t.Errorf("expected empty JWT, got %q", jwt)
	}
}

func TestLoadStoredJWT_ValidFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create credentials file at the platform-specific path
	credsDir, _ := auth.CredentialsDir()
	os.MkdirAll(credsDir, 0700)

	creds := map[string]any{
		"access_token":  "jwt_stored_token_123",
		"refresh_token": "rt_xxx",
		"session_id":    "sess",
	}
	data, _ := json.Marshal(creds)
	os.WriteFile(filepath.Join(credsDir, "credentials.json"), data, 0600)

	jwt, err := loadStoredJWT()
	if err != nil {
		t.Fatalf("loadStoredJWT: %v", err)
	}
	if jwt != "jwt_stored_token_123" {
		t.Errorf("expected jwt_stored_token_123, got %q", jwt)
	}
}

func TestLoadStoredJWT_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	credsDir, _ := auth.CredentialsDir()
	os.MkdirAll(credsDir, 0700)
	os.WriteFile(filepath.Join(credsDir, "credentials.json"), []byte("invalid{json"), 0600)

	_, err := loadStoredJWT()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoad_JWTFallbackWhenNoAPIKey(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvSandbox, "")

	// Create credentials file with JWT
	credsDir, _ := auth.CredentialsDir()
	os.MkdirAll(credsDir, 0700)
	creds := map[string]any{"access_token": "jwt_fallback_tok"}
	data, _ := json.Marshal(creds)
	os.WriteFile(filepath.Join(credsDir, "credentials.json"), data, 0600)

	cfg := Load("", false, "/nonexistent/path.yaml")
	if cfg.JWTToken != "jwt_fallback_tok" {
		t.Errorf("expected JWT fallback, got %q", cfg.JWTToken)
	}
}

func TestLoad_APIKeyPreventsJWTFallback(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv(EnvAPIKey, "vq_explicit_key")
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvSandbox, "")

	// Create credentials file with JWT
	credsDir, _ := auth.CredentialsDir()
	os.MkdirAll(credsDir, 0700)
	creds := map[string]any{"access_token": "jwt_should_not_load"}
	data, _ := json.Marshal(creds)
	os.WriteFile(filepath.Join(credsDir, "credentials.json"), data, 0600)

	cfg := Load("", false, "/nonexistent/path.yaml")
	if cfg.JWTToken != "" {
		t.Errorf("JWT should not be loaded when API key is present, got %q", cfg.JWTToken)
	}
}

func TestValidate_HTTPDowngrade(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{"https is ok", "https://api.vectrade.io/v1", false},
		{"http://localhost is ok", "http://localhost:8080", false},
		{"http://127.0.0.1 is ok", "http://127.0.0.1:3000", false},
		{"http non-local blocked", "http://api.vectrade.io/v1", true},
		{"http evil blocked", "http://evil.com/steal", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{APIKey: "vq_test", BaseURL: tt.baseURL}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_SandboxEnvValues(t *testing.T) {
	tests := []string{"true", "1", "yes", "TRUE", "YES"}
	for _, v := range tests {
		t.Run(v, func(t *testing.T) {
			t.Setenv(EnvSandbox, v)
			t.Setenv(EnvAPIKey, "")
			t.Setenv(EnvBaseURL, "")
			cfg := Load("", false, "/nonexistent/path.yaml")
			if !cfg.Sandbox {
				t.Errorf("expected sandbox=true for env value %q", v)
			}
		})
	}
}

func TestLoad_SandboxDoesNotOverrideCustomURL(t *testing.T) {
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvBaseURL, "https://custom.api.io/v1")
	t.Setenv(EnvSandbox, "")

	cfg := Load("", true, "/nonexistent/path.yaml")
	// Sandbox flag should not override a custom URL
	if cfg.BaseURL != "https://custom.api.io/v1" {
		t.Errorf("expected custom URL preserved, got %s", cfg.BaseURL)
	}
}
