package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

func TestFirstSeenUsesV3CompatibleMapShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "track-state.json")
	if err := os.WriteFile(path, []byte(`{"track-a":"2026-09-29T00:00:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := LoadFirstSeen(path)
	if err != nil {
		t.Fatal(err)
	}
	if state["track-a"] != "2026-09-29T00:00:00Z" {
		t.Fatalf("firstSeen compatibility drift: %+v", state)
	}
	state["track-b"] = "2026-09-30T00:00:00Z"
	if err := SaveFirstSeen(path, state); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadFirstSeen(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded) != 2 || reloaded["track-b"] == "" {
		t.Fatalf("firstSeen round-trip drift: %+v", reloaded)
	}
}

func TestOverlayStorePreservesV3PointerShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "track-overlays.json")
	if err := os.WriteFile(path, []byte(`{"track-a":{"artist":"Human Artist","updatedAt":"overlay-1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := LoadOverlays(path)
	if err != nil {
		t.Fatal(err)
	}
	if state["track-a"].Artist == nil || *state["track-a"].Artist != "Human Artist" || state["track-a"].Title != nil {
		t.Fatalf("overlay compatibility drift: %+v", state)
	}
	if err := SaveOverlays(path, state); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadOverlays(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded["track-a"].Artist == nil || *reloaded["track-a"].Artist != "Human Artist" {
		t.Fatalf("overlay round-trip drift: %+v", reloaded)
	}
}

func TestPlaybackWindowStorePreservesV3Shape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playback-windows.json")
	if err := os.WriteFile(path, []byte(`{"track-a":{"start":12.5,"end":42,"updatedAt":"window-1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := LoadPlaybackWindows(path)
	if err != nil {
		t.Fatal(err)
	}
	window := state["track-a"]
	if window.Start != 12.5 || window.End == nil || *window.End != 42 || window.UpdatedAt != "window-1" {
		t.Fatalf("playback window compatibility drift: %+v", window)
	}
	if err := SavePlaybackWindows(path, map[string]library.PlaybackWindow{"track-a": window}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadPlaybackWindows(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded["track-a"].End == nil || *reloaded["track-a"].End != 42 {
		t.Fatalf("playback window round-trip drift: %+v", reloaded)
	}
}

func TestMissingProjectionStateFilesStartEmpty(t *testing.T) {
	root := t.TempDir()
	firstSeen, err := LoadFirstSeen(filepath.Join(root, "missing-first-seen.json"))
	if err != nil || len(firstSeen) != 0 {
		t.Fatalf("missing firstSeen should be empty: %+v %v", firstSeen, err)
	}
	overlays, err := LoadOverlays(filepath.Join(root, "missing-overlays.json"))
	if err != nil || len(overlays) != 0 {
		t.Fatalf("missing overlays should be empty: %+v %v", overlays, err)
	}
	windows, err := LoadPlaybackWindows(filepath.Join(root, "missing-windows.json"))
	if err != nil || len(windows) != 0 {
		t.Fatalf("missing windows should be empty: %+v %v", windows, err)
	}
}
