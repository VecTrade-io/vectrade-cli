package cmd

import "path/filepath"

func mcpVSCodePath(home string) (string, string, error) {
	return filepath.Join(home, "Library", "Application Support", "Code", "User"), "settings.json", nil
}

func mcpClaudePath(home string) (string, string, error) {
	return filepath.Join(home, "Library", "Application Support", "Claude"), "claude_desktop_config.json", nil
}
