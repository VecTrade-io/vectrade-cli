package auth

import "os/exec"

// openBrowser opens the given URL in the default browser on Windows.
func openBrowser(url string) error {
	// "cmd /c start" would let cmd.exe interpret & and | in the query string.
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start() //nolint:gosec // no shell involved
}
