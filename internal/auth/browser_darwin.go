package auth

import "os/exec"

// openBrowser opens the given URL in the default browser on macOS.
func openBrowser(url string) error {
	return exec.Command("open", url).Start() //nolint:gosec // URL is passed as a single argv entry, not through a shell
}
