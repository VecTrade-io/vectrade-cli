package cmd

import (
	"os"
	"path/filepath"
)

func mcpVSCodePath(home string) (string, string, error) {
	return filepath.Join(os.Getenv("APPDATA"), "Code", "User"), "settings.json", nil
}

func mcpClaudePath(home string) (string, string, error) {
	return filepath.Join(os.Getenv("APPDATA"), "Claude"), "claude_desktop_config.json", nil
}
