package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library/filesystem"
)

func TestUpdateOverlaysPersistsThenProjectsWithoutMutatingRawCache(t *testing.T) {
	root := t.TempDir()
	p := paths(filepath.Join(root, "state"))
	source := filepath.Join(root, "music")
	media := filepath.Join(source, "tone.mp3")
	copyFixture(t, media)
	absolute, _ := filepath.Abs(media)
	id := filesystem.TrackID(absolute)

	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := opened.Library.Reconcile([]string{source}); err != nil {
		t.Fatal(err)
	}

	title := "Human Title"
	artist := "Human Artist"
	updated, err := opened.Library.UpdateOverlays(
		[]string{id, id, " "},
		OverlayPatch{Title: &title, Artist: &artist},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated != 1 {
		t.Fatalf("overlay target normalization drift: %d", updated)
	}
	projection := opened.Library.Snapshot([]string{source}, false)
	if len(projection.Tracks) != 1 ||
		projection.Tracks[0].Title != "Human Title" ||
		projection.Tracks[0].Artist != "Human Artist" ||
		projection.Tracks[0].SourceTitle != "VexStream Test Tone" {
		t.Fatalf("overlay projection drift: %+v", projection.Tracks)
	}

	cache, err := filesystem.LoadCache(p.Cache)
	if err != nil {
		t.Fatal(err)
	}
	entry := cache.Entries[filesystem.CacheKey(absolute)]
	if entry.Track.Title != "VexStream Test Tone" ||
		entry.Track.Artist != "VexStream QA" {
		t.Fatalf("raw cache absorbed overlay: %+v", entry.Track)
	}

	overlays, err := filesystem.LoadOverlays(p.Overlays)
	if err != nil || overlays[id].Title == nil ||
		*overlays[id].Title != "Human Title" {
		t.Fatalf("overlay persistence drift: %+v err=%v", overlays, err)
	}
}

func TestOverlayRestoreRemovesOnlyHumanProjection(t *testing.T) {
	root := t.TempDir()
	p := paths(filepath.Join(root, "state"))
	source := filepath.Join(root, "music")
	media := filepath.Join(source, "tone.mp3")
	copyFixture(t, media)
	absolute, _ := filepath.Abs(media)
	id := filesystem.TrackID(absolute)

	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := opened.Library.Reconcile([]string{source}); err != nil {
		t.Fatal(err)
	}
	title := "Edited"
	if _, err := opened.Library.UpdateOverlays(
		[]string{id},
		OverlayPatch{Title: &title},
		false,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := opened.Library.UpdateOverlays(
		[]string{id},
		OverlayPatch{},
		true,
	); err != nil {
		t.Fatal(err)
	}
	projection := opened.Library.Snapshot([]string{source}, false)
	if projection.Tracks[0].Title != "VexStream Test Tone" ||
		projection.Tracks[0].OverlayUpdatedAt != "" {
		t.Fatalf("overlay restore drift: %+v", projection.Tracks[0])
	}
	overlays, err := filesystem.LoadOverlays(p.Overlays)
	if err != nil || len(overlays) != 0 {
		t.Fatalf("restored overlay remains durable: %+v err=%v", overlays, err)
	}
}

func TestUpdatePlaybackWindowPersistsAndRestoreReturnsFullTrack(t *testing.T) {
	root := t.TempDir()
	p := paths(filepath.Join(root, "state"))
	source := filepath.Join(root, "music")
	media := filepath.Join(source, "tone.mp3")
	copyFixture(t, media)

	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := opened.Library.Reconcile([]string{source})
	if err != nil {
		t.Fatal(err)
	}
	id := projection.Tracks[0].ID
	end := 0.75

	window, present, err := opened.Library.UpdatePlaybackWindow(
		id,
		0.25,
		&end,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !present || window.Start != 0.25 ||
		window.End == nil || *window.End != 0.75 ||
		window.UpdatedAt == "" {
		t.Fatalf("playback window drift: %+v present=%v", window, present)
	}
	projected := opened.Library.Snapshot([]string{source}, false).Tracks[0]
	if projected.PlaybackStart != 0.25 ||
		projected.PlaybackEnd == nil ||
		*projected.PlaybackEnd != 0.75 {
		t.Fatalf("playback projection drift: %+v", projected)
	}

	if _, present, err := opened.Library.UpdatePlaybackWindow(
		id,
		0,
		nil,
		true,
	); err != nil || present {
		t.Fatalf("playback restore drift: present=%v err=%v", present, err)
	}
	projected = opened.Library.Snapshot([]string{source}, false).Tracks[0]
	if projected.PlaybackStart != 0 ||
		projected.PlaybackEnd != nil ||
		projected.PlaybackUpdatedAt != "" {
		t.Fatalf("full-track restore drift: %+v", projected)
	}
}

func TestOverlayMutationRejectsEmptyIntentBeforePersistence(t *testing.T) {
	root := t.TempDir()
	p := paths(root)
	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := opened.Library.UpdateOverlays(
		[]string{"track"},
		OverlayPatch{},
		false,
	); err == nil {
		t.Fatal("empty overlay mutation accepted")
	}
	if _, err := os.Stat(p.Overlays); !os.IsNotExist(err) {
		t.Fatalf("overlay file created for rejected mutation: %v", err)
	}
}

func TestForgetTrackStateIsIdempotentAcrossV3ReadableStores(t *testing.T) {
	root := t.TempDir()
	p := paths(filepath.Join(root, "state"))
	source := filepath.Join(root, "music")
	media := filepath.Join(source, "tone.mp3")
	copyFixture(t, media)
	absolute, _ := filepath.Abs(media)
	id := filesystem.TrackID(absolute)

	opened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := opened.Library.Reconcile([]string{source}); err != nil {
		t.Fatal(err)
	}
	title := "Edited"
	if _, err := opened.Library.UpdateOverlays(
		[]string{id},
		OverlayPatch{Title: &title},
		false,
	); err != nil {
		t.Fatal(err)
	}
	end := 0.75
	if _, _, err := opened.Library.UpdatePlaybackWindow(
		id,
		0.25,
		&end,
		false,
	); err != nil {
		t.Fatal(err)
	}

	if err := opened.Library.ForgetTrackState([]string{id, id}); err != nil {
		t.Fatal(err)
	}
	if err := opened.Library.ForgetTrackState([]string{id}); err != nil {
		t.Fatalf("idempotent cleanup drift: %v", err)
	}

	firstSeen, err := filesystem.LoadFirstSeen(p.FirstSeen)
	if err != nil {
		t.Fatal(err)
	}
	overlays, err := filesystem.LoadOverlays(p.Overlays)
	if err != nil {
		t.Fatal(err)
	}
	windows, err := filesystem.LoadPlaybackWindows(p.PlaybackWindows)
	if err != nil {
		t.Fatal(err)
	}
	if firstSeen[id] != "" || overlays[id].UpdatedAt != "" ||
		windows[id].UpdatedAt != "" {
		t.Fatalf(
			"purged projection state remains: first=%+v overlay=%+v window=%+v",
			firstSeen,
			overlays,
			windows,
		)
	}
}
