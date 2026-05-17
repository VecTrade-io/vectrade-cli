package auth

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestRequestAuthorizeURL_EmptyURLResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(authorizeResponse{URL: ""})
	}))
	defer srv.Close()

	_, err := requestAuthorizeURL(context.Background(), srv.URL, "google", 12345)
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestRequestAuthorizeURL_BadJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := requestAuthorizeURL(context.Background(), srv.URL, "google", 12345)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestExchangeCode_ServerError500(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	_, err := exchangeCode(context.Background(), srv.URL, "code", "state", 12345)
	if err == nil {
		t.Error("expected error for 500")
	}
}

func TestExchangeCode_BadJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := exchangeCode(context.Background(), srv.URL, "code", "state", 12345)
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestRefreshAccessToken_SavesCredentials(t *testing.T) {
	// Not parallel — uses HOME env
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"access":  "refreshed_token",
			"refresh": "new_refresh",
		})
	}))
	defer srv.Close()

	creds := &Credentials{
		RefreshToken: "old_refresh",
		BaseURL:      srv.URL,
		Provider:     "google",
	}

	result, err := RefreshAccessToken(srv.URL, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken != "refreshed_token" {
		t.Errorf("AccessToken = %q, want 'refreshed_token'", result.AccessToken)
	}

	// Verify credentials were saved
	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected saved credentials after refresh")
	}
	if loaded.AccessToken != "refreshed_token" {
		t.Errorf("saved AccessToken = %q", loaded.AccessToken)
	}
}

func TestRefreshAccessToken_KeepsOldRefreshToken(t *testing.T) {
	// Not parallel — uses HOME env
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Server returns no new refresh token
		json.NewEncoder(w).Encode(map[string]any{
			"access": "new_access",
		})
	}))
	defer srv.Close()

	creds := &Credentials{
		RefreshToken: "keep_this",
		BaseURL:      srv.URL,
		Provider:     "microsoft",
	}

	result, err := RefreshAccessToken(srv.URL, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RefreshToken != "keep_this" {
		t.Errorf("RefreshToken = %q, should keep 'keep_this'", result.RefreshToken)
	}
}

func TestRefreshAccessToken_BadJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	creds := &Credentials{RefreshToken: "tok"}
	_, err := RefreshAccessToken(srv.URL, creds)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestRefreshAccessToken_UserAgent(t *testing.T) {
	// Not parallel — mutates CLIVersion
	original := CLIVersion
	defer func() { CLIVersion = original }()
	CLIVersion = "2.0.0"

	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "vectrade-cli/2.0.0" {
			t.Errorf("User-Agent = %q, want vectrade-cli/2.0.0", ua)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"access": "tok"})
	}))
	defer srv.Close()

	creds := &Credentials{RefreshToken: "rt"}
	_, _ = RefreshAccessToken(srv.URL, creds)
}

func TestSupportedProviders_Count(t *testing.T) {
	if len(SupportedProviders) < 4 {
		t.Errorf("expected at least 4 providers, got %d", len(SupportedProviders))
	}
}

func TestExchangeCode_ContentType(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(tokenResponse{Access: "a", Refresh: "r"})
	}))
	defer srv.Close()

	_, _ = exchangeCode(context.Background(), srv.URL, "c", "s", 1234)
}

func TestRequestAuthorizeURL_UserAgent(t *testing.T) {
	// Not parallel — reads CLIVersion
	original := CLIVersion
	defer func() { CLIVersion = original }()
	CLIVersion = "3.0.0"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "vectrade-cli/3.0.0" {
			t.Errorf("User-Agent = %q, want vectrade-cli/3.0.0", ua)
		}
		json.NewEncoder(w).Encode(authorizeResponse{URL: "https://example.com/auth"})
	}))
	defer srv.Close()

	_, _ = requestAuthorizeURL(context.Background(), srv.URL, "google", 12345)
}

func TestLogin_FullFlow(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Mock backend that serves authorize URL and exchanges code
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/user/oauth/google/authorize/cli":
			// Return a URL that will trigger the callback on the local server
			port := r.URL.Query().Get("port")
			json.NewEncoder(w).Encode(authorizeResponse{
				URL: fmt.Sprintf("http://127.0.0.1:%s/callback?code=test_code&state=test_state", port),
			})
		case r.URL.Path == "/user/oauth/callback/cli":
			json.NewEncoder(w).Encode(tokenResponse{
				Access:    "login_access_token",
				Refresh:   "login_refresh_token",
				SessionID: "sess_123",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer backend.Close()

	// Mock browser that actually hits the callback URL
	mockBrowser := func(url string) error {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}

	creds, err := Login(LoginOptions{
		Provider:      "google",
		BaseURL:       backend.URL,
		Timeout:       10 * time.Second,
		BrowserOpener: mockBrowser,
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if creds.AccessToken != "login_access_token" {
		t.Errorf("AccessToken = %q, want login_access_token", creds.AccessToken)
	}
	if creds.RefreshToken != "login_refresh_token" {
		t.Errorf("RefreshToken = %q", creds.RefreshToken)
	}
	if creds.SessionID != "sess_123" {
		t.Errorf("SessionID = %q", creds.SessionID)
	}
	if creds.Provider != "google" {
		t.Errorf("Provider = %q", creds.Provider)
	}

	// Verify credentials were saved
	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if loaded == nil || loaded.AccessToken != "login_access_token" {
		t.Error("credentials not saved properly")
	}
}

func TestLogin_BrowserFailure_FallsBack(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/user/oauth/google/authorize/cli":
			port := r.URL.Query().Get("port")
			json.NewEncoder(w).Encode(authorizeResponse{
				URL: fmt.Sprintf("http://127.0.0.1:%s/callback?code=c&state=s", port),
			})
		case r.URL.Path == "/user/oauth/callback/cli":
			json.NewEncoder(w).Encode(tokenResponse{Access: "tok", Refresh: "ref"})
		}
	}))
	defer backend.Close()

	browserCalled := false
	mockBrowser := func(url string) error {
		browserCalled = true
		// Simulate failure opening browser but still hit the callback
		go func() {
			http.Get(url)
		}()
		return fmt.Errorf("no browser available")
	}

	creds, err := Login(LoginOptions{
		Provider:      "google",
		BaseURL:       backend.URL,
		Timeout:       10 * time.Second,
		BrowserOpener: mockBrowser,
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if !browserCalled {
		t.Error("browser opener should have been called")
	}
	if creds.AccessToken != "tok" {
		t.Errorf("AccessToken = %q", creds.AccessToken)
	}
}

func TestLogin_CallbackError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/oauth/google/authorize/cli" {
			port := r.URL.Query().Get("port")
			json.NewEncoder(w).Encode(authorizeResponse{
				URL: fmt.Sprintf("http://127.0.0.1:%s/callback?error=access_denied", port),
			})
		}
	}))
	defer backend.Close()

	mockBrowser := func(url string) error {
		http.Get(url)
		return nil
	}

	_, err := Login(LoginOptions{
		Provider:      "google",
		BaseURL:       backend.URL,
		Timeout:       10 * time.Second,
		BrowserOpener: mockBrowser,
	})
	if err == nil {
		t.Error("expected error for access_denied callback")
	}
}

func TestLogin_DefaultTimeout(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/oauth/google/authorize/cli" {
			port := r.URL.Query().Get("port")
			json.NewEncoder(w).Encode(authorizeResponse{
				URL: fmt.Sprintf("http://127.0.0.1:%s/callback?code=c&state=s", port),
			})
		} else if r.URL.Path == "/user/oauth/callback/cli" {
			json.NewEncoder(w).Encode(tokenResponse{Access: "a", Refresh: "r"})
		}
	}))
	defer backend.Close()

	mockBrowser := func(url string) error {
		http.Get(url)
		return nil
	}

	// Timeout = 0 should default to 5 minutes
	creds, err := Login(LoginOptions{
		Provider:      "google",
		BaseURL:       backend.URL,
		BrowserOpener: mockBrowser,
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if creds == nil {
		t.Error("expected credentials")
	}
}

func TestRequestAuthorizeURL_Non200(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("access denied"))
	}))
	defer srv.Close()

	_, err := requestAuthorizeURL(context.Background(), srv.URL, "google", 12345)
	if err == nil {
		t.Error("expected error for 403")
	}
}

func TestRequestAuthorizeURL_NetworkError(t *testing.T) {
	t.Parallel()
	_, err := requestAuthorizeURL(context.Background(), "http://127.0.0.1:1", "google", 12345)
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestExchangeCode_NetworkError(t *testing.T) {
	t.Parallel()
	_, err := exchangeCode(context.Background(), "http://127.0.0.1:1", "code", "state", 12345)
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestRefreshAccessToken_NetworkError(t *testing.T) {
	t.Parallel()
	creds := &Credentials{RefreshToken: "tok"}
	_, err := RefreshAccessToken("http://127.0.0.1:1", creds)
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestRequestAuthorizeURL_ContextCanceled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := requestAuthorizeURL(ctx, srv.URL, "google", 12345)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestExchangeCode_ContextCanceled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := exchangeCode(ctx, srv.URL, "code", "state", 12345)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestLogin_AuthorizeURLError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer backend.Close()

	_, err := Login(LoginOptions{
		Provider:      "google",
		BaseURL:       backend.URL,
		Timeout:       5 * time.Second,
		BrowserOpener: func(url string) error { return nil },
	})
	if err == nil {
		t.Error("expected error when authorize URL fails")
	}
}

func TestLogin_TokenExchangeError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/oauth/google/authorize/cli" {
			port := r.URL.Query().Get("port")
			json.NewEncoder(w).Encode(authorizeResponse{
				URL: fmt.Sprintf("http://127.0.0.1:%s/callback?code=c&state=s", port),
			})
		} else if r.URL.Path == "/user/oauth/callback/cli" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("exchange failed"))
		}
	}))
	defer backend.Close()

	mockBrowser := func(url string) error {
		http.Get(url)
		return nil
	}

	_, err := Login(LoginOptions{
		Provider:      "google",
		BaseURL:       backend.URL,
		Timeout:       5 * time.Second,
		BrowserOpener: mockBrowser,
	})
	if err == nil {
		t.Error("expected error when token exchange fails")
	}
}
