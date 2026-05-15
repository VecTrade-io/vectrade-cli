package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP (Model Context Protocol) configurations",
}

var mcpSetupCmd = &cobra.Command{
	Use:   "setup [ide]",
	Short: "Configure MCP for an AI IDE (cursor, vscode, claude, windsurf)",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPSetup,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpSetupCmd)
}

func runMCPSetup(cmd *cobra.Command, args []string) error {
	ide := args[0]

	configDir, fileName, err := getMCPConfigPath(ide)
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	configPath := filepath.Join(configDir, fileName)

	// Read existing config or start fresh
	existing := make(map[string]any)
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	// Add VecTrade MCP server entry
	mcpServers, ok := existing["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
	}

	mcpServers["vectrade"] = map[string]any{
		"command": "uvx",
		"args":    []string{"vectrade-mcp"},
		"env": map[string]string{
			"VECTRADE_API_KEY": "${VECTRADE_API_KEY}",
		},
	}
	existing["mcpServers"] = mcpServers

	// Write config
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("✓ VecTrade MCP configured for %s\n", ide)
	fmt.Printf("  Config: %s\n", configPath)
	fmt.Printf("\n  Make sure VECTRADE_API_KEY is set in your environment.\n")
	return nil
}

func getMCPConfigPath(ide string) (dir string, filename string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("finding home directory: %w", err)
	}

	switch ide {
	case "cursor":
		return filepath.Join(home, ".cursor"), "mcp.json", nil
	case "vscode":
		switch runtime.GOOS {
		case "windows":
			return filepath.Join(os.Getenv("APPDATA"), "Code", "User"), "settings.json", nil
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "Code", "User"), "settings.json", nil
		default:
			return filepath.Join(home, ".config", "Code", "User"), "settings.json", nil
		}
	case "claude":
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "Claude"), "claude_desktop_config.json", nil
		case "windows":
			return filepath.Join(os.Getenv("APPDATA"), "Claude"), "claude_desktop_config.json", nil
		default:
			return filepath.Join(home, ".config", "claude"), "claude_desktop_config.json", nil
		}
	case "windsurf":
		return filepath.Join(home, ".windsurf"), "mcp.json", nil
	case "cline":
		return filepath.Join(home, ".cline"), "mcp_settings.json", nil
	default:
		return "", "", fmt.Errorf("unsupported IDE: %s (supported: cursor, vscode, claude, windsurf, cline)", ide)
	}
}
