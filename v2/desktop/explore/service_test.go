package explore

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type providerFixture struct {
	rows map[string][]Candidate
	fail string
	seen []string
}

func (p *providerFixture) Search(_ context.Context, query string, limit int) ([]Candidate, error) {
	p.seen = append(p.seen, query)
	if query == p.fail {
		return nil, errors.New("provider failed")
	}
	rows := append([]Candidate(nil), p.rows[query]...)
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

type matcherFixture struct {
	seen []string
}

func (m *matcherFixture) Match(candidate Candidate) LibraryMatch {
	m.seen = append(m.seen, candidate.ProviderID)
	if candidate.ProviderID == "local" {
		return LibraryMatch{
			State: "IN_LIBRARY",
			TrackRef: "track-local",
			Relation: "SAME_SOURCE",
			Score: 100,
		}
	}
	return LibraryMatch{State: "NOT_IN_LIBRARY"}
}

func TestExploreBuildsV3ShapedFrameAndBalancedCandidates(t *testing.T) {
	provider := &providerFixture{rows: map[string][]Candidate{
		"Artist music": {
			{ProviderID: "seed", Title: "Song", Channel: "Artist"},
			{ProviderID: "a1", Title: "A1", Channel: "Creator A"},
			{ProviderID: "a2", Title: "A2", Channel: "Creator A"},
		},
		"2006 music": {
			{ProviderID: "b1", Title: "B1", Channel: "Creator B"},
		},
		"Song cover remix live": {
			{ProviderID: "a3", Title: "A3", Channel: "Creator A"},
			{ProviderID: "c1", Title: "C1", Channel: "Creator C"},
		},
	}}
	service := Service{
		Provider: provider,
		ProviderRef: "youtube",
		Limit: 6,
		CreatorCap: 2,
		Now: func() time.Time {
			return time.Date(2026, 10, 14, 1, 2, 3, 4, time.UTC)
		},
	}
	frame, err := service.Explore(context.Background(), Seed{
		ProviderID: "seed",
		Title: "Song",
		Artist: "Artist",
		Year: "2006",
	})
	if err != nil {
		t.Fatal(err)
	}
	if frame.SchemaVersion != SchemaVersion ||
		frame.Basis.Provider != "youtube" ||
		frame.Basis.Method != "SEARCH_DERIVED_MULTI_AXIS" ||
		frame.Basis.Selection != "ROUND_ROBIN" {
		t.Fatalf("frame basis drift: %+v", frame)
	}
	got := make([]string, 0, len(frame.Candidates))
	for _, candidate := range frame.Candidates {
		got = append(got, candidate.ProviderID)
		if candidate.ProviderID == "seed" {
			t.Fatal("seed leaked into candidates")
		}
	}
	want := []string{"a1", "b1", "a3", "a2", "c1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selection drift: got=%v want=%v", got, want)
	}
	if frame.GeneratedAt != "2026-10-14T01:02:03.000000004Z" {
		t.Fatalf("generatedAt drift: %s", frame.GeneratedAt)
	}
}

func TestExploreAnnotatesCandidatesThroughInjectedLibraryMatcher(t *testing.T) {
	provider := &providerFixture{rows: map[string][]Candidate{
		"Song": {{ProviderID: "local", Title: "Other Song", Channel: "Other"}},
		"Song cover remix live": {},
	}}
	matcher := &matcherFixture{}
	frame, err := (Service{Provider: provider, Matcher: matcher}).Explore(
		context.Background(),
		Seed{Title: "Song"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame.Candidates) != 1 ||
		frame.Candidates[0].LibraryMatch.State != "IN_LIBRARY" ||
		frame.Candidates[0].LibraryMatch.TrackRef != "track-local" {
		t.Fatalf("library relationship drift: %+v", frame.Candidates)
	}
	if !reflect.DeepEqual(matcher.seen, []string{"local"}) {
		t.Fatalf("matcher calls drift: %v", matcher.seen)
	}
}

func TestExploreStopsOnProviderFailureInsteadOfPublishingPartialFrame(t *testing.T) {
	provider := &providerFixture{
		rows: map[string][]Candidate{},
		fail: "Song cover remix live",
	}
	_, err := (Service{Provider: provider}).Explore(
		context.Background(),
		Seed{Title: "Song"},
	)
	if err == nil || err.Error() != "provider failed" {
		t.Fatalf("provider failure drift: %v", err)
	}
}

func TestExploreRequiresTitleAndProvider(t *testing.T) {
	if _, err := (Service{}).Explore(context.Background(), Seed{}); err == nil ||
		err.Error() != "explore seed title required" {
		t.Fatalf("missing title drift: %v", err)
	}
	if _, err := (Service{}).Explore(context.Background(), Seed{Title: "Song"}); err == nil ||
		err.Error() != "explore provider required" {
		t.Fatalf("missing provider drift: %v", err)
	}
}

func TestExploreCarriesAxisQueryAndDefaultLibraryState(t *testing.T) {
	provider := &providerFixture{rows: map[string][]Candidate{
		"Song": {{ProviderID: "one", Title: "Other"}},
		"Song cover remix live": {},
	}}
	frame, err := (Service{Provider: provider}).Explore(
		context.Background(),
		Seed{Title: "Song"},
	)
	if err != nil {
		t.Fatal(err)
	}
	candidate := frame.Candidates[0]
	if candidate.Axis != "CLOSER" ||
		candidate.AxisLabel != "Closer" ||
		candidate.Query != "Song" ||
		candidate.LibraryMatch.State != "NOT_IN_LIBRARY" {
		t.Fatalf("candidate annotation drift: %+v", candidate)
	}
}
