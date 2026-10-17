package librarymatch

import (
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

func TestSeedResolverProjectsCurrentLibraryMeaningWithoutPathCustody(t *testing.T) {
	resolver := NewSeedResolver(func() library.Projection {
		return library.Projection{
			Tracks: []library.Track{{
				ID:         "track-a",
				Path:       "/private/music/song.mp3",
				Title:      "Song",
				Artist:     "Artist",
				Album:      "Album",
				Genre:      "Dream Pop",
				Year:       "2006",
				ProviderID: "provider-a",
				SourceURL:  "https://example/song",
				Channel:    "Uploader",
				Duration:   180,
			}},
			Paths: map[string]string{
				"track-a": "/private/music/song.mp3",
			},
		}
	})

	seed, ok := resolver.ResolveTrackSeed(" track-a ")
	if !ok {
		t.Fatal("current Library track was not resolved")
	}
	if seed.TrackRef != "track-a" ||
		seed.Title != "Song" ||
		seed.Artist != "Artist" ||
		seed.ProviderID != "provider-a" ||
		seed.URL != "https://example/song" {
		t.Fatalf("Explore seed drift: %+v", seed)
	}
	if seed.Title == "/private/music/song.mp3" ||
		seed.URL == "/private/music/song.mp3" {
		t.Fatalf("filesystem custody leaked into Explore seed: %+v", seed)
	}
}

func TestSeedResolverUsesSourceArtistOnlyWhenCurrentArtistIsBlank(t *testing.T) {
	resolver := NewSeedResolver(func() library.Projection {
		return library.Projection{
			Tracks: []library.Track{{
				ID:           "track-a",
				Title:        "Song",
				Artist:       "",
				SourceArtist: "Source Artist",
			}},
		}
	})
	seed, ok := resolver.ResolveTrackSeed("track-a")
	if !ok || seed.Artist != "Source Artist" {
		t.Fatalf("source-artist fallback drift: %+v ok=%v", seed, ok)
	}
}

func TestSeedResolverRejectsStaleOrBlankTrackRefs(t *testing.T) {
	resolver := NewSeedResolver(func() library.Projection {
		return library.Projection{}
	})
	for _, ref := range []string{"", " ", "missing"} {
		if _, ok := resolver.ResolveTrackSeed(ref); ok {
			t.Fatalf("stale Explore seed accepted: %q", ref)
		}
	}
}
