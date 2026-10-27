//go:build windows

package runtimeenv

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func Setup(ctx context.Context, sharedRuntime string) (string, error) {
	venv := filepath.Join(sharedRuntime, "venv")
	python := filepath.Join(venv, "Scripts", "python.exe")
	if _, err := os.Stat(python); err != nil {
		base := ""
		args := []string{}
		if path, lookErr := exec.LookPath("py.exe"); lookErr == nil {
			base = path
			args = []string{"-3", "-m", "venv", venv}
		} else if path, lookErr := exec.LookPath("python.exe"); lookErr == nil {
			base = path
			args = []string{"-m", "venv", venv}
		} else {
			return "", errors.New("Python 3 was not found. Install Python 3.10+ first, then retry")
		}
		if err := runSetup(ctx, 5*time.Minute, base, args...); err != nil {
			return "", fmt.Errorf("create private VexStream Python runtime: %w", err)
		}
	}
	if err := runSetup(
		ctx,
		15*time.Minute,
		python,
		"-m",
		"pip",
		"install",
		"--upgrade",
		"pip",
	); err != nil {
		return "", fmt.Errorf("update private Python runtime: %w", err)
	}
	if err := runSetup(
		ctx,
		15*time.Minute,
		python,
		"-m",
		"pip",
		"install",
		"yt-dlp[default,curl-cffi]",
	); err != nil {
		return "", fmt.Errorf("install YouTube dependencies: %w", err)
	}
	if !validPython(python) {
		return "", errors.New("private VexStream Python runtime did not validate")
	}

	if findFFmpeg() == "" {
		if winget, err := exec.LookPath("winget.exe"); err == nil {
			_ = runSetup(
				ctx,
				20*time.Minute,
				winget,
				"install",
				"--id",
				"Gyan.FFmpeg",
				"-e",
				"--accept-package-agreements",
				"--accept-source-agreements",
				"--silent",
			)
		}
	}
	if findFFmpeg() == "" {
		return "YouTube search and inspection are ready. MP3 import still needs FFmpeg.", nil
	}
	return "YouTube search, inspection, and MP3 import tools are ready on this PC.", nil
}

func runSetup(
	parent context.Context,
	timeout time.Duration,
	name string,
	args ...string,
) error {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	command := exec.CommandContext(ctx, name, args...)
	command.Env = BridgeEnv()
	command.SysProcAttr = hiddenProcAttr()
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"%s failed: %s",
			filepath.Base(name),
			string(output),
		)
	}
	return nil
}
