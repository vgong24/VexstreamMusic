package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

func fixturePath(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "tests", "fixtures", "vexstream-test-tone.mp3")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture unavailable: %v", err)
	}
	return path
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeCatalogRow(t *testing.T, root, filename, indexedAt string) {
	t.Helper()
	path := filepath.Join(root, "_vexmedia", "library.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	row := map[string]any{
		"filename":   filename,
		"providerId": "provider-1",
		"indexedAt":  indexedAt,
		"metadata": map[string]any{
			"sourceUrl":    "https://example.test/watch/1",
			"channel":      "VexStream QA",
			"duration":     1.25,
			"tags":         []string{"test", "tone"},
			"categories":   []string{"Music"},
			"thumbnailUrl": "https://example.test/thumb.jpg",
		},
	}
	data, _ := json.Marshal(row)
	if err := os.WriteFile(path, append(data, '
'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadTrackPreservesV3ID3AndCatalogProvenance(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "tone.mp3")
	copyFile(t, fixturePath(t), media)
	writeCatalogRow(t, root, "tone.mp3", "2026-09-29T00:00:00Z")

	track := ReadTrack(media, root)
	if track.Title != "VexStream Test Tone" || track.Artist != "VexStream QA" {
		t.Fatalf("ID3 metadata drift: %+v", track)
	}
	if track.ProviderID != "provider-1" || track.SourceURL == "" || track.Duration != 1.25 {
		t.Fatalf("catalog provenance missing: %+v", track)
	}
	if track.ArtistBasis != "legacy.youtube.channel.unverified" {
		t.Fatalf("channel provenance inference drift: %q", track.ArtistBasis)
	}
	if !reflect.DeepEqual(track.Tags, []string{"test", "tone"}) || !reflect.DeepEqual(track.Categories, []string{"Music"}) {
		t.Fatalf("catalog lists drift: %+v", track)
	}
}

func TestScanReusesUnchangedRawCacheEntry(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "tone.mp3")
	copyFile(t, fixturePath(t), media)
	absolute, _ := filepath.Abs(media)
	stat, err := os.Stat(absolute)
	if err != nil {
		t.Fatal(err)
	}
	id := TrackID(absolute)
	previous := map[string]library.CacheEntry{
		CacheKey(absolute): {
			Path: absolute, Size: stat.Size(), ModTime: stat.ModTime().UnixNano(), CatalogStamp: 0,
			Track: library.Track{ID: id, Path: absolute, Filename: "tone.mp3", Source: root, Title: "Cached raw title"},
		},
	}

	got, err := Scan(ScanInput{Sources: []string{root}, Previous: previous, FirstSeen: map[string]string{id: "first-seen"}})
	if err != nil {
		t.Fatal(err)
	}
	entry := got.Entries[CacheKey(absolute)]
	if entry.Track.Title != "Cached raw title" {
		t.Fatalf("unchanged cache was reparsed: %+v", entry.Track)
	}
	if entry.Track.AddedAt != "first-seen" {
		t.Fatalf("existing AddedAt not preserved: %+v", entry.Track)
	}
}

func TestScanInvalidatesCacheWhenFileFingerprintChanges(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "tone.mp3")
	copyFile(t, fixturePath(t), media)
	absolute, _ := filepath.Abs(media)
	stat, err := os.Stat(absolute)
	if err != nil {
		t.Fatal(err)
	}
	previous := map[string]library.CacheEntry{
		CacheKey(absolute): {
			Path: absolute, Size: stat.Size() + 1, ModTime: stat.ModTime().UnixNano(),
			Track: library.Track{ID: TrackID(absolute), Source: root, Title: "Stale cache"},
		},
	}

	got, err := Scan(ScanInput{Sources: []string{root}, Previous: previous})
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[CacheKey(absolute)].Track.Title != "VexStream Test Tone" {
		t.Fatalf("changed file did not reparse source metadata: %+v", got.Entries[CacheKey(absolute)].Track)
	}
}

func TestScanUsesIndexedAtForFirstSeenAndSkipsIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "tone.mp3")
	copyFile(t, fixturePath(t), media)
	copyFile(t, fixturePath(t), filepath.Join(root, ".git", "hidden.mp3"))
	writeCatalogRow(t, root, "tone.mp3", "2026-09-29T00:00:00Z")

	got, err := Scan(ScanInput{Sources: []string{root}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("ignored directory leaked into scan: %+v", got.Entries)
	}
	for _, entry := range got.Entries {
		if entry.Track.AddedAt != "2026-09-29T00:00:00Z" {
			t.Fatalf("IndexedAt did not seed AddedAt: %+v", entry.Track)
		}
		if got.FirstSeen[entry.Track.ID] != entry.Track.AddedAt {
			t.Fatalf("firstSeen map not retained: %+v", got.FirstSeen)
		}
	}
}

func TestCacheV2RoundTripsAndLegacyV1IsDiscarded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library-cache.json")
	entries := map[string]library.CacheEntry{
		"a": {Path: "/music/a.mp3", Track: library.Track{ID: "a", Title: "A"}},
	}
	if err := SaveCache(path, entries); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadCache(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DiscardedLegacy || loaded.Entries[CacheKey("/music/a.mp3")].Track.Title != "A" {
		t.Fatalf("v2 cache round-trip drift: %+v", loaded)
	}

	legacy := []byte(`{"schemaVersion":"vexstream.library-index/v1","entries":[{"path":"/old.mp3"}]}`)
	if err := os.WriteFile(path, legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err = LoadCache(path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.DiscardedLegacy || len(loaded.Entries) != 0 {
		t.Fatalf("legacy v1 cache was not deliberately discarded: %+v", loaded)
	}
}

func TestCacheRejectsUnknownSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library-cache.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":"vexstream.library-index/v999"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCache(path); err == nil {
		t.Fatal("unsupported cache schema was silently accepted")
	}
}

func TestScanInvalidatesRawCacheWhenCatalogStampChanges(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "tone.mp3")
	copyFile(t, fixturePath(t), media)
	writeCatalogRow(t, root, "tone.mp3", "2026-09-29T00:00:00Z")

	first, err := Scan(ScanInput{Sources: []string{root}})
	if err != nil {
		t.Fatal(err)
	}
	absolute, _ := filepath.Abs(media)
	key := CacheKey(absolute)
	cached := first.Entries[key]
	cached.Track.ProviderID = "stale-provider"
	previous := map[string]library.CacheEntry{key: cached}

	catalogPath := filepath.Join(root, "_vexmedia", "library.jsonl")
	writeCatalogRow(t, root, "tone.mp3", "2026-09-30T00:00:00Z")
	stat, err := os.Stat(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	future := stat.ModTime().Add(2 * time.Second)
	if err := os.Chtimes(catalogPath, future, future); err != nil {
		t.Fatal(err)
	}

	second, err := Scan(ScanInput{Sources: []string{root}, Previous: previous, FirstSeen: first.FirstSeen})
	if err != nil {
		t.Fatal(err)
	}
	if second.Entries[key].Track.ProviderID != "provider-1" {
		t.Fatalf("catalog change reused stale raw cache: %+v", second.Entries[key].Track)
	}
	if second.Entries[key].Track.AddedAt != first.Entries[key].Track.AddedAt {
		t.Fatalf("catalog refresh changed AddedAt identity: first=%q second=%q", first.Entries[key].Track.AddedAt, second.Entries[key].Track.AddedAt)
	}
}
