//go:build darwin

// Package picker owns the native desktop folder-selection adapter.
package picker

import (
	"errors"
	"os/exec"
	"strings"
)

func Choose() (string, error) {
	output, err := exec.Command(
		"/usr/bin/osascript",
		"-e",
		`POSIX path of (choose folder with prompt "Choose a music folder for VexStream")`,
	).Output()
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", errors.New("folder selection cancelled")
	}
	return path, nil
}
