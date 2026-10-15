package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/filesystem"
)

func fixturePath(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "tests", "fixtures", "vexstream-test-tone.mp3")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture unavailable: %v", err)
	}
	return path
}

func copyFixture(t *testing.T, destination string) {
	t.Helper()
	data, err := os.ReadFile(fixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func paths(root string) Paths {
	return Paths{
		Cache:           filepath.Join(root, "library-index.json"),
		FirstSeen:       filepath.Join(root, "track-first-seen.json"),
		Overlays:        filepath.Join(root, "library-overlays.json"),
		PlaybackWindows: filepath.Join(root, "playback-windows.json"),
	}
}

func str(value string) *string       { return &value }
func number(value float64) *float64 { return &value }

func TestOpenFailsClosedOnCorruptHumanProjectionState(t *testing.T) {
	root := t.TempDir()
	p := paths(root)
	if err := os.WriteFile(p.Overlays, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p); err == nil {
		t.Fatal("corrupt human overlay state was silently dropped")
	}
}

func TestOpenReportsDiscardedLegacyCacheWithoutDiscardingV3CompatibleHumanState(t *testing.T) {
	root := t.TempDir()
	p := paths(root)
	if err := os.WriteFile(
		p.Cache,
		[]byte(`{"schemaVersion":"vexstream.library-index/v1","entries":[{"path":"/stale.mp3"}]}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.SaveFirstSeen(p.FirstSeen, map[string]string{"track-a": "first"}); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.SaveOverlays(
		p.Overlays,
		map[string]library.Overlay{"track-a": {Artist: str("Human Artist")}},
	); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.SavePlaybackWindows(
		p.PlaybackWindows,
		map[string]library.PlaybackWindow{"track-a": {Start: 12.5, End: number(42)}},
	); err != nil {
		t.Fatal(err)
	}

	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if !opened.DiscardedLegacyCache {
		t.Fatal("legacy rebuildable cache was not surfaced as discarded")
	}
	if got := opened.Library.firstSeen["track-a"]; got != "first" {
		t.Fatalf("firstSeen lost: %q", got)
	}
	if got := opened.Library.overlays["track-a"].Artist; got == nil || *got != "Human Artist" {
		t.Fatalf("overlay lost: %+v", opened.Library.overlays)
	}
	if got := opened.Library.windows["track-a"].End; got == nil || *got != 42 {
		t.Fatalf("playback window lost: %+v", opened.Library.windows)
	}
}

func TestReconcileComposesScanPersistenceAndProjection(t *testing.T) {
	root := t.TempDir()
	stateRoot := filepath.Join(root, "state")
	source := filepath.Join(root, "music")
	media := filepath.Join(source, "tone.mp3")
	copyFixture(t, media)
	absolute, _ := filepath.Abs(media)
	trackID := filesystem.TrackID(absolute)
	p := paths(stateRoot)
	if err := filesystem.SaveOverlays(
		p.Overlays,
		map[string]library.Overlay{
			trackID: {
				Title:     str("Display Tone"),
				Artist:    str("Victor's Artist"),
				UpdatedAt: "overlay-1",
			},
		},
	); err != nil {
		t.Fatal(err)
	}
	if err := filesystem.SavePlaybackWindows(
		p.PlaybackWindows,
		map[string]library.PlaybackWindow{
			trackID: {
				Start:     0.25,
				End:       number(0.75),
				UpdatedAt: "window-1",
			},
		},
	); err != nil {
		t.Fatal(err)
	}

	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := opened.Library.Reconcile([]string{source})
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Tracks) != 1 {
		t.Fatalf("expected one projected track: %+v", projection.Tracks)
	}
	track := projection.Tracks[0]
	if track.ID != trackID || track.Title != "Display Tone" || track.Artist != "Victor's Artist" {
		t.Fatalf("projection drift: %+v", track)
	}
	if track.SourceTitle != "VexStream Test Tone" || track.SourceArtist != "VexStream QA" {
		t.Fatalf("raw source truth not preserved: %+v", track)
	}
	if track.PlaybackStart != 0.25 || track.PlaybackEnd == nil || *track.PlaybackEnd != 0.75 {
		t.Fatalf("playback window missing: %+v", track)
	}
	if projection.Paths[trackID] != absolute {
		t.Fatalf("private path custody drift: %+v", projection.Paths)
	}

	if _, err := os.Stat(p.Cache); err != nil {
		t.Fatalf("cache not persisted: %v", err)
	}
	firstSeen, err := filesystem.LoadFirstSeen(p.FirstSeen)
	if err != nil || firstSeen[trackID] == "" {
		t.Fatalf("firstSeen not persisted: %+v %v", firstSeen, err)
	}
	cache, err := filesystem.LoadCache(p.Cache)
	if err != nil || cache.Entries[filesystem.CacheKey(absolute)].Track.SourceTitle != "" {
		t.Fatalf("raw cache absorbed display projection: %+v %v", cache, err)
	}
}

func TestSnapshotCanUseLoadedCacheForRecoveryWithoutActiveSources(t *testing.T) {
	root := t.TempDir()
	p := paths(root)
	entry := library.CacheEntry{
		Path: "/lost/song.mp3",
		Track: library.Track{
			ID:     "lost",
			Path:   "/lost/song.mp3",
			Source: "/lost",
			Title:  "Recovered",
		},
	}
	if err := filesystem.SaveCache(
		p.Cache,
		map[string]library.CacheEntry{filesystem.CacheKey(entry.Path): entry},
	); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	ordinary := opened.Library.Snapshot(nil, false)
	if len(ordinary.Tracks) != 0 {
		t.Fatalf("inactive source leaked into ordinary projection: %+v", ordinary)
	}
	recovery := opened.Library.Snapshot(nil, true)
	if len(recovery.Tracks) != 1 || recovery.Tracks[0].ID != "lost" {
		t.Fatalf("recovery projection lost cache: %+v", recovery)
	}
}

func TestReconcileDropsRawEntriesNoLongerInActiveSources(t *testing.T) {
	root := t.TempDir()
	p := paths(filepath.Join(root, "state"))
	sourceA := filepath.Join(root, "a")
	sourceB := filepath.Join(root, "b")
	copyFixture(t, filepath.Join(sourceA, "a.mp3"))
	copyFixture(t, filepath.Join(sourceB, "b.mp3"))

	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	first, err := opened.Library.Reconcile([]string{sourceA, sourceB})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Tracks) != 2 {
		t.Fatalf("initial reconcile drift: %+v", first.Tracks)
	}
	second, err := opened.Library.Reconcile([]string{sourceA})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Tracks) != 1 || second.Tracks[0].Source != sourceA {
		t.Fatalf("removed source retained in catalog: %+v", second.Tracks)
	}
	cache, err := filesystem.LoadCache(p.Cache)
	if err != nil || len(cache.Entries) != 1 {
		t.Fatalf("raw cache retained inactive source: %+v %v", cache, err)
	}
}

func TestReconcileWritesFirstSeenBeforeCacheSoCacheFailureCannotEraseIdentity(t *testing.T) {
	root := t.TempDir()
	p := paths(filepath.Join(root, "state"))
	source := filepath.Join(root, "music")
	copyFixture(t, filepath.Join(source, "tone.mp3"))

	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}

	// Force only the cache destination to fail after Open; FirstSeen remains writable.
	if err := os.MkdirAll(p.Cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := opened.Library.Reconcile([]string{source}); err == nil {
		t.Fatal("cache persistence failure was not surfaced")
	}
	firstSeen, err := filesystem.LoadFirstSeen(p.FirstSeen)
	if err != nil || len(firstSeen) != 1 {
		t.Fatalf("firstSeen identity was not preserved before cache failure: %+v %v", firstSeen, err)
	}
	if len(opened.Library.entries) != 0 {
		t.Fatalf("failed reconcile replaced in-memory raw truth: %+v", opened.Library.entries)
	}
}

func TestPersistedCacheRemainsRawJsonWithoutOverlayFields(t *testing.T) {
	root := t.TempDir()
	p := paths(root)
	source := filepath.Join(root, "music")
	media := filepath.Join(source, "tone.mp3")
	copyFixture(t, media)
	absolute, _ := filepath.Abs(media)
	id := filesystem.TrackID(absolute)

	if err := filesystem.SaveOverlays(
		p.Overlays,
		map[string]library.Overlay{id: {Title: str("Edited")}},
	); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := opened.Library.Reconcile([]string{source}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(p.Cache)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if len(text) == 0 ||
		!strings.Contains(text, "VexStream Test Tone") ||
		strings.Contains(text, `"title": "Edited"`) {
		t.Fatalf("cache mixed raw and human projection state: %s", text)
	}
}
