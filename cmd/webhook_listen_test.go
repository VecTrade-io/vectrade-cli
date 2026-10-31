package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunWebhookListen_StopsWhenContextIsCancelled(t *testing.T) {
	streamed := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if _, err := w.Write([]byte(": keepalive\n\n")); err != nil {
			t.Errorf("write keepalive: %v", err)
		}
		w.(http.Flusher).Flush()
		close(streamed)
		<-r.Context().Done()
	}))
	defer srv.Close()

	t.Setenv("VECTRADE_API_KEY", "vq_test_key_12345")
	t.Setenv("VECTRADE_BASE_URL", srv.URL)

	origPort, origForward := listenPort, listenForward
	listenPort, listenForward = 5050, "http://localhost:3000/hooks"
	defer func() { listenPort, listenForward = origPort, origForward }()

	ctx, cancel := context.WithCancel(context.Background())
	webhookListenCmd.SetContext(ctx)
	defer webhookListenCmd.SetContext(context.Background())

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan error, 1)
	go func() { done <- runWebhookListen(webhookListenCmd, nil) }()

	select {
	case <-streamed:
	case <-time.After(5 * time.Second):
		t.Fatal("listener never connected to the event stream")
	}
	cancel()

	var err error
	select {
	case err = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("listener did not stop after the context was cancelled")
	}

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := readTestOutput(t, r)
	for _, want := range []string{"port 5050", "Forwarding to: http://localhost:3000/hooks", "Stopped listening"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got: %s", want, output)
		}
	}
}

func TestRunWebhookListen_RequiresCredentials(t *testing.T) {
	t.Setenv("VECTRADE_API_KEY", "")
	t.Setenv("HOME", t.TempDir())

	origCfg := cfgFile
	cfgFile = filepath.Join(t.TempDir(), "missing.yaml")
	defer func() { cfgFile = origCfg }()

	if err := runWebhookListen(webhookListenCmd, nil); err == nil {
		t.Fatal("expected an error when no API key is configured")
	}
}
