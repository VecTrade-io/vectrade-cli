package auth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCallbackServer_StartsAndListens(t *testing.T) {
	srv, err := newCallbackServer()
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	srv.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.server.Shutdown(ctx)
	}()

	port := srv.Port()
	if port < 1024 || port > 65535 {
		t.Errorf("Port() = %d, want ephemeral port 1024-65535", port)
	}
}

func TestCallbackServer_SuccessfulCallback(t *testing.T) {
	srv, err := newCallbackServer()
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	srv.Start()

	port := srv.Port()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simulate the OAuth provider redirecting to callback
	callbackURL := "http://127.0.0.1:" + itoa(port) + "/callback?code=test_code_123&state=test_state"
	resp, err := http.Get(callbackURL) //nolint:gosec // test-only localhost
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("callback status = %d, want 200", resp.StatusCode)
	}

	// Verify HTML success page returned
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("empty response body")
	}

	// WaitForCallback should return immediately with the code
	result, err := srv.WaitForCallback(ctx)
	if err != nil {
		t.Fatalf("WaitForCallback: %v", err)
	}
	if result.Code != "test_code_123" {
		t.Errorf("Code = %q, want %q", result.Code, "test_code_123")
	}
	if result.State != "test_state" {
		t.Errorf("State = %q, want %q", result.State, "test_state")
	}
	if result.Error != "" {
		t.Errorf("Error = %q, want empty", result.Error)
	}
}

func TestCallbackServer_ErrorCallback(t *testing.T) {
	srv, err := newCallbackServer()
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	srv.Start()

	port := srv.Port()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	callbackURL := "http://127.0.0.1:" + itoa(port) + "/callback?error=access_denied&error_description=User+denied+access"
	resp, err := http.Get(callbackURL) //nolint:gosec // test-only localhost
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	resp.Body.Close()

	result, err := srv.WaitForCallback(ctx)
	if err != nil {
		t.Fatalf("WaitForCallback: %v", err)
	}
	if result.Error != "User denied access" {
		t.Errorf("Error = %q, want %q", result.Error, "User denied access")
	}
}

func TestCallbackServer_MissingCode(t *testing.T) {
	srv, err := newCallbackServer()
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	srv.Start()

	port := srv.Port()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	callbackURL := "http://127.0.0.1:" + itoa(port) + "/callback?state=orphan_state"
	resp, err := http.Get(callbackURL) //nolint:gosec // test-only localhost
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	result, err := srv.WaitForCallback(ctx)
	if err != nil {
		t.Fatalf("WaitForCallback: %v", err)
	}
	if result.Error == "" {
		t.Error("expected non-empty error for missing code")
	}
}

func TestCallbackServer_Timeout(t *testing.T) {
	srv, err := newCallbackServer()
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	srv.Start()

	// Use a very short timeout so the test doesn't wait long
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = srv.WaitForCallback(ctx)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestSupportedProviders(t *testing.T) {
	expected := map[string]bool{
		"google":    false,
		"microsoft": false,
		"apple":     false,
		"x":         false,
	}

	for _, p := range SupportedProviders {
		if _, ok := expected[p]; !ok {
			t.Errorf("unexpected provider %q", p)
		}
		expected[p] = true
	}

	for p, found := range expected {
		if !found {
			t.Errorf("missing expected provider %q", p)
		}
	}
}

func TestCallbackServer_XSSPrevention(t *testing.T) {
	srv, err := newCallbackServer()
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	srv.Start()

	port := srv.Port()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Inject XSS payload via error_description
	xssPayload := "<script>alert('xss')</script>"
	callbackURL := "http://127.0.0.1:" + itoa(port) + "/callback?error=test&error_description=" + xssPayload
	resp, err := http.Get(callbackURL) //nolint:gosec // test-only localhost
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	htmlBody := string(body)

	// The raw <script> tag must NOT appear in the HTML response
	if strings.Contains(htmlBody, "<script>alert") {
		t.Error("XSS: raw <script> tag found in error page — error_description not escaped")
	}
	// The escaped version should be present
	if !strings.Contains(htmlBody, "&lt;script&gt;") {
		t.Error("expected HTML-escaped script tag in response")
	}

	// Drain the result channel
	_, _ = srv.WaitForCallback(ctx)
}

func TestCredentialsDir_ReturnsPath(t *testing.T) {
	dir, err := CredentialsDir()
	if err != nil {
		t.Fatalf("CredentialsDir(): %v", err)
	}
	if dir == "" {
		t.Error("CredentialsDir() returned empty string")
	}
	if !strings.Contains(dir, "vectrade") {
		t.Errorf("CredentialsDir() = %q, expected path containing 'vectrade'", dir)
	}
}

func TestCallbackServer_NonCallbackPath(t *testing.T) {
	srv, err := newCallbackServer()
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	srv.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.server.Shutdown(ctx)
	}()

	port := srv.Port()
	// Request a path that isn't /callback — should 404
	resp, err := http.Get("http://127.0.0.1:" + itoa(port) + "/other") //nolint:gosec
	if err != nil {
		t.Fatalf("GET /other: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for /other, got %d", resp.StatusCode)
	}
}

func TestCallbackServer_MultipleCallbacks(t *testing.T) {
	srv, err := newCallbackServer()
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	srv.Start()

	port := srv.Port()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First callback
	resp, err := http.Get("http://127.0.0.1:" + itoa(port) + "/callback?code=first&state=s1") //nolint:gosec
	if err != nil {
		t.Fatalf("first callback: %v", err)
	}
	resp.Body.Close()

	result, err := srv.WaitForCallback(ctx)
	if err != nil {
		t.Fatalf("WaitForCallback: %v", err)
	}
	if result.Code != "first" {
		t.Errorf("expected 'first', got %q", result.Code)
	}
}

func TestCallbackServer_SuccessPage_HTML(t *testing.T) {
	srv, err := newCallbackServer()
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	srv.Start()

	port := srv.Port()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := http.Get("http://127.0.0.1:" + itoa(port) + "/callback?code=test&state=s") //nolint:gosec
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content-type, got %q", ct)
	}
	if !strings.Contains(string(body), "Authenticated") {
		t.Error("success page should contain 'Authenticated'")
	}

	_, _ = srv.WaitForCallback(ctx)
}

// itoa is a simple int-to-string helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
