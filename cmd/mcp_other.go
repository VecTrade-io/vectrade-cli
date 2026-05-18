//go:build !darwin && !windows

package cmd

import "path/filepath"

func mcpVSCodePath(home string) (string, string, error) {
	return filepath.Join(home, ".config", "Code", "User"), "settings.json", nil
}

func mcpClaudePath(home string) (string, string, error) {
	return filepath.Join(home, ".config", "claude"), "claude_desktop_config.json", nil
}
