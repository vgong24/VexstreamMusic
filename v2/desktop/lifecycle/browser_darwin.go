//go:build darwin

package lifecycle

import "os/exec"

func OpenBrowser(url string) error {
	return exec.Command("/usr/bin/open", url).Start()
}
