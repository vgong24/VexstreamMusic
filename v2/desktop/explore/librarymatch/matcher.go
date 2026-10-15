// Package librarymatch adapts current Library truth into Explore's advisory relationship vocabulary.
//
// It reuses the Import duplicate-review scorer; it does not mutate Library or decide import admission.
package librarymatch

import (
	"github.com/vgong24/VexstreamMusic/v2/desktop/explore"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/duplicates"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

type SnapshotFunc func() []library.Track

type Matcher struct {
	snapshot SnapshotFunc
}

func New(snapshot SnapshotFunc) *Matcher {
	if snapshot == nil {
		panic("Explore Library matcher requires snapshot")
	}
	return &Matcher{snapshot: snapshot}
}

func (m *Matcher) Match(candidate explore.Candidate) explore.LibraryMatch {
	matches := duplicates.Find(duplicates.Incoming{
		ProviderID:    candidate.ProviderID,
		URL:           candidate.URL,
		Title:         candidate.Title,
		SourceCreator: candidate.Channel,
		Duration:      candidate.Duration,
	}, m.snapshot())
	if len(matches) == 0 {
		return explore.LibraryMatch{State: "NOT_IN_LIBRARY"}
	}

	best := matches[0]
	state := "POSSIBLE_MATCH"
	switch {
	case best.Relation == "SAME_SOURCE":
		state = "IN_LIBRARY"
	case best.Score >= 75:
		state = "LIKELY_MATCH"
	}

	return explore.LibraryMatch{
		State:      state,
		TrackRef:   best.TrackRef,
		Title:      best.Title,
		Artist:     best.Artist,
		Confidence: best.Confidence,
		Relation:   best.Relation,
		Score:      best.Score,
	}
}
