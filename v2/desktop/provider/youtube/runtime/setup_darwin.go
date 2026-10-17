//go:build darwin

package runtimeenv

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func Setup(ctx context.Context, sharedRuntime string) (string, error) {
	base := findDarwinPython()
	brew := findBrew()
	if base == "" && brew != "" {
		if err := runSetup(ctx, 20*time.Minute, brew, "install", "python"); err != nil {
			return "", fmt.Errorf("install Python with Homebrew: %w", err)
		}
		base = findDarwinPython()
	}
	if base == "" {
		return "", errors.New("Python 3.10+ was not found. Install Python 3 once, then retry setup")
	}

	venv := filepath.Join(sharedRuntime, "venv")
	python := filepath.Join(venv, "bin", "python3")
	if _, err := os.Stat(python); err != nil {
		if err := os.MkdirAll(filepath.Dir(venv), 0o755); err != nil {
			return "", err
		}
		if err := runSetup(
			ctx,
			5*time.Minute,
			base,
			"-m",
			"venv",
			venv,
		); err != nil {
			return "", fmt.Errorf("create private VexStream Python runtime: %w", err)
		}
	}
	if err := runSetup(
		ctx,
		10*time.Minute,
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

	if findFFmpeg() == "" && brew != "" {
		_ = runSetup(ctx, 30*time.Minute, brew, "install", "ffmpeg")
	}
	if findFFmpeg() == "" {
		return "YouTube search and inspection are ready. MP3 import still needs FFmpeg.", nil
	}
	return "YouTube search, inspection, and MP3 import tools are ready on this Mac.", nil
}

func findDarwinPython() string {
	candidates := []string{
		"/opt/homebrew/bin/python3",
		"/usr/local/bin/python3",
		"/Library/Frameworks/Python.framework/Versions/Current/bin/python3",
		"/opt/local/bin/python3",
		"/usr/bin/python3",
	}
	if path, err := exec.LookPath("python3"); err == nil {
		candidates = append([]string{path}, candidates...)
	}
	if matches, _ := filepath.Glob(
		"/Library/Frameworks/Python.framework/Versions/*/bin/python3",
	); len(matches) > 0 {
		candidates = append(candidates, matches...)
	}
	for _, candidate := range candidates {
		if python310OrNewer(candidate) {
			return candidate
		}
	}
	return ""
}

func findBrew() string {
	for _, candidate := range []string{
		"/opt/homebrew/bin/brew",
		"/usr/local/bin/brew",
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	if path, err := exec.LookPath("brew"); err == nil {
		return strings.TrimSpace(path)
	}
	return ""
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
