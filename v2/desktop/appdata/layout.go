// Package appdata owns Desktop V4's persistence placement.
//
// Shared paths are intentionally V3-readable. V4-only runtime/recovery paths live below v4/ so a
// protected V3 daily driver never has to understand their schemas.
package appdata

import (
	"os"
	"path/filepath"
	"runtime"
)

type Layout struct {
	Base string

	// Shared with V3. V4 writers must preserve the accepted V3-readable shape.
	Config           string
	ConfigBackup     string
	LibraryCache     string
	FirstSeen        string
	Overlays         string
	PlaybackWindows  string
	Playlists        string
	TrashState       string
	DeletedHistory   string
	TrashFiles       string
	SharedRuntime    string

	// V4-only custody.
	V4Root           string
	CurrentURL       string
	ImportStaging    string
	TrashPurgeJournal string
	ProviderBridge   string
}

func Base() string {
	if runtime.GOOS == "windows" {
		if value := os.Getenv("LOCALAPPDATA"); value != "" {
			return filepath.Join(value, "VexStreamMusic")
		}
	}
	if value, err := os.UserConfigDir(); err == nil {
		return filepath.Join(value, "VexStreamMusic")
	}
	return filepath.Join(os.TempDir(), "VexStreamMusic")
}

func At(base string) Layout {
	base = filepath.Clean(base)
	v4 := filepath.Join(base, "v4")
	return Layout{
		Base: base,
		Config: filepath.Join(base, "config.json"),
		ConfigBackup: filepath.Join(base, "backups", "config.latest.json"),
		LibraryCache: filepath.Join(base, "library-index.json"),
		FirstSeen: filepath.Join(base, "track-first-seen.json"),
		Overlays: filepath.Join(base, "library-overlays.json"),
		PlaybackWindows: filepath.Join(base, "playback-windows.json"),
		Playlists: filepath.Join(base, "playlists.json"),
		TrashState: filepath.Join(base, "trash.json"),
		DeletedHistory: filepath.Join(base, "deleted-history.json"),
		TrashFiles: filepath.Join(base, "trash", "files"),
		SharedRuntime: filepath.Join(base, "runtime"),
		V4Root: v4,
		CurrentURL: filepath.Join(v4, "current-url.txt"),
		ImportStaging: filepath.Join(v4, "staging"),
		TrashPurgeJournal: filepath.Join(v4, "trash-purge-journal.json"),
		ProviderBridge: filepath.Join(v4, "bridge", "media_bridge.py"),
	}
}
