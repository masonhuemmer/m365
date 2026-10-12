package graph

import (
	"fmt"
	"os"
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

// OpenBrowser opens the OAuth authorize URL. When no browser can be launched it prints
// the URL so the user can open it by hand; login keeps waiting on the loopback listener.
func OpenBrowser(rawURL string) error {
	name, args := browserCommand(runtime.GOOS, rawURL)
	// argv, not a shell; the URL is the OAuth authorize endpoint.
	if err := exec.Command(name, args...).Start(); err != nil { // #nosec G204
		fmt.Fprintf(os.Stderr, "Could not open a browser (%v). Open this URL to sign in:\n%s\n", err, rawURL)
	}
	return nil
}
