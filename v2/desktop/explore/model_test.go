package explore

import (
	"reflect"
	"testing"
)

func TestQueriesPreserveV3MultiAxisFormation(t *testing.T) {
	got := Queries(Seed{
		Title:  "Song",
		Artist: "Artist",
		Album:  "Album",
		Genre:  "Dream Pop",
		Year:   "2006",
	})
	want := []Query{
		{Axis: "CLOSER", Label: "Closer", Query: "Artist Album"},
		{Axis: "NEIGHBORHOOD", Label: "Neighborhood", Query: "Dream Pop music"},
		{Axis: "ERA", Label: "Same era", Query: "2006 Dream Pop music"},
		{Axis: "VERSIONS", Label: "Versions", Query: "Song cover remix live"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("query formation drift:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestQueriesUseChannelAsCreatorFallback(t *testing.T) {
	got := Queries(Seed{Title: "Song", Channel: "Uploader", Genre: "Rock"})
	if len(got) < 1 || got[0].Query != "Uploader Rock" {
		t.Fatalf("channel fallback drift: %+v", got)
	}
}

func TestSelectCandidatesRoundRobinsDedupesAndCapsCreators(t *testing.T) {
	pools := [][]Candidate{
		{
			{ProviderID: "a1", Title: "A1", Channel: "Creator A"},
			{ProviderID: "a2", Title: "A2", Channel: "Creator A"},
			{ProviderID: "a3", Title: "A3", Channel: "Creator A"},
		},
		{
			{ProviderID: "b1", Title: "B1", Channel: "Creator B"},
			{ProviderID: "a1", Title: "duplicate", Channel: "Other"},
			{ProviderID: "c1", Title: "C1", Channel: "Creator C"},
		},
	}
	got := SelectCandidates(pools, 5, 2)
	ids := make([]string, 0, len(got))
	for _, candidate := range got {
		ids = append(ids, candidate.ProviderID)
	}
	want := []string{"a1", "b1", "a2", "c1"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("selection drift: got=%v want=%v", ids, want)
	}
}

func TestSelectCandidatesFallsBackToURLAndNormalizedTextIdentity(t *testing.T) {
	pools := [][]Candidate{
		{
			{URL: "https://example/one", Title: "One", Channel: "Creator"},
			{Title: "Song (Official Video)", Channel: "Creator"},
		},
		{
			{URL: "HTTPS://EXAMPLE/ONE", Title: "duplicate URL", Channel: "Other"},
			{Title: "Song", Channel: "Creator"},
			{Title: "Other", Channel: "Creator B"},
		},
	}
	got := SelectCandidates(pools, 10, 10)
	if len(got) != 3 {
		t.Fatalf("fallback identity did not dedupe candidates: %+v", got)
	}
}

func TestSameSeedRecognizesProviderURLAndNormalizedTitleCreator(t *testing.T) {
	seed := Seed{ProviderID: "ABC", URL: "https://example/one", Title: "Song", Artist: "Artist"}
	if !SameSeed(seed, Candidate{ProviderID: "abc"}) {
		t.Fatal("provider identity match missed")
	}
	if !SameSeed(seed, Candidate{URL: "HTTPS://EXAMPLE/ONE"}) {
		t.Fatal("URL identity match missed")
	}
	if !SameSeed(seed, Candidate{Title: "Song (Official Video)", Channel: "Artist"}) {
		t.Fatal("normalized title/creator match missed")
	}
	if SameSeed(seed, Candidate{Title: "Another Song", Channel: "Artist"}) {
		t.Fatal("different candidate treated as seed")
	}
}

func TestSelectionLimitAndCreatorCapDefaultsRemainBounded(t *testing.T) {
	if got := SelectCandidates([][]Candidate{{{ProviderID: "a"}}}, 0, 1); got != nil {
		t.Fatalf("zero limit should return nil: %+v", got)
	}
	got := SelectCandidates([][]Candidate{{
		{ProviderID: "a", Channel: "same"},
		{ProviderID: "b", Channel: "same"},
	}}, 2, 0)
	if len(got) != 2 {
		t.Fatalf("default creator cap should use limit: %+v", got)
	}
}
