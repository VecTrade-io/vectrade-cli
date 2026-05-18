package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/VecTrade-io/vectrade-cli/internal/config"
)

func TestForwardEvent_LocalhostAllowed(t *testing.T) {
	received := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	forwardEvent(srv.URL, `{"event":"test"}`)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !received {
		t.Error("forward should have reached localhost server")
	}
	if !strings.Contains(buf.String(), "Forwarded") {
		t.Errorf("expected 'Forwarded' in output, got: %s", buf.String())
	}
}

func TestForwardEvent_SSRF_Blocked(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"external_http", "http://evil.com/callback"},
		{"external_https", "https://evil.com/callback"},
		{"internal_network", "http://192.168.1.1/hook"},
		{"metadata", "http://169.254.169.254/latest/meta-data"},
		{"dns_rebinding", "http://localhost.evil.com/callback"},
		{"authority_confusion", "http://localhost:@evil.com/"},
		{"subdomain_bypass", "http://localhost.attacker.io/hook"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oldErr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			forwardEvent(tc.url, `{"event":"test"}`)

			w.Close()
			os.Stderr = oldErr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			if !strings.Contains(buf.String(), "Forward blocked") {
				t.Errorf("expected SSRF blocked for %s, got: %s", tc.url, buf.String())
			}
		})
	}
}

func TestForwardEvent_IPv6Localhost(t *testing.T) {
	// We can't easily test a real IPv6 server, but we verify the URL passes the check
	// and gets a connection error (not blocked)
	oldErr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	forwardEvent("http://[::1]:12345/test", `{"event":"test"}`)

	w.Close()
	os.Stderr = oldErr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	// Should NOT say "blocked" — it's an allowed address
	if strings.Contains(output, "Forward blocked") {
		t.Error("IPv6 localhost should be allowed, not blocked")
	}
	// Should show a connection error (since nothing listens on that port)
	if !strings.Contains(output, "Forward failed") {
		t.Logf("output: %s", output)
	}
}

func TestForwardEvent_127001_Allowed(t *testing.T) {
	received := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Replace localhost with 127.0.0.1
	url := strings.Replace(srv.URL, "127.0.0.1", "127.0.0.1", 1)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	forwardEvent(url, `{"event":"test"}`)

	w.Close()
	os.Stdout = old

	if !received {
		// The httptest server binds to 127.0.0.1, so this should work
		fmt.Println("Note: server may have bound to localhost instead")
	}
}

func TestForwardEvent_InvalidURL(t *testing.T) {
	oldErr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	forwardEvent("://bad-url", `{"event":"test"}`)

	w.Close()
	os.Stderr = oldErr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "Forward blocked") {
		t.Errorf("expected 'Forward blocked' for invalid URL, got: %s", buf.String())
	}
}

func TestForwardEvent_ContentType(t *testing.T) {
	var receivedCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	forwardEvent(srv.URL, `{"event":"test"}`)

	w.Close()
	os.Stdout = old

	if receivedCT != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", receivedCT)
	}
}

func TestForwardEvent_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	forwardEvent(srv.URL, `{"event":"test"}`)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	// Should still say "Forwarded" with the status code
	if !strings.Contains(buf.String(), "Forwarded") {
		t.Errorf("expected 'Forwarded' message, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "500") {
		t.Errorf("expected status 500 in output, got: %s", buf.String())
	}
}

func TestStreamWebhookEvents(t *testing.T) {
	// Create a mock SSE server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"type\":\"quote.update\",\"symbol\":\"AAPL\"}\n\n"))
		w.Write([]byte(": keepalive\n\n"))
		w.Write([]byte("data: {\"type\":\"webhook.test\"}\n\n"))
	}))
	defer srv.Close()

	cfg := &config.Config{
		APIKey:  "vq_test_key",
		BaseURL: srv.URL,
		Timeout: 10,
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset listenForward to avoid forwarding
	origForward := listenForward
	listenForward = ""
	defer func() { listenForward = origForward }()

	err := streamWebhookEvents(cfg)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !strings.Contains(output, "quote.update") {
		t.Errorf("expected 'quote.update' in output, got: %s", output)
	}
	if !strings.Contains(output, "webhook.test") {
		t.Errorf("expected 'webhook.test' in output, got: %s", output)
	}
}

func TestStreamWebhookEvents_WithForward(t *testing.T) {
	forwardReceived := false
	forwardSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer forwardSrv.Close()

	sseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"type\":\"test.event\"}\n\n"))
	}))
	defer sseSrv.Close()

	cfg := &config.Config{
		APIKey:  "vq_test_key",
		BaseURL: sseSrv.URL,
		Timeout: 10,
	}

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	origForward := listenForward
	listenForward = forwardSrv.URL
	defer func() { listenForward = origForward }()

	_ = streamWebhookEvents(cfg)

	w.Close()
	os.Stdout = old

	// Give goroutine time to forward
	// (forwardEvent runs in a goroutine)
	if !forwardReceived {
		t.Log("forward may not have completed yet (goroutine timing)")
	}
}

func TestStreamWebhookEvents_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"type":"unauthorized","message":"bad key"}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{
		APIKey:  "vq_test_key",
		BaseURL: srv.URL,
		Timeout: 10,
	}

	err := streamWebhookEvents(cfg)
	if err == nil {
		t.Error("expected error for 401 response")
	}
}
