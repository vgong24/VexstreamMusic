package librarymatch

import (
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/explore"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

func TestMatcherProjectsCurrentLibraryRelationship(t *testing.T) {
	matcher := New(func() []library.Track {
		return []library.Track{{
			ID: "track-a",
			Title: "Song",
			Artist: "Artist",
			ProviderID: "provider-a",
			SourceURL: "https://example/song",
			Duration: 180,
		}}
	})
	got := matcher.Match(explore.Candidate{
		ProviderID: "provider-a",
		Title: "Song",
		Channel: "Artist",
		URL: "https://example/song",
		Duration: 180,
	})
	if got.State != "IN_LIBRARY" || got.TrackRef != "track-a" || got.Relation != "SAME_SOURCE" {
		t.Fatalf("same-source projection drift: %+v", got)
	}
}

func TestMatcherDoesNotInventLibraryMembershipWithoutEvidence(t *testing.T) {
	matcher := New(func() []library.Track {
		return []library.Track{{ID: "track-a", Title: "Completely Different", Artist: "Other"}}
	})
	got := matcher.Match(explore.Candidate{Title: "Song", Channel: "Artist"})
	if got.State != "NOT_IN_LIBRARY" || got.TrackRef != "" {
		t.Fatalf("no-match projection drift: %+v", got)
	}
}
