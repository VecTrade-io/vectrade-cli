package cmd

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetMCPConfigPath_Cursor(t *testing.T) {
	dir, filename, err := getMCPConfigPath("cursor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filename != "mcp.json" {
		t.Errorf("expected mcp.json, got %s", filename)
	}
	if !strings.Contains(dir, ".cursor") {
		t.Errorf("expected .cursor in path, got %s", dir)
	}
}

func TestGetMCPConfigPath_VSCode(t *testing.T) {
	dir, filename, err := getMCPConfigPath("vscode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filename != "settings.json" {
		t.Errorf("expected settings.json, got %s", filename)
	}
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(dir, "Code/User") {
			t.Errorf("expected Code/User in path, got %s", dir)
		}
	case "linux":
		if !strings.Contains(dir, ".config/Code/User") {
			t.Errorf("expected .config/Code/User in path, got %s", dir)
		}
	}
}

func TestGetMCPConfigPath_Claude(t *testing.T) {
	dir, filename, err := getMCPConfigPath("claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	switch runtime.GOOS {
	case "darwin":
		if filename != "claude_desktop_config.json" {
			t.Errorf("expected claude_desktop_config.json, got %s", filename)
		}
		if !strings.Contains(dir, "Claude") {
			t.Errorf("expected Claude in path, got %s", dir)
		}
	}
}

func TestGetMCPConfigPath_Windsurf(t *testing.T) {
	dir, filename, err := getMCPConfigPath("windsurf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filename != "mcp.json" {
		t.Errorf("expected mcp.json, got %s", filename)
	}
	if !strings.Contains(dir, ".windsurf") {
		t.Errorf("expected .windsurf in path, got %s", dir)
	}
}

func TestGetMCPConfigPath_Cline(t *testing.T) {
	dir, filename, err := getMCPConfigPath("cline")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filename != "mcp_settings.json" {
		t.Errorf("expected mcp_settings.json, got %s", filename)
	}
	if !strings.Contains(dir, ".cline") {
		t.Errorf("expected .cline in path, got %s", dir)
	}
}

func TestGetMCPConfigPath_Unsupported(t *testing.T) {
	_, _, err := getMCPConfigPath("neovim")
	if err == nil {
		t.Error("expected error for unsupported IDE")
	}
	if !strings.Contains(err.Error(), "unsupported IDE") {
		t.Errorf("expected 'unsupported IDE' in error, got: %v", err)
	}
}
