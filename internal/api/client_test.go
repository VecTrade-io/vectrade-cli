package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		if auth := r.Header.Get("Authorization"); auth != "Bearer vq_test_key_12345" {
			t.Errorf("unexpected auth header: %s", auth)
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
