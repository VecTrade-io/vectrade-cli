// Package api provides the HTTP client for the VecTrade API.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/VecTrade-io/vectrade-cli/internal/config"
)

// maxResponseSize is the maximum allowed response body size (10MB).
const maxResponseSize = 10 * 1024 * 1024

// Client is the VecTrade API HTTP client.
type Client struct {
	baseURL    string
	authType   config.AuthType
	authValue  string
	httpClient *http.Client
	userAgent  string
}

// ClientOption configures the API client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// NewClient creates an API client from resolved config.
func NewClient(cfg *config.Config, opts ...ClientOption) *Client {
	authType, authValue := cfg.AuthInfo()
	c := &Client{
		baseURL:   cfg.BaseURL,
		authType:  authType,
		authValue: authValue,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		userAgent: "vectrade-cli/" + Version,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Version is set at build time via ldflags.
var Version = "dev"

// APIError represents an error response from the VecTrade API.
type APIError struct {
	StatusCode int
	Type       string `json:"type"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("API error %d (%s): %s [request_id: %s]", e.StatusCode, e.Type, e.Message, e.RequestID)
	}
	return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.Type, e.Message)
}

// Get performs a GET request to the given path.
func (c *Client) Get(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Set(k, v)
		}
		reqURL += "?" + values.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, body)
	}

	return body, nil
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, body)
	}

	return body, nil
}

// StreamGet performs a GET request and returns the response body as a reader for SSE streaming.
func (c *Client) StreamGet(ctx context.Context, path string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		resp.Body.Close()
		return nil, parseAPIError(resp.StatusCode, body)
	}

	return resp.Body, nil
}

// StreamPost performs a POST request and returns the response body for SSE streaming.
func (c *Client) StreamPost(ctx context.Context, path string, payload any) (io.ReadCloser, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		resp.Body.Close()
		return nil, parseAPIError(resp.StatusCode, body)
	}

	return resp.Body, nil
}

func (c *Client) setHeaders(req *http.Request) {
	switch c.authType {
	case config.AuthTypeAPIKey:
		req.Header.Set("X-API-Key", c.authValue)
	case config.AuthTypeJWT:
		req.Header.Set("Authorization", "Bearer "+c.authValue)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
}

// Delete performs a DELETE request to the given path.
func (c *Client) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		return parseAPIError(resp.StatusCode, body)
	}

	return nil
}

func parseAPIError(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(body, &errResp); err != nil {
		return &APIError{
			StatusCode: statusCode,
			Type:       "unknown",
			Message:    string(body),
		}
	}
	return &APIError{
		StatusCode: statusCode,
		Type:       errResp.Error.Type,
		Message:    errResp.Error.Message,
		RequestID:  errResp.RequestID,
	}
}
