package graph

import (
	"os/exec"
	"runtime"
)

// browserCommand returns the argv that opens rawURL in the default browser on goos.
// Windows uses rundll32 because cmd /c start treats & in the URL as a command separator.
func browserCommand(goos, rawURL string) (string, []string) {
	switch goos {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", rawURL}
	case "darwin":
		return "open", []string{rawURL}
	default:
		return "xdg-open", []string{rawURL}
	}
}

// OpenBrowser opens the OAuth authorize URL in the default browser.
func OpenBrowser(rawURL string) error {
	name, args := browserCommand(runtime.GOOS, rawURL)
	// argv, not a shell; the URL is the OAuth authorize endpoint.
	return exec.Command(name, args...).Start() // #nosec G204
}
