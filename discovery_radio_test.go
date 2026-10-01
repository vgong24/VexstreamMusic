package main

import "testing"

func TestDiscoveryQueriesAreMultiAxisAndDeterministic(t *testing.T) {
	seed := DiscoverySeed{Title: "03 Charlie", Artist: "Red Hot Chili Peppers", Album: "Stadium Arcadium", Genre: "Alternative", Year: "2006"}
	got := discoveryQueries(seed)
	want := []DiscoveryQuery{
		{Axis: "CLOSER", Label: "Closer", Query: "Red Hot Chili Peppers Stadium Arcadium"},
		{Axis: "NEIGHBORHOOD", Label: "Neighborhood", Query: "Alternative music"},
		{Axis: "ERA", Label: "Same era", Query: "2006 Alternative music"},
		{Axis: "VERSIONS", Label: "Versions", Query: "03 Charlie cover remix live"},
	}
	if len(got) != len(want) {
		t.Fatalf("queries=%+v want=%+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("queries=%+v want=%+v", got, want)
		}
	}
}

func TestDiscoveryQueriesDegradeWithoutGenreOrYear(t *testing.T) {
	seed := DiscoverySeed{Title: "Seed Song", Artist: "Seed Artist"}
	got := discoveryQueries(seed)
	if len(got) != 2 {
		t.Fatalf("queries=%+v want 2", got)
	}
	if got[0].Axis != "CLOSER" || got[1].Axis != "VERSIONS" {
		t.Fatalf("queries=%+v", got)
	}
}

func TestDiscoverySelectionRoundRobinsAxesAndCapsCreators(t *testing.T) {
	pools := [][]DiscoveryCandidate{
		{
			{ProviderID: "a1", Title: "A1", Channel: "Same Artist", Axis: "CLOSER"},
			{ProviderID: "a2", Title: "A2", Channel: "Same Artist", Axis: "CLOSER"},
			{ProviderID: "a3", Title: "A3", Channel: "Same Artist", Axis: "CLOSER"},
		},
		{
			{ProviderID: "b1", Title: "B1", Channel: "Neighbor One", Axis: "NEIGHBORHOOD"},
			{ProviderID: "b2", Title: "B2", Channel: "Neighbor Two", Axis: "NEIGHBORHOOD"},
		},
		{
			{ProviderID: "c1", Title: "C1", Channel: "Era Artist", Axis: "ERA"},
			{ProviderID: "c2", Title: "C2", Channel: "Era Artist", Axis: "ERA"},
		},
	}
	got := selectDiscoveryCandidates(pools, 7, 2)
	if len(got) != 6 {
		t.Fatalf("selected=%+v want 6 unique under creator cap", got)
	}
	wantAxes := []string{"CLOSER", "NEIGHBORHOOD", "ERA", "CLOSER", "NEIGHBORHOOD", "ERA"}
	for i, axis := range wantAxes {
		if got[i].Axis != axis {
			t.Fatalf("selected axis[%d]=%s want=%s all=%+v", i, got[i].Axis, axis, got)
		}
	}
	countSame := 0
	for _, candidate := range got {
		if candidate.Channel == "Same Artist" {
			countSame++
		}
	}
	if countSame != 2 {
		t.Fatalf("same creator count=%d want 2", countSame)
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
