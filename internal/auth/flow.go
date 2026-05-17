package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTP header/content-type constants to satisfy linter duplicate-literal checks.
const (
	hdrAccept      = "Accept"
	hdrContentType = "Content-Type"
	hdrUserAgent   = "User-Agent"
	mimeJSON       = "application/json"
	uaPrefix       = "vectrade-cli/"

	errContactAPI = "contacting API: %w"
	errReadResp   = "reading response: %w"
)

// SupportedProviders lists the OAuth providers the CLI supports.
var SupportedProviders = []string{"google", "microsoft", "apple", "x"}

// LoginOptions configures the browser-based OAuth login flow.
type LoginOptions struct {
	Provider string // OAuth provider name (google, microsoft, apple, x)
	BaseURL  string // API base URL (e.g. https://api.vectrade.io/v1)
	Timeout  time.Duration
}

// tokenResponse mirrors the backend's OAuthTokenResponse.
type tokenResponse struct {
	Access    string `json:"access"`
	Refresh   string `json:"refresh"`
	SessionID string `json:"session_id"`
}

// authorizeResponse mirrors the backend's OAuthAuthorizeResponse.
type authorizeResponse struct {
	URL string `json:"url"`
}

// Login performs the full browser-based OAuth login flow:
//  1. Starts a local HTTP server on a random port
//  2. Requests an authorization URL from the backend (CLI-specific endpoint)
//  3. Opens the user's browser to the OAuth provider
//  4. Waits for the callback with the authorization code
//  5. Exchanges the code for JWT tokens via the backend
//  6. Stores credentials locally
//
// Returns the stored credentials or an error.
func Login(opts LoginOptions) (*Credentials, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// 1. Start local callback server
	srv, err := newCallbackServer()
	if err != nil {
		return nil, fmt.Errorf("starting callback server: %w", err)
	}
	srv.Start()

	port := srv.Port()
	fmt.Printf("  ▸ Local server listening on http://127.0.0.1:%d/callback\n", port)

	// 2. Request authorization URL from backend
	authURL, err := requestAuthorizeURL(ctx, opts.BaseURL, opts.Provider, port)
	if err != nil {
		return nil, fmt.Errorf("requesting authorization URL: %w", err)
	}

	// 3. Open browser
	fmt.Println("  ▸ Opening browser for authentication...")
	if err := openBrowser(authURL); err != nil {
		// Fall back to manual URL
		fmt.Printf("\n  Could not open browser automatically.\n  Open this URL in your browser:\n\n    %s\n\n", authURL)
	}

	fmt.Println("  ▸ Waiting for authentication callback...")

	// 4. Wait for callback
	result, err := srv.WaitForCallback(ctx)
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("authentication failed: %s", result.Error)
	}

	// 5. Exchange code for tokens
	fmt.Println("  ▸ Exchanging authorization code for tokens...")
	tokens, err := exchangeCode(ctx, opts.BaseURL, result.Code, result.State, port)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// 6. Store credentials
	creds := &Credentials{
		AccessToken:  tokens.Access,
		RefreshToken: tokens.Refresh,
		SessionID:    tokens.SessionID,
		ExpiresAt:    time.Now().Add(24 * time.Hour), // Default; backend may provide exact TTL
		Provider:     opts.Provider,
		BaseURL:      opts.BaseURL,
	}

	if err := SaveCredentials(creds); err != nil {
		return nil, fmt.Errorf("storing credentials: %w", err)
	}

	return creds, nil
}

// requestAuthorizeURL calls the backend's CLI-specific authorize endpoint.
func requestAuthorizeURL(ctx context.Context, baseURL, provider string, port int) (string, error) {
	url := fmt.Sprintf("%s/user/oauth/%s/authorize/cli?port=%d", baseURL, provider, port)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set(hdrAccept, mimeJSON)
	req.Header.Set(hdrUserAgent, uaPrefix+cliVersion())

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf(errContactAPI, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf(errReadResp, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var authResp authorizeResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if authResp.URL == "" {
		return "", fmt.Errorf("empty authorization URL returned")
	}

	return authResp.URL, nil
}

// exchangeCode calls the backend's CLI-specific callback endpoint to exchange
// the authorization code for JWT tokens.
func exchangeCode(ctx context.Context, baseURL, code, state string, port int) (*tokenResponse, error) {
	url := baseURL + "/user/oauth/callback/cli"

	payload := fmt.Sprintf(`{"code":%q,"state":%q,"port":%d}`, code, state, port)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(hdrContentType, mimeJSON)
	req.Header.Set(hdrAccept, mimeJSON)
	req.Header.Set(hdrUserAgent, uaPrefix+cliVersion())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errContactAPI, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(errReadResp, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange returned %d: %s", resp.StatusCode, string(body))
	}

	var tokens tokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("parsing tokens: %w", err)
	}

	return &tokens, nil
}

// RefreshAccessToken attempts to refresh an expired access token using the refresh token.
func RefreshAccessToken(baseURL string, creds *Credentials) (*Credentials, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	url := baseURL + "/user/auth/refresh"
	payload := fmt.Sprintf(`{"refresh_token":%q}`, creds.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(hdrContentType, mimeJSON)
	req.Header.Set(hdrAccept, mimeJSON)
	req.Header.Set(hdrUserAgent, uaPrefix+cliVersion())

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errContactAPI, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(errReadResp, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed (%d): %s — run 'vectrade auth login' to re-authenticate", resp.StatusCode, string(body))
	}

	var tokens struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
	}
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}

	creds.AccessToken = tokens.Access
	if tokens.Refresh != "" {
		creds.RefreshToken = tokens.Refresh
	}
	creds.ExpiresAt = time.Now().Add(24 * time.Hour)

	if err := SaveCredentials(creds); err != nil {
		return nil, fmt.Errorf("storing refreshed credentials: %w", err)
	}

	return creds, nil
}

// CLIVersion is set by the cmd package at init time to provide the version for User-Agent headers.
var CLIVersion = "dev"

// cliVersion returns the CLI version for User-Agent headers.
func cliVersion() string {
	return CLIVersion
}
