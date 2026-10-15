package library

import (
	"reflect"
	"testing"
)

func str(v string) *string      { return &v }
func number(v float64) *float64 { return &v }

func TestProjectTrackPreservesRawMetadataAndLayersOverlay(t *testing.T) {
	raw := Track{ID: "a", Title: "Source title", Artist: "Source artist", Album: "Source album", Genre: "Source genre"}
	overlay := Overlay{Title: str("Display title"), Artist: str("Display artist"), UpdatedAt: "overlay-1"}
	got := ProjectTrack(raw, &overlay, nil)
	if got.Title != "Display title" || got.Artist != "Display artist" {
		t.Fatalf("overlay not projected: %+v", got)
	}
	if got.SourceTitle != "Source title" || got.SourceArtist != "Source artist" || got.SourceAlbum != "Source album" || got.SourceGenre != "Source genre" {
		t.Fatalf("source metadata not preserved: %+v", got)
	}
	if raw.SourceTitle != "" || raw.Title != "Source title" {
		t.Fatalf("raw track mutated: %+v", raw)
	}
}

func TestProjectTrackTreatsYoutubeChannelAsProvenanceUntilHumanArtistOverlay(t *testing.T) {
	raw := Track{ID: "a", Artist: "Uploader Channel", ArtistBasis: "youtube.channel"}
	without := ProjectTrack(raw, nil, nil)
	if without.Artist != "" || without.SourceArtist != "Uploader Channel" {
		t.Fatalf("untrusted artist projection drift: %+v", without)
	}
	with := ProjectTrack(raw, &Overlay{Artist: str("Real Artist")}, nil)
	if with.Artist != "Real Artist" || with.SourceArtist != "Uploader Channel" {
		t.Fatalf("human overlay did not restore artist separately: %+v", with)
	}
}

func TestPlaybackWindowIsProjectionNotRawTrackIdentity(t *testing.T) {
	raw := Track{ID: "a", PlaybackStart: 99, PlaybackEnd: number(100), PlaybackUpdatedAt: "stale"}
	window := PlaybackWindow{Start: 12.5, End: number(42), UpdatedAt: "window-2"}
	got := ProjectTrack(raw, nil, &window)
	if got.PlaybackStart != 12.5 || got.PlaybackEnd == nil || *got.PlaybackEnd != 42 || got.PlaybackUpdatedAt != "window-2" {
		t.Fatalf("playback window projection drift: %+v", got)
	}
	if raw.PlaybackStart != 99 || raw.PlaybackEnd == nil || *raw.PlaybackEnd != 100 {
		t.Fatalf("raw track mutated: %+v", raw)
	}
}

func TestProjectCatalogFiltersSourcesSortsDisplayMetadataAndKeepsPrivatePaths(t *testing.T) {
	entries := map[string]CacheEntry{
		"z": {Path: "/music/z.mp3", Track: Track{ID: "z", Source: "/music", Artist: "zeta", Album: "B", Title: "Second"}},
		"a": {Path: "/music/a.mp3", Track: Track{ID: "a", Source: "/music", Artist: "alpha", Album: "A", Title: "First"}},
		"x": {Path: "/other/x.mp3", Track: Track{ID: "x", Source: "/other", Artist: "other", Title: "Hidden"}},
	}
	overlays := map[string]Overlay{"z": {Artist: str("Beta")}}
	got := ProjectCatalog(entries, []string{"/MUSIC"}, overlays, nil, false)
	refs := []string{got.Tracks[0].ID, got.Tracks[1].ID}
	if !reflect.DeepEqual(refs, []string{"a", "z"}) {
		t.Fatalf("catalog order/filter drift: %v", refs)
	}
	if got.Paths["a"] != "/music/a.mp3" || got.Paths["z"] != "/music/z.mp3" {
		t.Fatalf("path custody missing: %+v", got.Paths)
	}
	if _, ok := got.Paths["x"]; ok {
		t.Fatalf("inactive source leaked into projection: %+v", got.Paths)
	}
}

func TestUseAllSupportsRecoveryProjectionWithoutChangingSourceTruth(t *testing.T) {
	entries := map[string]CacheEntry{"x": {Path: "/lost/x.mp3", Track: Track{ID: "x", Source: "/lost", Title: "Recovered"}}}
	got := ProjectCatalog(entries, nil, nil, nil, true)
	if len(got.Tracks) != 1 || got.Tracks[0].ID != "x" {
		t.Fatalf("recovery projection lost cached track: %+v", got)
	}
}

func TestProjectCatalogNormalizesSourcePathsBeforeComparison(t *testing.T) {
	entries := map[string]CacheEntry{
		"a": {Path: "/music/a.mp3", Track: Track{ID: "a", Source: "/music", Title: "Track"}},
	}
	got := ProjectCatalog(entries, []string{"/music/../music"}, nil, nil, false)
	if len(got.Tracks) != 1 || got.Tracks[0].ID != "a" {
		t.Fatalf("cleaned source path did not match: %+v", got)
	}
}
