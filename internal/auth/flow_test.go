package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestAuthorizeURL_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/oauth/google/authorize/cli" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("port") != "12345" {
			t.Errorf("unexpected port param: %s", r.URL.Query().Get("port"))
		}
		json.NewEncoder(w).Encode(authorizeResponse{URL: "https://accounts.google.com/o/oauth2/auth?..."})
	}))
	defer srv.Close()

	ctx := context.Background()
	url, err := requestAuthorizeURL(ctx, srv.URL, "google", 12345)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}
}

func TestRequestAuthorizeURL_APIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid provider"}`))
	}))
	defer srv.Close()

	ctx := context.Background()
	_, err := requestAuthorizeURL(ctx, srv.URL, "invalid", 12345)
	if err == nil {
		t.Error("expected error for bad request")
	}
}

func TestRequestAuthorizeURL_EmptyURL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(authorizeResponse{URL: ""})
	}))
	defer srv.Close()

	ctx := context.Background()
	_, err := requestAuthorizeURL(ctx, srv.URL, "google", 12345)
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestExchangeCode_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/oauth/callback/cli" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(tokenResponse{
			Access:    "access_tok",
			Refresh:   "refresh_tok",
			SessionID: "sess_123",
		})
	}))
	defer srv.Close()

	ctx := context.Background()
	tokens, err := exchangeCode(ctx, srv.URL, "code123", "state456", 9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.Access != "access_tok" {
		t.Errorf("expected access_tok, got %s", tokens.Access)
	}
	if tokens.SessionID != "sess_123" {
		t.Errorf("expected sess_123, got %s", tokens.SessionID)
	}
}

func TestExchangeCode_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	ctx := context.Background()
	_, err := exchangeCode(ctx, srv.URL, "code", "state", 9999)
	if err == nil {
		t.Error("expected error for server error")
	}
}

func TestRefreshAccessToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/auth/refresh" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]string{
			"access":  "new_access",
			"refresh": "new_refresh",
		})
	}))
	defer srv.Close()

	// Use a temp dir for credentials
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	creds := &Credentials{
		AccessToken:  "old_access",
		RefreshToken: "old_refresh",
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // expired
		BaseURL:      srv.URL,
	}

	result, err := RefreshAccessToken(srv.URL, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken != "new_access" {
		t.Errorf("expected new_access, got %s", result.AccessToken)
	}
	if result.RefreshToken != "new_refresh" {
		t.Errorf("expected new_refresh, got %s", result.RefreshToken)
	}
}

func TestRefreshAccessToken_Failure(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("token expired"))
	}))
	defer srv.Close()

	creds := &Credentials{RefreshToken: "bad_token"}
	_, err := RefreshAccessToken(srv.URL, creds)
	if err == nil {
		t.Error("expected error for failed refresh")
	}
}

func TestCLIVersion_Default(t *testing.T) {
	// Default value should be "dev"
	original := CLIVersion
	defer func() { CLIVersion = original }()

	CLIVersion = "dev"
	if got := cliVersion(); got != "dev" {
		t.Errorf("cliVersion() = %q, want %q", got, "dev")
	}
}

func TestCLIVersion_Set(t *testing.T) {
	original := CLIVersion
	defer func() { CLIVersion = original }()

	CLIVersion = "1.2.3"
	if got := cliVersion(); got != "1.2.3" {
		t.Errorf("cliVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestExchangeCode_UserAgent(t *testing.T) {
	// Not parallel — mutates global CLIVersion

	original := CLIVersion
	defer func() { CLIVersion = original }()
	CLIVersion = "0.5.0"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "vectrade-cli/0.5.0" {
			t.Errorf("User-Agent = %q, want %q", ua, "vectrade-cli/0.5.0")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(tokenResponse{
			Access:  "tok",
			Refresh: "ref",
		})
	}))
	defer srv.Close()

	_, err := exchangeCode(context.Background(), srv.URL, "code", "state", 12345)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
