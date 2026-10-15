// Package runtimeenv owns discovery/materialization of the local YouTube provider runtime.
//
// The Python environment is intentionally reusable with V3. The V4 bridge script itself is
// materialized side-by-side by appdata placement.
package runtimeenv

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"
)

type Status struct {
	SearchReady bool   `json:"searchReady"`
	FFmpegReady bool   `json:"ffmpegReady"`
	ImportReady bool   `json:"importReady"`
	Python      string `json:"python,omitempty"`
	FFmpeg      string `json:"ffmpeg,omitempty"`
	Detail      string `json:"detail"`
}

func Resolve(sharedRuntime string) Status {
	python := findPython(sharedRuntime)
	ffmpeg := findFFmpeg()
	status := Status{
		SearchReady: python != "",
		FFmpegReady: ffmpeg != "",
		ImportReady: python != "" && ffmpeg != "",
		Python:      python,
		FFmpeg:      ffmpeg,
	}
	switch {
	case status.ImportReady:
		status.Detail = "YouTube search, inspection, and MP3 import tools are ready."
	case status.SearchReady:
		status.Detail = "YouTube search and source inspection are ready; MP3 import still needs FFmpeg."
	default:
		status.Detail = "Local playback is ready; YouTube search/import needs one-time runtime setup."
	}
	return status
}

func BridgeEnv() []string {
	env := os.Environ()
	if goruntime.GOOS == "darwin" {
		base := os.Getenv("PATH")
		path := strings.Join([]string{
			"/opt/homebrew/bin",
			"/usr/local/bin",
			"/opt/local/bin",
			"/usr/bin",
			"/bin",
			"/usr/sbin",
			"/sbin",
			base,
		}, string(os.PathListSeparator))
		filtered := make([]string, 0, len(env)+3)
		for _, item := range env {
			if !strings.HasPrefix(item, "PATH=") {
				filtered = append(filtered, item)
			}
		}
		env = append(filtered, "PATH="+path)
	}
	return append(env, "PYTHONUTF8=1", "PYTHONIOENCODING=utf-8")
}

func MaterializeBridge(path string, data []byte) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("provider bridge path required")
	}
	if len(data) == 0 {
		return "", errors.New("provider bridge bytes required")
	}
	digest := sha256.Sum256(data)
	ref := hex.EncodeToString(digest[:])

	if current, err := os.ReadFile(path); err == nil && bytes.Equal(current, data) {
		return ref, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return ref, nil
}

func validPython(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		path,
		"-c",
		"import yt_dlp, curl_cffi",
	)
	command.Env = BridgeEnv()
	command.SysProcAttr = hiddenProcAttr()
	return command.Run() == nil
}

func python310OrNewer(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		path,
		"-c",
		"import sys; raise SystemExit(0 if sys.version_info >= (3,10) else 1)",
	)
	command.Env = BridgeEnv()
	command.SysProcAttr = hiddenProcAttr()
	return command.Run() == nil
}

func privatePython(sharedRuntime string) string {
	if goruntime.GOOS == "windows" {
		return filepath.Join(sharedRuntime, "venv", "Scripts", "python.exe")
	}
	return filepath.Join(sharedRuntime, "venv", "bin", "python3")
}

func findPython(sharedRuntime string) string {
	candidates := []string{privatePython(sharedRuntime)}
	if goruntime.GOOS == "windows" {
		local := os.Getenv("LOCALAPPDATA")
		candidates = append(candidates,
			filepath.Join(
				local,
				"VexMediaRescue",
				"runtime",
				"venv",
				"Scripts",
				"python.exe",
			),
		)
	}
	if goruntime.GOOS == "darwin" {
		candidates = append(candidates,
			"/opt/homebrew/bin/python3",
			"/usr/local/bin/python3",
			"/Library/Frameworks/Python.framework/Versions/Current/bin/python3",
			"/opt/local/bin/python3",
			"/usr/bin/python3",
		)
	}
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			candidates = append(candidates, path)
		}
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		if validPython(candidate) {
			return candidate
		}
	}
	return ""
}

func findFFmpeg() string {
	candidates := []string{}
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		candidates = append(candidates, path)
	}
	if goruntime.GOOS == "darwin" {
		candidates = append(candidates,
			"/opt/homebrew/bin/ffmpeg",
			"/usr/local/bin/ffmpeg",
			"/opt/local/bin/ffmpeg",
		)
	}
	if goruntime.GOOS == "windows" {
		local := os.Getenv("LOCALAPPDATA")
		candidates = append(candidates,
			filepath.Join(
				local,
				"Microsoft",
				"WinGet",
				"Links",
				"ffmpeg.exe",
			),
			findWindowsWinGetFFmpeg(local),
		)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func findWindowsWinGetFFmpeg(local string) string {
	if goruntime.GOOS != "windows" || strings.TrimSpace(local) == "" {
		return ""
	}
	packages := filepath.Join(local, "Microsoft", "WinGet", "Packages")
	roots, _ := filepath.Glob(filepath.Join(packages, "Gyan.FFmpeg_*"))
	for _, root := range roots {
		found := ""
		_ = filepath.Walk(root, func(
			path string,
			info os.FileInfo,
			err error,
		) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			if strings.EqualFold(info.Name(), "ffmpeg.exe") {
				found = path
				return filepath.SkipAll
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}
