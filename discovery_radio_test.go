package main

import "testing"

func TestDiscoveryQueriesAreDeterministicAndBounded(t *testing.T) {
	seed := DiscoverySeed{Title: "03 Charlie", Artist: "Red Hot Chili Peppers", Album: "Stadium Arcadium", Genre: "Alternative"}
	got := discoveryQueries(seed)
	want := []string{"Red Hot Chili Peppers Alternative", "Red Hot Chili Peppers Stadium Arcadium"}
	if len(got) != len(want) {
		t.Fatalf("queries=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("queries=%v want=%v", got, want)
		}
	}
}
func TestDiscoveryCandidateKeyPrefersStableProviderIdentity(t *testing.T) {
	a := DiscoveryCandidate{ProviderID: "ABC123", Title: "One", Channel: "Artist"}
	b := DiscoveryCandidate{ProviderID: "abc123", Title: "Different", Channel: "Other"}
	if discoveryCandidateKey(a) != discoveryCandidateKey(b) {
		t.Fatalf("provider identity should dedupe")
	}
}
func TestDiscoveryLibraryMatchDistinguishesExactAndAbsent(t *testing.T) {
	a := &App{tracks: []Track{{ID: "local", Title: "Existing Song", Artist: "Artist", ProviderID: "provider-1", SourceURL: "https://www.youtube.com/watch?v=provider-1", Duration: 180}}}
	exact := a.discoveryLibraryMatch(DiscoveryCandidate{ProviderID: "provider-1", URL: "https://www.youtube.com/watch?v=provider-1", Title: "Existing Song", Channel: "Artist", Duration: 180})
	if exact.State != "IN_LIBRARY" || exact.TrackID != "local" {
		t.Fatalf("exact=%+v", exact)
	}
	absent := a.discoveryLibraryMatch(DiscoveryCandidate{ProviderID: "other", URL: "https://www.youtube.com/watch?v=other", Title: "Completely Different", Channel: "Someone Else", Duration: 240})
	if absent.State != "NOT_IN_LIBRARY" {
		t.Fatalf("absent=%+v", absent)
	}
}
