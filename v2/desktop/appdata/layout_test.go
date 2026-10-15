package appdata

import (
	"path/filepath"
	"testing"
)

func TestLayoutKeepsV3ReadableStoresSharedAndV4RecoverySideBySide(t *testing.T) {
	base := filepath.Join(t.TempDir(), "VexStreamMusic")
	got := At(base)

	shared := map[string]string{
		"config":    got.Config,
		"cache":     got.LibraryCache,
		"firstSeen": got.FirstSeen,
		"overlays":  got.Overlays,
		"windows":   got.PlaybackWindows,
		"playlists": got.Playlists,
		"trash":     got.TrashState,
		"history":   got.DeletedHistory,
	}
	for name, path := range shared {
		if filepath.Dir(path) != base && name != "config" {
			if name != "config" {
				// config backup/trash files/runtime are intentionally nested but still shared.
			}
		}
		if path == "" {
			t.Fatalf("shared %s path missing", name)
		}
	}

	if got.Config != filepath.Join(base, "config.json") ||
		got.LibraryCache != filepath.Join(base, "library-index.json") ||
		got.FirstSeen != filepath.Join(base, "track-first-seen.json") ||
		got.Overlays != filepath.Join(base, "library-overlays.json") ||
		got.PlaybackWindows != filepath.Join(base, "playback-windows.json") ||
		got.Playlists != filepath.Join(base, "playlists.json") ||
		got.TrashState != filepath.Join(base, "trash.json") ||
		got.DeletedHistory != filepath.Join(base, "deleted-history.json") {
		t.Fatalf("protected V3 store placement drift: %+v", got)
	}

	for name, path := range map[string]string{
		"currentURL":    got.CurrentURL,
		"staging":       got.ImportStaging,
		"playerSession": got.PlayerSession,
		"purgeJournal":  got.TrashPurgeJournal,
		"bridge":        got.ProviderBridge,
	} {
		relative, err := filepath.Rel(got.V4Root, path)
		if err != nil ||
			relative == ".." ||
			filepath.IsAbs(relative) ||
			relative == "." {
			t.Fatalf("V4-only %s escaped v4 root: %q err=%v", name, path, err)
		}
	}
}

func TestSharedProviderRuntimeStaysOutsideV4SpecificRoot(t *testing.T) {
	base := filepath.Join(t.TempDir(), "VexStreamMusic")
	got := At(base)
	if got.SharedRuntime != filepath.Join(base, "runtime") {
		t.Fatalf("shared runtime placement drift: %q", got.SharedRuntime)
	}
	if got.SharedRuntime == filepath.Join(got.V4Root, "runtime") {
		t.Fatal("V4 unnecessarily duplicated established provider runtime")
	}
}

func TestPlayerSessionStaysSideBySideWithProtectedV3Stores(t *testing.T) {
	base := filepath.Join(t.TempDir(), "VexStreamMusic")
	got := At(base)

	if got.PlayerSession != filepath.Join(
		base,
		"v4",
		"player-session.json",
	) {
		t.Fatalf("player session placement drift: %q", got.PlayerSession)
	}

	if got.PlayerSession == filepath.Join(base, "queue.json") ||
		got.PlayerSession == filepath.Join(base, "player-session.json") {
		t.Fatal("V4 listening session leaked into protected V3 root")
	}
}

func TestProtectedCurrentURLRemainsV3OwnedOutsideV4Root(t *testing.T) {
	base := filepath.Join(t.TempDir(), "VexStreamMusic")
	got := At(base)

	if got.ProtectedCurrentURL != filepath.Join(base, "current-url.txt") {
		t.Fatalf("protected V3 claim placement drift: %q", got.ProtectedCurrentURL)
	}
	relative, err := filepath.Rel(got.V4Root, got.ProtectedCurrentURL)
	if err != nil {
		t.Fatal(err)
	}
	if relative != filepath.Join("..", "current-url.txt") {
		t.Fatalf("protected V3 claim was absorbed into V4 custody: %q", relative)
	}
}
