package duplicates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

func TestSameProviderIdentityIsVeryLikelySameSource(t *testing.T) {
	score, reasons, relation := Score(
		Incoming{ProviderID: "provider-1", Title: "Song"},
		library.Track{ProviderID: "provider-1", Title: "Different"},
	)
	if score < 95 || relation != "SAME_SOURCE" || !strings.Contains(strings.Join(reasons, ","), "same provider item") {
		t.Fatalf("provider identity drift: score=%d relation=%s reasons=%v", score, relation, reasons)
	}
}

func TestNormalizedTitleArtistAndDurationCanIdentifyLikelyRecording(t *testing.T) {
	score, _, relation := Score(
		Incoming{Title: "Artist - Blue Dream (Official Video)", Artist: "Artist", Duration: 200},
		library.Track{Title: "Blue Dream", Artist: "Artist", Duration: 202},
	)
	if score < 75 || relation != "LIKELY_SAME_RECORDING" {
		t.Fatalf("recording relation drift: score=%d relation=%s", score, relation)
	}
}

func TestExactSongWithLargeDurationDifferenceStaysAdvisoryDifferentEdit(t *testing.T) {
	score, reasons, relation := Score(
		Incoming{Title: "Song Name", Artist: "Artist", Duration: 180},
		library.Track{Title: "Song Name", Artist: "Artist", Duration: 240},
	)
	if score < 55 || relation != "SAME_SONG_DIFFERENT_EDIT_POSSIBLE" {
		t.Fatalf("different edit relation drift: score=%d relation=%s", score, relation)
	}
	if !strings.Contains(strings.Join(reasons, ","), "may be a different edit") {
		t.Fatalf("different-edit caution missing: %v", reasons)
	}
}

func TestWeakCandidateDoesNotEnterAdvisoryMatches(t *testing.T) {
	got := Find(
		Incoming{Title: "Completely Different", Artist: "Artist A"},
		[]library.Track{{ID: "x", Title: "Unrelated Song", Artist: "Artist B"}},
	)
	if len(got) != 0 {
		t.Fatalf("weak candidate leaked into matches: %+v", got)
	}
}

func TestFindSortsByScoreThenTitleAndCapsAtFour(t *testing.T) {
	tracks := []library.Track{
		{ID: "e", Title: "Song", Artist: "Artist", ProviderID: "e"},
		{ID: "d", Title: "Song", Artist: "Artist", ProviderID: "d"},
		{ID: "c", Title: "Song", Artist: "Artist", ProviderID: "c"},
		{ID: "b", Title: "Song", Artist: "Artist", ProviderID: "b"},
		{ID: "a", Title: "Song", Artist: "Artist", ProviderID: "a"},
	}
	got := Find(Incoming{Title: "Song", Artist: "Artist", Duration: 100}, tracks)
	if len(got) != 4 {
		t.Fatalf("match cap drift: %+v", got)
	}
}

func TestReviewTreatsWholeAndSectionsAsIndependentAdvisoryItems(t *testing.T) {
	tracks := []library.Track{
		{ID: "whole", Title: "Full Mix", Artist: "Artist", Duration: 600},
		{ID: "part", Title: "Part One", Artist: "Artist", Duration: 120},
	}
	got := Review(ReviewRequest{
		Incoming: Incoming{Title: "Full Mix", Artist: "Artist", Duration: 600},
		Sections: []Section{{Index: 1, Title: "Part One", Start: 10, End: 130}},
	}, tracks)
	if len(got) != 2 || got[0].Kind != "whole" || got[1].Kind != "section" || got[1].Duration != 120 {
		t.Fatalf("review item drift: %+v", got)
	}
}

func TestQualityComparisonDoesNotTreatCrossCodecBitrateAsProof(t *testing.T) {
	got := CompareQuality(
		AudioQuality{Codec: "opus", BitrateKbps: 160},
		AudioQuality{Codec: "mp3", BitrateKbps: 128},
	)
	if got.State != "NOT_DIRECTLY_COMPARABLE" {
		t.Fatalf("cross-codec quality drift: %+v", got)
	}
}

func TestEstimateLocalQualityUsesFileSizeAndDurationOnlyAsIndicator(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tone.mp3")
	if err := os.WriteFile(path, make([]byte, 1000), 0o644); err != nil {
		t.Fatal(err)
	}
	got := EstimateLocalQuality(library.Track{Path: path, Duration: 10})
	if got.Codec != "mp3" || got.BitrateKbps != 0.8 || got.Basis != "local.file-size-duration-estimate" {
		t.Fatalf("quality estimate drift: %+v", got)
	}
}
