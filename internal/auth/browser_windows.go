package auth

import "os/exec"

// openBrowser opens the given URL in the default browser on Windows.
func openBrowser(url string) error {
	return exec.Command("cmd", "/c", "start", url).Start()
}
