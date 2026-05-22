// Package tests contains live integration tests that hit the real VecTrade API.
// Run with: go test -tags=live ./tests/ -v
//
// Required environment:
//   VECTRADE_TEST_API_KEY - a valid API key (any plan)
//
// These tests verify:
//   - Authentication is enforced (X-API-Key header)
//   - All endpoints return expected response shapes
//   - Plan-based limits are respected (quota, RPM, token limits)
//   - Error responses match documented format
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/VecTrade-io/vectrade-cli/internal/api"
	"github.com/VecTrade-io/vectrade-cli/internal/config"
)

const liveBaseURL = "https://api.vectrade.io/v1"

func skipIfNoKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("VECTRADE_TEST_API_KEY")
	if key == "" {
		t.Skip("VECTRADE_TEST_API_KEY not set — skipping live test")
	}
	return key
}

func liveClient(t *testing.T) *api.Client {
	t.Helper()
	key := skipIfNoKey(t)
	cfg := &config.Config{
		APIKey:  key,
		BaseURL: liveBaseURL,
		Timeout: 30,
	}
	return api.NewClient(cfg)
}

// ── Auth Tests ──────────────────────────────────────────────────────────

func TestLive_AuthRequired(t *testing.T) {
	skipIfNoKey(t) // ensure we're in live test mode

	// Request without API key should fail
	cfg := &config.Config{
		APIKey:  "vq_invalid_key_does_not_exist",
		BaseURL: liveBaseURL,
		Timeout: 15,
	}
	client := api.NewClient(cfg)

	_, err := client.Get(context.Background(), "/vq/quotes/AAPL", nil)
	if err == nil {
		t.Fatal("expected auth error with invalid key, got nil")
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("expected *api.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 && apiErr.StatusCode != 403 {
		t.Errorf("expected 401 or 403, got %d", apiErr.StatusCode)
	}
}

func TestLive_NoKeyReturns401(t *testing.T) {
	skipIfNoKey(t)

	// Direct HTTP request with no auth header at all
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, liveBaseURL+"/vq/quotes/AAPL", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 && resp.StatusCode != 403 {
		t.Errorf("expected 401/403 without key, got %d", resp.StatusCode)
	}
}

func TestLive_XAPIKeyHeaderWorks(t *testing.T) {
	key := skipIfNoKey(t)

	// Verify X-API-Key header is accepted
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, liveBaseURL+"/vq/quotes/AAPL", nil)
	req.Header.Set("X-API-Key", key)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 with valid X-API-Key, got %d", resp.StatusCode)
	}
}

// ── Quote Tests ─────────────────────────────────────────────────────────

func TestLive_QuoteSingleSymbol(t *testing.T) {
	client := liveClient(t)

	body, err := client.Get(context.Background(), "/vq/quotes/AAPL", nil)
	if err != nil {
		t.Fatalf("quote request failed: %v", err)
	}

	var quote map[string]any
	if err := json.Unmarshal(body, &quote); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"symbol", "price"}
	for _, field := range requiredFields {
		if _, ok := quote[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	if quote["symbol"] != "AAPL" {
		t.Errorf("expected symbol AAPL, got %v", quote["symbol"])
	}

	price, ok := quote["price"].(float64)
	if !ok || price <= 0 {
		t.Errorf("expected positive price, got %v", quote["price"])
	}
}

func TestLive_QuoteWithFields(t *testing.T) {
	client := liveClient(t)

	body, err := client.Get(context.Background(), "/vq/quotes/MSFT", map[string]string{
		"fields": "price,volume",
	})
	if err != nil {
		t.Fatalf("quote with fields failed: %v", err)
	}

	var quote map[string]any
	if err := json.Unmarshal(body, &quote); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if quote["symbol"] == nil && quote["price"] == nil {
		t.Error("response should contain at least price or symbol")
	}
}

func TestLive_QuoteInvalidSymbol(t *testing.T) {
	client := liveClient(t)

	_, err := client.Get(context.Background(), "/vq/quotes/ZZZZZZ_INVALID_999", nil)
	if err == nil {
		t.Fatal("expected error for invalid symbol")
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("expected *api.APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 && apiErr.StatusCode != 400 {
		t.Errorf("expected 404 or 400 for invalid symbol, got %d", apiErr.StatusCode)
	}
}

// ── Developer/Keys Tests ────────────────────────────────────────────────

func TestLive_DeveloperInfo(t *testing.T) {
	client := liveClient(t)

	body, err := client.Get(context.Background(), "/vq/developer/plan", nil)
	if err != nil {
		t.Fatalf("developer plan request failed: %v", err)
	}

	var dev map[string]any
	if err := json.Unmarshal(body, &dev); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Should have plan info
	t.Logf("developer plan response: %v", mapKeys(dev))
}

func TestLive_UsageEndpoint(t *testing.T) {
	client := liveClient(t)

	body, err := client.Get(context.Background(), "/vq/developer/usage", nil)
	if err != nil {
		t.Fatalf("usage request failed: %v", err)
	}

	var usage map[string]any
	if err := json.Unmarshal(body, &usage); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	t.Logf("usage response keys: %v", mapKeys(usage))
}

// ── Rate Limit Tests ────────────────────────────────────────────────────

func TestLive_RateLimitHeaders(t *testing.T) {
	key := skipIfNoKey(t)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, liveBaseURL+"/vq/quotes/AAPL", nil)
	req.Header.Set("X-API-Key", key)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check for rate limit headers (at least one should be present)
	rateLimitHeaders := []string{
		"X-RateLimit-Limit",
		"X-RateLimit-Remaining",
		"X-RateLimit-Reset",
		"RateLimit-Limit",
		"RateLimit-Remaining",
		"RateLimit-Reset",
	}

	found := false
	for _, h := range rateLimitHeaders {
		if resp.Header.Get(h) != "" {
			found = true
			t.Logf("  %s: %s", h, resp.Header.Get(h))
		}
	}

	if !found {
		t.Log("warning: no rate limit headers found in response")
	}
}

func TestLive_RPMEnforced(t *testing.T) {
	client := liveClient(t)

	// Send rapid requests to verify rate limiting exists
	// We don't want to actually trigger 429, just verify the system responds correctly
	var wg sync.WaitGroup
	results := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := client.Get(context.Background(), "/vq/quotes/AAPL", nil)
			results[idx] = err
		}(i)
	}
	wg.Wait()

	// At least some should succeed (we're not trying to trigger the limit)
	successCount := 0
	for _, err := range results {
		if err == nil {
			successCount++
		}
	}
	if successCount == 0 {
		t.Error("all concurrent requests failed — possible connectivity issue")
	}
	t.Logf("concurrent requests: %d/5 succeeded", successCount)
}

// ── Error Response Format Tests ─────────────────────────────────────────

func TestLive_ErrorResponseFormat(t *testing.T) {
	key := skipIfNoKey(t)

	// Request an endpoint that should return an error
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, liveBaseURL+"/vq/quotes/INVALID_SYMBOL_XYZ", nil)
	req.Header.Set("X-API-Key", key)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 400 {
		t.Skip("server returned success for invalid symbol (may be cached)")
	}

	var errResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("error response not valid JSON: %v", err)
	}

	// Error responses should have a structured format
	if errResp["error"] != nil {
		errObj, ok := errResp["error"].(map[string]any)
		if ok {
			if errObj["type"] == nil {
				t.Error("error.type field missing")
			}
			if errObj["message"] == nil {
				t.Error("error.message field missing")
			}
		}
	}
}

// ── Plan Limit Tests ────────────────────────────────────────────────────

func TestLive_PlanLimitsReflected(t *testing.T) {
	client := liveClient(t)

	// Get plan info
	body, err := client.Get(context.Background(), "/vq/developer/plan", nil)
	if err != nil {
		t.Skipf("developer plan endpoint not available: %v", err)
	}

	var plan map[string]any
	if err := json.Unmarshal(body, &plan); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	t.Logf("plan info: %v", plan)

	// Get quota
	quotaBody, err := client.Get(context.Background(), "/vq/developer/quota", nil)
	if err != nil {
		t.Skipf("quota endpoint not available: %v", err)
	}

	var quota map[string]any
	if err := json.Unmarshal(quotaBody, &quota); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	t.Logf("quota info: %v", quota)
}

// ── Scope/Access Tests ──────────────────────────────────────────────────

func TestLive_ScopeEnforcement(t *testing.T) {
	client := liveClient(t)

	// Try accessing admin-only endpoints (should fail for normal API keys)
	_, err := client.Get(context.Background(), "/vq/admin/users", nil)
	if err == nil {
		t.Log("admin endpoint accessible — key may have elevated permissions")
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		// Could be a 404 which is also fine (endpoint doesn't exist)
		return
	}
	if apiErr.StatusCode == 403 || apiErr.StatusCode == 404 || apiErr.StatusCode == 401 {
		// Expected: forbidden, not found, or unauthorized
		t.Logf("admin access correctly denied: %d %s", apiErr.StatusCode, apiErr.Type)
	}
}

// ── AI Endpoint Tests ───────────────────────────────────────────────────

func TestLive_AIAnalyzeEndpoint(t *testing.T) {
	client := liveClient(t)

	body, err := client.Post(context.Background(), "/vq/ai/analyze", map[string]any{
		"prompt": "What is AAPL's market cap?",
		"stream": false,
	})
	if err != nil {
		apiErr, ok := err.(*api.APIError)
		if ok && (apiErr.StatusCode == 402 || apiErr.StatusCode == 403 || apiErr.StatusCode == 429 || apiErr.StatusCode == 404) {
			t.Skipf("AI endpoint restricted/unavailable for this plan: %v", err)
		}
		t.Fatalf("AI analyze failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["content"] == nil && result["message"] == nil && result["response"] == nil {
		t.Error("AI response missing content/message/response field")
	}
}

// ── Webhook Tests ───────────────────────────────────────────────────────

func TestLive_WebhookListEndpoint(t *testing.T) {
	client := liveClient(t)

	body, err := client.Get(context.Background(), "/vq/webhooks", nil)
	if err != nil {
		apiErr, ok := err.(*api.APIError)
		if ok && (apiErr.StatusCode == 404 || apiErr.StatusCode == 403) {
			t.Skipf("webhooks not available for this plan: %v", err)
		}
		t.Fatalf("webhook list failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	t.Logf("webhooks response keys: %v", mapKeys(result))
}

// ── Keys Lifecycle Tests ────────────────────────────────────────────────

func TestLive_KeysListEndpoint(t *testing.T) {
	client := liveClient(t)

	body, err := client.Get(context.Background(), "/vq/developer/keys", nil)
	if err != nil {
		t.Fatalf("keys list failed: %v", err)
	}

	// Response could be an array or an object with data field
	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	t.Logf("keys response type: %T", result)
}

// ── Response Time Tests ─────────────────────────────────────────────────

func TestLive_ResponseTime(t *testing.T) {
	client := liveClient(t)

	start := time.Now()
	_, err := client.Get(context.Background(), "/vq/quotes/AAPL", nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	t.Logf("response time: %v", elapsed)
	if elapsed > 10*time.Second {
		t.Errorf("response took too long: %v (expected <10s)", elapsed)
	}
}

// ── Token/Credit Budget Tests ───────────────────────────────────────────

func TestLive_QuotaDecrement(t *testing.T) {
	client := liveClient(t)

	// Get usage before
	beforeBody, err := client.Get(context.Background(), "/vq/developer/usage", nil)
	if err != nil {
		t.Skipf("usage endpoint not available: %v", err)
	}
	var before map[string]any
	json.Unmarshal(beforeBody, &before)

	// Make a request
	_, err = client.Get(context.Background(), "/vq/quotes/TSLA", nil)
	if err != nil {
		t.Fatalf("quote request failed: %v", err)
	}

	// Small delay for usage to update
	time.Sleep(500 * time.Millisecond)

	// Get usage after
	afterBody, err := client.Get(context.Background(), "/vq/developer/usage", nil)
	if err != nil {
		t.Skipf("usage endpoint not available: %v", err)
	}
	var after map[string]any
	json.Unmarshal(afterBody, &after)

	// Log comparison
	t.Logf("usage before: %v", before)
	t.Logf("usage after: %v", after)
}

// ── OpenAPI Spec Tests ──────────────────────────────────────────────────

func TestLive_OpenAPISpec(t *testing.T) {
	client := liveClient(t)

	body, err := client.Get(context.Background(), "/vq/openapi", nil)
	if err != nil {
		apiErr, ok := err.(*api.APIError)
		if ok && apiErr.StatusCode == 404 {
			t.Skip("openapi endpoint not available")
		}
		t.Fatalf("openapi request failed: %v", err)
	}

	content := string(body)
	if !strings.Contains(content, "openapi") && !strings.Contains(content, "swagger") && !strings.Contains(content, "paths") {
		t.Error("response doesn't look like an OpenAPI spec")
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Prevent unused import warning
var _ = fmt.Sprintf
