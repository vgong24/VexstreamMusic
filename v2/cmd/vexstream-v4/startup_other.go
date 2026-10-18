//go:build !windows

package main

import (
	"fmt"
	"os"

	"github.com/vgong24/VexstreamMusic/v2/desktop/lifecycle"
)

func handoffProtectedDesktop(
	_ string,
	claim lifecycle.LiveClaim,
) (bool, error) {
	return false, protectedDesktopConflict(
		claim,
	)
}

func reportStartupFailure(
	startupErr error,
) {
	fmt.Fprintln(
		os.Stderr,
		"VexStream Desktop V4:",
		startupErr,
	)
}
