// Package compat holds executable rollback contracts between the V4 desktop and the protected V3.
//
// These tests intentionally mirror the accepted V3 JSON structs. Shared V4 stores must round-trip
// through those V3 shapes without semantic JSON loss. New V4-only fields belong in V4-only sidecars.
package compat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	desktopconfig "github.com/vgong24/VexstreamMusic/v2/desktop/config"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/filesystem"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/trash"
	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
)

type v3Config struct {
	Sources []string `json:"sources"`
	Ignored []string `json:"ignored"`
}

type v3Track struct {
	ID                string   `json:"id"`
	Path              string   `json:"path"`
	Filename          string   `json:"filename"`
	Title             string   `json:"title"`
	Artist            string   `json:"artist"`
	Album             string   `json:"album"`
	Genre             string   `json:"genre"`
	Year              string   `json:"year"`
	Duration          float64  `json:"duration"`
	Source            string   `json:"source"`
	ProviderID        string   `json:"providerId,omitempty"`
	SourceURL         string   `json:"sourceUrl,omitempty"`
	Channel           string   `json:"channel,omitempty"`
	ArtistBasis       string   `json:"artistBasis,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Categories        []string `json:"categories,omitempty"`
	Description       string   `json:"description,omitempty"`
	ThumbnailURL      string   `json:"thumbnailUrl,omitempty"`
	IndexedAt         string   `json:"indexedAt,omitempty"`
	AddedAt           string   `json:"addedAt"`
	SourceTitle       string   `json:"sourceTitle,omitempty"`
	SourceArtist      string   `json:"sourceArtist,omitempty"`
	SourceAlbum       string   `json:"sourceAlbum,omitempty"`
	SourceGenre       string   `json:"sourceGenre,omitempty"`
	OverlayUpdatedAt  string   `json:"overlayUpdatedAt,omitempty"`
	PlaybackStart     float64  `json:"playbackStart,omitempty"`
	PlaybackEnd       *float64 `json:"playbackEnd,omitempty"`
	PlaybackUpdatedAt string   `json:"playbackUpdatedAt,omitempty"`
}

type v3Overlay struct {
	Title     *string `json:"title,omitempty"`
	Artist    *string `json:"artist,omitempty"`
	Album     *string `json:"album,omitempty"`
	Genre     *string `json:"genre,omitempty"`
	UpdatedAt string  `json:"updatedAt"`
}

type v3PlaybackWindow struct {
	Start     float64  `json:"start,omitempty"`
	End       *float64 `json:"end,omitempty"`
	UpdatedAt string   `json:"updatedAt"`
}

type v3CacheEntry struct {
	Path         string  `json:"path"`
	Size         int64   `json:"size"`
	ModTime      int64   `json:"modTime"`
	CatalogStamp int64   `json:"catalogStamp"`
	Track        v3Track `json:"track"`
}

type v3LibraryCache struct {
	SchemaVersion string         `json:"schemaVersion"`
	GeneratedAt   string         `json:"generatedAt"`
	Entries       []v3CacheEntry `json:"entries"`
}

type v3Playlist struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	TrackIDs  []string `json:"trackIds"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

type v3TrashItem struct {
	ID           string  `json:"id"`
	Track        v3Track `json:"track"`
	OriginalPath string  `json:"originalPath"`
	TrashPath    string  `json:"trashPath"`
	TrashedAt    string  `json:"trashedAt"`
}

type v3DeletedHistoryItem struct {
	ID           string  `json:"id"`
	Track        v3Track `json:"track"`
	OriginalPath string  `json:"originalPath"`
	DeletedAt    string  `json:"deletedAt"`
}

type noopTrashCleaner struct{}

func (noopTrashCleaner) CleanPurged([]trash.Item) error { return nil }

func str(value string) *string   { return &value }
func num(value float64) *float64 { return &value }

func assertV3RoundTrip[T any](
	t *testing.T,
	path string,
	target *T,
) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var actual any
	if err := json.Unmarshal(raw, &actual); err != nil {
		t.Fatalf("shared file is not JSON: %v", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("V3 mirror could not read shared file: %v", err)
	}
	v3Bytes, err := json.Marshal(target)
	if err != nil {
		t.Fatal(err)
	}
	var projected any
	if err := json.Unmarshal(v3Bytes, &projected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, projected) {
		t.Fatalf(
			"V4 shared store contains state a V3 read/write cycle would lose:\nactual=%s\nv3=%s",
			string(raw),
			string(v3Bytes),
		)
	}
}

func representativeTrack(path, source string) library.Track {
	end := 42.5
	return library.Track{
		ID:                "track-a",
		Path:              path,
		Filename:          filepath.Base(path),
		Title:             "Song",
		Artist:            "Artist",
		Album:             "Album",
		Genre:             "Genre",
		Year:              "2026",
		Duration:          123.5,
		Source:            source,
		ProviderID:        "provider-a",
		SourceURL:         "https://example/song",
		Channel:           "Uploader",
		ArtistBasis:       "metadata.artist",
		Tags:              []string{"one", "two"},
		Categories:        []string{"Music"},
		Description:       "Description",
		ThumbnailURL:      "https://example/thumb",
		IndexedAt:         "2026-10-14T01:00:00Z",
		AddedAt:           "2026-10-14T01:00:01Z",
		SourceTitle:       "Source Song",
		SourceArtist:      "Source Artist",
		SourceAlbum:       "Source Album",
		SourceGenre:       "Source Genre",
		OverlayUpdatedAt:  "2026-10-14T01:00:02Z",
		PlaybackStart:     1.25,
		PlaybackEnd:       &end,
		PlaybackUpdatedAt: "2026-10-14T01:00:03Z",
	}
}

func TestV4SharedConfigRoundTripsThroughProtectedV3Shape(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "music")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "config.json")
	service, err := desktopconfig.Open(
		path,
		filepath.Join(root, "backups", "config.latest.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddSource(source); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Ignore(filepath.Join(root, "ignored")); err != nil {
		t.Fatal(err)
	}

	var mirror v3Config
	assertV3RoundTrip(t, path, &mirror)
	if len(mirror.Sources) != 1 || len(mirror.Ignored) != 1 {
		t.Fatalf("V3 config meaning drift: %+v", mirror)
	}
}

func TestV4SharedLibraryStateRoundTripsThroughProtectedV3Shapes(t *testing.T) {
	root := t.TempDir()

	firstSeenPath := filepath.Join(root, "track-first-seen.json")
	if err := filesystem.SaveFirstSeen(
		firstSeenPath,
		map[string]string{"track-a": "2026-10-14T01:00:01Z"},
	); err != nil {
		t.Fatal(err)
	}
	var firstSeen map[string]string
	assertV3RoundTrip(t, firstSeenPath, &firstSeen)

	overlayPath := filepath.Join(root, "library-overlays.json")
	if err := filesystem.SaveOverlays(
		overlayPath,
		map[string]library.Overlay{
			"track-a": {
				Title:     str("Edited"),
				Artist:    str("Edited Artist"),
				Album:     str("Edited Album"),
				Genre:     str("Edited Genre"),
				UpdatedAt: "2026-10-14T02:00:00Z",
			},
		},
	); err != nil {
		t.Fatal(err)
	}
	var overlays map[string]v3Overlay
	assertV3RoundTrip(t, overlayPath, &overlays)

	windowPath := filepath.Join(root, "playback-windows.json")
	if err := filesystem.SavePlaybackWindows(
		windowPath,
		map[string]library.PlaybackWindow{
			"track-a": {
				Start:     3.5,
				End:       num(9.25),
				UpdatedAt: "2026-10-14T03:00:00Z",
			},
		},
	); err != nil {
		t.Fatal(err)
	}
	var windows map[string]v3PlaybackWindow
	assertV3RoundTrip(t, windowPath, &windows)
}

func TestV4SharedLibraryCacheRemainsExactV3V2Shape(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "music")
	path := filepath.Join(source, "song.mp3")
	cachePath := filepath.Join(root, "library-index.json")
	track := representativeTrack(path, source)
	if err := filesystem.SaveCache(
		cachePath,
		map[string]library.CacheEntry{
			filesystem.CacheKey(path): {
				Path:         path,
				Size:         1234,
				ModTime:      55,
				CatalogStamp: 66,
				Track:        track,
			},
		},
	); err != nil {
		t.Fatal(err)
	}

	var mirror v3LibraryCache
	assertV3RoundTrip(t, cachePath, &mirror)
	if mirror.SchemaVersion != "vexstream.library-index/v2" ||
		len(mirror.Entries) != 1 ||
		mirror.Entries[0].Track.ID != "track-a" {
		t.Fatalf("V3 cache meaning drift: %+v", mirror)
	}
}

func TestV4SharedPlaylistsRemainExactV3ArrayShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playlists.json")
	service, err := playlists.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(
		"Road Trip",
		[]string{"track-a", "missing-track"},
	); err != nil {
		t.Fatal(err)
	}

	var mirror []v3Playlist
	assertV3RoundTrip(t, path, &mirror)
	if len(mirror) != 1 ||
		len(mirror[0].TrackIDs) != 2 ||
		mirror[0].TrackIDs[1] != "missing-track" {
		t.Fatalf("V3 playlist meaning drift: %+v", mirror)
	}
}

func TestV4SharedTrashAndDeletedHistoryRemainExactV3ArrayShapes(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "music")
	media := filepath.Join(source, "song.mp3")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(media, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}

	paths := trash.Paths{
		TrashState:     filepath.Join(root, "trash.json"),
		DeletedHistory: filepath.Join(root, "deleted-history.json"),
		FilesRoot:      filepath.Join(root, "trash", "files"),
		PurgeJournal:   filepath.Join(root, "v4", "trash-purge-journal.json"),
	}
	service, err := trash.Open(paths, noopTrashCleaner{})
	if err != nil {
		t.Fatal(err)
	}
	moved, err := service.Move([]trash.MoveInput{{
		Track: representativeTrack(media, source),
		Path:  media,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 1 {
		t.Fatalf("trash move drift: %+v", moved)
	}

	var trashMirror []v3TrashItem
	assertV3RoundTrip(t, paths.TrashState, &trashMirror)
	if len(trashMirror) != 1 || trashMirror[0].Track.ID != "track-a" {
		t.Fatalf("V3 trash meaning drift: %+v", trashMirror)
	}

	if _, err := service.Purge([]string{moved[0].ID}, true); err != nil {
		t.Fatal(err)
	}
	var emptyTrash []v3TrashItem
	assertV3RoundTrip(t, paths.TrashState, &emptyTrash)
	if len(emptyTrash) != 0 {
		t.Fatalf("purged V3 trash not empty: %+v", emptyTrash)
	}

	var history []v3DeletedHistoryItem
	assertV3RoundTrip(t, paths.DeletedHistory, &history)
	if len(history) != 1 ||
		history[0].Track.ID != "track-a" ||
		history[0].OriginalPath != media {
		t.Fatalf("V3 deleted history meaning drift: %+v", history)
	}
}
