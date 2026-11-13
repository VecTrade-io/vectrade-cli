package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWebhookListen_RejectsUnsupportedStreamingWithoutConnecting(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
	}))
	defer srv.Close()

	t.Setenv("VECTRADE_API_KEY", "vq_test_key_12345")
	t.Setenv("VECTRADE_BASE_URL", srv.URL)

	err := runWebhookListen(webhookListenCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "streaming is not available") {
		t.Fatalf("expected an explicit unsupported-feature error, got %v", err)
	}
	if requests != 0 {
		t.Fatalf("unexpected requests to an unsupported streaming endpoint: %d", requests)
	}
	if !webhookListenCmd.Hidden {
		t.Fatal("unsupported listener must be hidden from command help")
	}
}

func TestRunWebhookListen_RequiresCredentials(t *testing.T) {
	t.Setenv("VECTRADE_API_KEY", "")
	t.Setenv("VECTRADE_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")

	// Flag variables persist between tests in this package.
	origCfg, origKey := cfgFile, apiKey
	cfgFile, apiKey = filepath.Join(t.TempDir(), "missing.yaml"), ""
	defer func() { cfgFile, apiKey = origCfg, origKey }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	webhookListenCmd.SetContext(ctx)
	defer webhookListenCmd.SetContext(context.Background())

	if err := runWebhookListen(webhookListenCmd, nil); err == nil {
		t.Fatal("expected an error when no API key is configured")
	}
}
