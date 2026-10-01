package main

import (
	"strings"
	"testing"
)

func TestDuplicateDetectionHandlesArtistPrefixAndDifferentVideoEdit(t *testing.T) {
	existing := Track{Title: "Ain't It Fun", Artist: "Paramore", Duration: 297}
	score, reasons, relation := duplicateScore(
		"Paramore: Ain't It Fun [OFFICIAL VIDEO]",
		"",
		"Paramore",
		"different-youtube-id",
		"https://www.youtube.com/watch?v=different",
		228,
		existing,
	)
	if score < 55 {
		t.Fatalf("expected duplicate advisory score >=55, got %d (%v)", score, reasons)
	}
	if relation != "SAME_SONG_DIFFERENT_EDIT_POSSIBLE" {
		t.Fatalf("expected different-edit relation, got %q", relation)
	}
	joined := strings.Join(reasons, "|")
	if !strings.Contains(joined, "same normalized song title") || !strings.Contains(joined, "source creator matches existing artist") {
		t.Fatalf("expected title+creator evidence, got %v", reasons)
	}
}

func TestDuplicateDetectionIgnoresApostropheStyle(t *testing.T) {
	existing := Track{Title: "Ain't It Fun", Artist: "Paramore", Duration: 297}
	_, _, relation := duplicateScore("Aint It Fun", "Paramore", "", "", "", 297, existing)
	if relation != "LIKELY_SAME_RECORDING" {
		t.Fatalf("apostrophe-insensitive title should remain same recording candidate, got %q", relation)
	}
}

func TestQualityComparisonDoesNotRankDifferentCodecsByBitrate(t *testing.T) {
	q := compareAudioQuality(
		AudioQuality{Codec: "opus", BitrateKbps: 130},
		AudioQuality{Codec: "mp3", BitrateKbps: 192},
	)
	if q.State != "NOT_DIRECTLY_COMPARABLE" {
		t.Fatalf("different codecs must not get a quality winner, got %#v", q)
	}
}

func TestExistingArtistSuggestionUsesExplicitTitleCredit(t *testing.T) {
	a := &App{tracks: []Track{
		{Title: "Work It", Artist: "Missy Elliott"},
		{Title: "Get Ur Freak On", Artist: "Missy Elliott"},
		{Title: "Roundabout", Artist: "Yes"},
	}}
	s := a.inferExistingArtistSuggestion("Missy Elliott - Lose Control (feat. Ciara & Fat Man Scoop) [Official Music Video]", "Missy Elliott")
	if s == nil {
		t.Fatal("expected an existing-library artist suggestion")
	}
	if s.Artist != "Missy Elliott" || s.Confidence != "HIGH" || s.MatchingTrackCount != 2 || !s.ChannelCorroborated {
		t.Fatalf("unexpected suggestion: %#v", s)
	}
}

func TestExistingArtistSuggestionDoesNotUseLooseSubstring(t *testing.T) {
	a := &App{tracks: []Track{{Title: "Roundabout", Artist: "Yes"}}}
	if s := a.inferExistingArtistSuggestion("Yes We Can Make Music", "Some Channel"); s != nil {
		t.Fatalf("loose word prefix must not become an artist suggestion: %#v", s)
	}
	if s := a.inferExistingArtistSuggestion("Yes - Roundabout", "Some Channel"); s == nil || s.Artist != "Yes" {
		t.Fatalf("explicit artist-credit boundary should match existing artist, got %#v", s)
	}
}
