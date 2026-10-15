//go:build windows

// Package picker owns the native desktop folder-selection adapter.
package picker

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"
)

func Choose() (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms;$d=New-Object System.Windows.Forms.FolderBrowserDialog;$d.Description='Choose a music folder for VexStream';$d.ShowNewFolderButton=$false;if($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK){[Console]::Write($d.SelectedPath)}`
	command := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-STA",
		"-Command",
		script,
	)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", errors.New("folder selection cancelled")
	}
	return path, nil
}
