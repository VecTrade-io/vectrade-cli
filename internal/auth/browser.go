// Package auth implements OAuth browser-based authentication for the VecTrade CLI.
package auth

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openBrowser opens the given URL in the user's default browser.
// It supports macOS, Linux, and Windows.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform %s — open this URL manually:\n  %s", runtime.GOOS, url)
	}

	return cmd.Start()
}
