package auth

import "os/exec"

// openBrowser opens the given URL in the default browser on Linux.
func openBrowser(url string) error {
	return exec.Command("xdg-open", url).Start()
}
