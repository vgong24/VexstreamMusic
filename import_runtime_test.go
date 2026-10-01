package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindWindowsWinGetFFmpegFindsNestedGyanPackage(t *testing.T) {
	local := t.TempDir()
	ffmpeg := filepath.Join(local, "Microsoft", "WinGet", "Packages", "Gyan.FFmpeg_Microsoft.Winget.Source_8wekyb3d8bbwe", "ffmpeg-8.0-full_build", "bin", "ffmpeg.exe")
	if err := os.MkdirAll(filepath.Dir(ffmpeg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ffmpeg, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := findWindowsWinGetFFmpeg(local)
	if got != ffmpeg {
		t.Fatalf("expected %q, got %q", ffmpeg, got)
	}
}

func TestFindWindowsWinGetFFmpegIgnoresUnrelatedPackages(t *testing.T) {
	local := t.TempDir()
	unrelated := filepath.Join(local, "Microsoft", "WinGet", "Packages", "Other.Package_Source", "bin", "ffmpeg.exe")
	if err := os.MkdirAll(filepath.Dir(unrelated), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unrelated, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findWindowsWinGetFFmpeg(local); got != "" {
		t.Fatalf("expected no Gyan FFmpeg discovery, got %q", got)
	}
}
