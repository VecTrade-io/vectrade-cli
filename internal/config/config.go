// Package config handles CLI configuration loading and resolution.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// EnvAPIKey is the environment variable for the API key.
	EnvAPIKey = "VECTRADE_API_KEY"

	// EnvBaseURL is the environment variable for a custom base URL.
	EnvBaseURL = "VECTRADE_BASE_URL"

	// EnvSandbox is the environment variable to enable sandbox mode.
	EnvSandbox = "VECTRADE_SANDBOX"

	// DefaultBaseURL is the production API base URL.
	DefaultBaseURL = "https://api.vectrade.io/v1"

	// SandboxBaseURL is the sandbox API base URL.
	// Sandbox uses the same domain — mode is determined by API key prefix (vq_test_ vs vq_live_).
	SandboxBaseURL = DefaultBaseURL

	// DefaultTimeout is the default request timeout in seconds.
	DefaultTimeout = 30
)

// Config represents the resolved CLI configuration.
type Config struct {
	APIKey   string `yaml:"api_key"`
	BaseURL  string `yaml:"base_url"`
	Sandbox  bool   `yaml:"sandbox"`
	Timeout  int    `yaml:"timeout"`
	Output   string `yaml:"output"` // "table", "json", "csv"
	JWTToken string `yaml:"-"`      // Populated from stored credentials, not config file
}

// ErrNoAPIKey is returned when no API key can be resolved.
var ErrNoAPIKey = errors.New("no API key configured. Run 'vectrade auth login' or set VECTRADE_API_KEY")

// Load resolves configuration from (in priority order):
// 1. Explicit flags (apiKey, sandbox)
// 2. Environment variables
// 3. Config file (~/.vectrade/config.yaml)
// 4. Defaults
func Load(flagAPIKey string, flagSandbox bool, configPath string) (*Config, error) {
	cfg := &Config{
		BaseURL: DefaultBaseURL,
		Timeout: DefaultTimeout,
		Output:  "table",
	}

	// Load config file if it exists
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	if data, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	// Environment variables override config file
	if envKey := os.Getenv(EnvAPIKey); envKey != "" {
		cfg.APIKey = envKey
	}
	if envURL := os.Getenv(EnvBaseURL); envURL != "" {
		cfg.BaseURL = envURL
	}
	if envSandbox := os.Getenv(EnvSandbox); isTruthy(envSandbox) {
		cfg.Sandbox = true
	}

	// Flags override everything
	if flagAPIKey != "" {
		cfg.APIKey = flagAPIKey
	}
	if flagSandbox {
		cfg.Sandbox = true
	}

	// Resolve sandbox URL
	if cfg.Sandbox && cfg.BaseURL == DefaultBaseURL {
		cfg.BaseURL = SandboxBaseURL
	}

	// Strip trailing slash
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	// If no API key resolved, try loading JWT from stored credentials
	if cfg.APIKey == "" {
		if jwt, err := loadStoredJWT(); err == nil && jwt != "" {
			cfg.JWTToken = jwt
		}
	}

	return cfg, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if c.APIKey == "" && c.JWTToken == "" {
		return ErrNoAPIKey
	}
	// Validate base URL scheme to prevent SSRF/downgrade attacks
	if !strings.HasPrefix(c.BaseURL, "https://") && !strings.HasPrefix(c.BaseURL, "http://localhost") && !strings.HasPrefix(c.BaseURL, "http://127.0.0.1") {
		return errors.New("base_url must use HTTPS (except for localhost development)")
	}
	return nil
}

// AuthHeader returns the appropriate Authorization header value.
// Prefers API key over JWT when both are present.
func (c *Config) AuthHeader() string {
	if c.APIKey != "" {
		return "Bearer " + c.APIKey
	}
	return "Bearer " + c.JWTToken
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".vectrade", "config.yaml")
}

func isTruthy(val string) bool {
	v := strings.ToLower(strings.TrimSpace(val))
	return v == "true" || v == "1" || v == "yes"
}

// loadStoredJWT reads the JWT access token from stored credentials.
// Returns empty string if no credentials are stored.
func loadStoredJWT() (string, error) {
	dir, err := credentialDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil // no credentials file — not an error
	}
	var creds struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", err
	}
	return creds.AccessToken, nil
}

// credentialDir returns the platform-specific directory for VecTrade credentials.
func credentialDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "vectrade"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", errors.New("APPDATA not set")
		}
		return filepath.Join(appData, "vectrade"), nil
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "vectrade"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "vectrade"), nil
	}
}
