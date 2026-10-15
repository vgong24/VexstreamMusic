//go:build windows

package lifecycle

import "os/exec"

func OpenBrowser(url string) error {
	return exec.Command(
		"rundll32.exe",
		"url.dll,FileProtocolHandler",
		url,
	).Start()
}
