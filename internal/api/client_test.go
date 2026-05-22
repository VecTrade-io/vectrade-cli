package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VecTrade-io/vectrade-cli/internal/config"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	srv := httptest.NewServer(handler)
	cfg := &config.Config{
		APIKey:  "vq_test_key_12345",
		BaseURL: srv.URL,
		Timeout: 10,
	}
	client := NewClient(cfg)
	return srv, client
}

func TestGet_Success(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "vq_test_key_12345" {
			t.Errorf("unexpected X-API-Key header: %s", apiKey)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"symbol": "AAPL", "price": 198.5})
	})
	defer srv.Close()

	body, err := client.Get(context.Background(), "/quotes/AAPL", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result["symbol"] != "AAPL" {
		t.Errorf("expected AAPL, got %v", result["symbol"])
	}
}

func TestGet_WithParams(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fields") != "price,volume" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"price": 198.5}`))
	})
	defer srv.Close()

	_, err := client.Get(context.Background(), "/quotes/AAPL", map[string]string{"fields": "price,volume"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGet_APIError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error":      map[string]any{"type": "not_found", "message": "Symbol not found"},
			"request_id": "req_123",
		})
	})
	defer srv.Close()

	_, err := client.Get(context.Background(), "/quotes/INVALID", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
	if apiErr.Type != "not_found" {
		t.Errorf("expected not_found, got %s", apiErr.Type)
	}
	if apiErr.RequestID != "req_123" {
		t.Errorf("expected req_123, got %s", apiErr.RequestID)
	}
}

func TestPost_Success(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected JSON content-type, got %s", ct)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content": "Analysis result"}`))
	})
	defer srv.Close()

	body, err := client.Post(context.Background(), "/ai/analyze", map[string]any{
		"prompt": "Analyze AAPL",
		"stream": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result["content"] != "Analysis result" {
		t.Errorf("unexpected content: %v", result["content"])
	}
}

func TestUserAgent(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua == "" {
			t.Error("expected User-Agent header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, _ = client.Get(context.Background(), "/test", nil)
}

func TestGet_ParamsURLEncoded(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Verify special characters are properly encoded
		q := r.URL.Query().Get("query")
		if q != "market cap > 1B & pe < 20" {
			t.Errorf("unexpected decoded query: %s", q)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := client.Get(context.Background(), "/screener", map[string]string{
		"query": "market cap > 1B & pe < 20",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	err := client.Delete(context.Background(), "/webhooks/wh_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_APIError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error":      map[string]any{"type": "forbidden", "message": "insufficient permissions"},
			"request_id": "req_del_err",
		})
	})
	defer srv.Close()

	err := client.Delete(context.Background(), "/keys/k_123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("expected 403, got %d", apiErr.StatusCode)
	}
}

func TestStreamGet_Success(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("expected SSE accept header, got %s", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"type\":\"quote.update\"}\n\n"))
	})
	defer srv.Close()

	body, err := client.StreamGet(context.Background(), "/webhooks/listen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	buf := make([]byte, 256)
	n, _ := body.Read(buf)
	if n == 0 {
		t.Error("expected non-empty stream body")
	}
}

func TestStreamGet_APIError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"type":"unauthorized","message":"invalid token"}}`))
	})
	defer srv.Close()

	_, err := client.StreamGet(context.Background(), "/webhooks/listen")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
}

func TestStreamPost_Success(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content-type, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("expected SSE accept header, got %s", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"content\":\"hello\"}\n\ndata: [DONE]\n\n"))
	})
	defer srv.Close()

	body, err := client.StreamPost(context.Background(), "/ai/analyze", map[string]any{"prompt": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	buf := make([]byte, 256)
	n, _ := body.Read(buf)
	if n == 0 {
		t.Error("expected non-empty stream body")
	}
}

func TestStreamPost_APIError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"type":"rate_limited","message":"too many requests"},"request_id":"req_rl"}`))
	})
	defer srv.Close()

	_, err := client.StreamPost(context.Background(), "/ai/analyze", map[string]any{"prompt": "test"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 429 {
		t.Errorf("expected 429, got %d", apiErr.StatusCode)
	}
	if apiErr.RequestID != "req_rl" {
		t.Errorf("expected req_rl, got %s", apiErr.RequestID)
	}
}

func TestAPIError_Error_WithRequestID(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Type:       "not_found",
		Message:    "Symbol not found",
		RequestID:  "req_abc",
	}
	got := err.Error()
	if got != "API error 404 (not_found): Symbol not found [request_id: req_abc]" {
		t.Errorf("unexpected error string: %s", got)
	}
}

func TestAPIError_Error_WithoutRequestID(t *testing.T) {
	err := &APIError{
		StatusCode: 500,
		Type:       "internal",
		Message:    "server error",
	}
	got := err.Error()
	if got != "API error 500 (internal): server error" {
		t.Errorf("unexpected error string: %s", got)
	}
}

func TestParseAPIError_NonJSON(t *testing.T) {
	err := parseAPIError(502, []byte("Bad Gateway"))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 502 {
		t.Errorf("expected 502, got %d", apiErr.StatusCode)
	}
	if apiErr.Type != "unknown" {
		t.Errorf("expected 'unknown' type, got %s", apiErr.Type)
	}
	if apiErr.Message != "Bad Gateway" {
		t.Errorf("expected raw body as message, got %s", apiErr.Message)
	}
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 99}
	cfg := &config.Config{
		APIKey:  "vq_test",
		BaseURL: "https://api.vectrade.io/v1",
		Timeout: 10,
	}
	client := NewClient(cfg, WithHTTPClient(customClient))
	if client.httpClient != customClient {
		t.Error("WithHTTPClient option did not set custom client")
	}
}

func TestPost_APIError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error":      map[string]any{"type": "validation", "message": "invalid payload"},
			"request_id": "req_post_err",
		})
	})
	defer srv.Close()

	_, err := client.Post(context.Background(), "/keys", map[string]string{"label": ""})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Type != "validation" {
		t.Errorf("expected 'validation', got %s", apiErr.Type)
	}
}

func TestGet_ResponseSizeLimit(t *testing.T) {
	// Server returns a body larger than maxResponseSize
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write 11MB of data (exceeds 10MB limit)
		w.Write([]byte(strings.Repeat("x", maxResponseSize+1024)))
	})
	defer srv.Close()

	body, err := client.Get(context.Background(), "/large", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Body should be capped at maxResponseSize
	if len(body) > maxResponseSize {
		t.Errorf("response body %d bytes exceeds limit %d", len(body), maxResponseSize)
	}
}

func TestGet_QueryParams(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("version") != "2.0" {
			t.Errorf("expected version=2.0, got %s", r.URL.Query().Get("version"))
		}
		if r.URL.Query().Get("format") != "yaml" {
			t.Errorf("expected format=yaml, got %s", r.URL.Query().Get("format"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	defer srv.Close()

	_, err := client.Get(context.Background(), "/test", map[string]string{
		"version": "2.0",
		"format":  "yaml",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMaxResponseSizeConstant(t *testing.T) {
	expected := 10 * 1024 * 1024
	if maxResponseSize != expected {
		t.Errorf("maxResponseSize = %d, want %d (10MB)", maxResponseSize, expected)
	}
}

func TestPost_ResponseSizeLimit(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strings.Repeat("y", maxResponseSize+512)))
	})
	defer srv.Close()

	body, err := client.Post(context.Background(), "/large", map[string]string{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) > maxResponseSize {
		t.Errorf("POST response %d bytes exceeds limit %d", len(body), maxResponseSize)
	}
}

func TestDelete_ResponseSizeLimit(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		// Large error response
		w.Write([]byte(strings.Repeat("e", maxResponseSize+512)))
	})
	defer srv.Close()

	err := client.Delete(context.Background(), "/large")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestStreamGet_ResponseSizeLimit(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(strings.Repeat("z", maxResponseSize+100)))
	})
	defer srv.Close()

	_, err := client.StreamGet(context.Background(), "/large")
	if err == nil {
		t.Fatal("expected error for 400")
	}
}

func TestStreamPost_ResponseSizeLimit(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(strings.Repeat("w", maxResponseSize+100)))
	})
	defer srv.Close()

	_, err := client.StreamPost(context.Background(), "/large", map[string]string{"key": "val"})
	if err == nil {
		t.Fatal("expected error for 400")
	}
}

func TestGet_NoParams(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	})
	defer srv.Close()

	_, err := client.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewClient_UserAgent(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()
	Version = "1.5.0"

	cfg := &config.Config{
		APIKey:  "vq_test",
		BaseURL: "https://api.vectrade.io/v1",
		Timeout: 10,
	}
	client := NewClient(cfg)
	if client.userAgent != "vectrade-cli/1.5.0" {
		t.Errorf("userAgent = %q, want %q", client.userAgent, "vectrade-cli/1.5.0")
	}
}

func TestNewClient_BaseURL(t *testing.T) {
	cfg := &config.Config{
		APIKey:  "vq_test",
		BaseURL: "https://custom.api.io/v2",
		Timeout: 30,
	}
	client := NewClient(cfg)
	if client.baseURL != "https://custom.api.io/v2" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://custom.api.io/v2")
	}
}

func TestSetHeaders(t *testing.T) {
	cfg := &config.Config{
		APIKey:  "vq_test_headers",
		BaseURL: "https://api.vectrade.io/v1",
		Timeout: 10,
	}
	client := NewClient(cfg)

	req, _ := http.NewRequest(http.MethodGet, "https://api.vectrade.io/v1/test", nil)
	client.setHeaders(req)

	if got := req.Header.Get("X-API-Key"); got != "vq_test_headers" {
		t.Errorf("X-API-Key = %q, want vq_test_headers", got)
	}
	if got := req.Header.Get("User-Agent"); !strings.HasPrefix(got, "vectrade-cli/") {
		t.Errorf("User-Agent = %q, expected vectrade-cli/ prefix", got)
	}
	if got := req.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q, want application/json", got)
	}
}

func TestSetHeaders_JWT(t *testing.T) {
	cfg := &config.Config{
		JWTToken: "jwt_test_token",
		BaseURL:  "https://api.vectrade.io/v1",
		Timeout:  10,
	}
	client := NewClient(cfg)

	req, _ := http.NewRequest(http.MethodGet, "https://api.vectrade.io/v1/test", nil)
	client.setHeaders(req)

	if got := req.Header.Get("Authorization"); got != "Bearer jwt_test_token" {
		t.Errorf("Authorization = %q, want 'Bearer jwt_test_token'", got)
	}
	if got := req.Header.Get("X-API-Key"); got != "" {
		t.Errorf("X-API-Key should be empty for JWT auth, got %q", got)
	}
}

func TestVersion_Default(t *testing.T) {
	// Version should have a default
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestAPIError_Error_EmptyType(t *testing.T) {
	err := &APIError{StatusCode: 503, Type: "", Message: "unavailable"}
	got := err.Error()
	if !strings.Contains(got, "503") {
		t.Errorf("expected 503 in error, got: %s", got)
	}
	if !strings.Contains(got, "unavailable") {
		t.Errorf("expected 'unavailable' in error, got: %s", got)
	}
}

func TestGet_ContextCanceled(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Get(ctx, "/test", nil)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestPost_ContextCanceled(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Post(ctx, "/test", map[string]string{"key": "val"})
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestStreamGet_ContextCanceled(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.StreamGet(ctx, "/test")
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestStreamPost_ContextCanceled(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.StreamPost(ctx, "/test", map[string]string{"key": "val"})
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestDelete_ContextCanceled(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Delete(ctx, "/test")
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestPost_MarshalError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	// Channel values can't be marshaled to JSON
	_, err := client.Post(context.Background(), "/test", make(chan int))
	if err == nil {
		t.Error("expected marshal error")
	}
	if !strings.Contains(err.Error(), "marshaling") {
		t.Errorf("expected marshaling error, got: %v", err)
	}
}

func TestStreamPost_MarshalError(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	_, err := client.StreamPost(context.Background(), "/test", make(chan int))
	if err == nil {
		t.Error("expected marshal error")
	}
}

func TestStreamGet_Success_ReturnsBody(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: hello\n\n"))
	})
	defer srv.Close()

	body, err := client.StreamGet(context.Background(), "/stream")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	buf := make([]byte, 100)
	n, _ := body.Read(buf)
	if !strings.Contains(string(buf[:n]), "hello") {
		t.Errorf("expected 'hello' in stream, got: %s", string(buf[:n]))
	}
}

func TestStreamPost_Success_ReturnsBody(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: world\n\n"))
	})
	defer srv.Close()

	body, err := client.StreamPost(context.Background(), "/stream", map[string]string{"q": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer body.Close()

	buf := make([]byte, 100)
	n, _ := body.Read(buf)
	if !strings.Contains(string(buf[:n]), "world") {
		t.Errorf("expected 'world' in stream, got: %s", string(buf[:n]))
	}
}

func TestDelete_Success_NoContent(t *testing.T) {
	srv, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent) // 204
	})
	defer srv.Close()

	err := client.Delete(context.Background(), "/items/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAPIError_WithRequestID(t *testing.T) {
	body := `{"error":{"type":"rate_limit","message":"too many requests"},"request_id":"req_abc"}`
	err := parseAPIError(429, []byte(body))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.RequestID != "req_abc" {
		t.Errorf("RequestID = %q", apiErr.RequestID)
	}
	errStr := apiErr.Error()
	if !strings.Contains(errStr, "req_abc") {
		t.Errorf("error string missing request_id: %s", errStr)
	}
	if !strings.Contains(errStr, "rate_limit") {
		t.Errorf("error string missing type: %s", errStr)
	}
}
