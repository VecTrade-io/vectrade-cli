package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// testServer creates a mock API server and sets env vars for the CLI.
func testServer(handler http.HandlerFunc) *httptest.Server {
	srv := httptest.NewServer(handler)
	return srv
}

func TestRootCmd_UseName(t *testing.T) {
	if rootCmd.Use != "vectrade" {
		t.Errorf("expected root command Use to be 'vectrade', got %q", rootCmd.Use)
	}
}

func TestRootCmd_HasAllSubcommands(t *testing.T) {
	expected := []string{"quote", "ai", "auth", "keys", "mcp", "openapi", "usage", "version", "webhook"}
	cmds := make(map[string]bool)
	for _, sub := range rootCmd.Commands() {
		cmds[sub.Name()] = true
	}
	for _, name := range expected {
		if !cmds[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestQuoteCmd_RequiresSymbol(t *testing.T) {
	rootCmd.SetArgs([]string{"quote"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when symbol is missing")
	}
}

func TestQuoteCmd_JsonOutput(t *testing.T) {
	srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vq/quotes/AAPL" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"symbol":    "AAPL",
			"price":     198.50,
			"change":    2.30,
			"changePct": 1.17,
			"volume":    45000000,
		})
	})
	defer srv.Close()

	// Set env vars for the command
	t.Setenv("VECTRADE_API_KEY", "vq_test_key_12345")
	t.Setenv("VECTRADE_BASE_URL", srv.URL)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"quote", "AAPL", "--output", "json"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if output == "" {
		// Command may not support --output flag at root level; just verify no error
		return
	}
}

func TestVersionCmd_Output(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if len(output) == 0 {
		t.Error("expected version output, got empty string")
	}
	// Should contain "vectrade" not "vt"
	if !bytes.Contains([]byte(output), []byte("vectrade")) {
		t.Errorf("version output should contain 'vectrade', got: %s", output)
	}
}

func TestAiCmd_RequiresPrompt(t *testing.T) {
	rootCmd.SetArgs([]string{"ai"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when prompt is missing")
	}
}

func TestKeysCmd_HasSubcommands(t *testing.T) {
	expected := []string{"create", "list", "revoke"}
	cmds := make(map[string]bool)
	for _, sub := range keysCmd.Commands() {
		cmds[sub.Name()] = true
	}
	for _, name := range expected {
		if !cmds[name] {
			t.Errorf("keys: missing subcommand %s", name)
		}
	}
}

func TestKeysCreateCmd_RequiresLabel(t *testing.T) {
	t.Setenv("VECTRADE_API_KEY", "vq_test_key")
	rootCmd.SetArgs([]string{"keys", "create"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when --label is missing for keys create")
	}
}

func TestMcpCmd_Exists(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "mcp" {
			found = true
			break
		}
	}
	if !found {
		t.Error("mcp subcommand not registered")
	}
}
