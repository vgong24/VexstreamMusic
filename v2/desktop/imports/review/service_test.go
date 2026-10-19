package review

import (
	"context"
	"errors"
	"testing"

	importprovider "github.com/vgong24/VexstreamMusic/v2/desktop/imports/provider"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

type inspectorFixture struct {
	inspection importprovider.Inspection
	err        error
	seen       string
}

func (f *inspectorFixture) Inspect(
	_ context.Context,
	url string,
) (importprovider.Inspection, error) {
	f.seen = url
	return f.inspection, f.err
}

func TestInspectAndReviewComposesProviderAndCurrentLibrary(t *testing.T) {
	inspector := &inspectorFixture{
		inspection: importprovider.Inspection{
			ProviderID: "provider-a",
			URL:        "https://example/song",
			Title:      "Song",
			Artist:     "Artist",
			Channel:    "Uploader",
			Duration:   180,
		},
	}
	service := Service{
		Inspector: inspector,
		Library: func() []library.Track {
			return []library.Track{{
				ID:         "track-a",
				Title:      "Song",
				Artist:     "Artist",
				ProviderID: "provider-a",
				SourceURL:  "https://example/song",
				Duration:   180,
			}}
		},
	}

	got, err := service.InspectAndReview(
		context.Background(),
		"https://example/song",
	)
	if err != nil {
		t.Fatal(err)
	}
	if inspector.seen != "https://example/song" ||
		got.Inspection.ProviderID != "provider-a" ||
		!got.Advisory ||
		got.MatchCount != 1 ||
		len(got.Items) != 1 ||
		len(got.Items[0].Matches) != 1 ||
		got.Items[0].Matches[0].TrackRef != "track-a" ||
		got.Items[0].Matches[0].Relation != "SAME_SOURCE" {
		t.Fatalf("review composition drift: %+v", got)
	}
}

func TestInspectAndReviewKeepsNoMatchAsSuccessfulAdvisory(t *testing.T) {
	service := Service{
		Inspector: &inspectorFixture{
			inspection: importprovider.Inspection{
				Title: "New Song",
			},
		},
		Library: func() []library.Track { return nil },
	}

	got, err := service.InspectAndReview(
		context.Background(),
		"https://example/new",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Advisory || got.MatchCount != 0 || len(got.Items) != 0 {
		t.Fatalf("no-match advisory drift: %+v", got)
	}
}

func TestInspectAndReviewPreservesProviderFailure(t *testing.T) {
	expected := errors.New("provider failed")
	service := Service{
		Inspector: &inspectorFixture{err: expected},
		Library:   func() []library.Track { return nil },
	}

	_, err := service.InspectAndReview(
		context.Background(),
		"https://example/song",
	)
	if !errors.Is(err, expected) {
		t.Fatalf("provider failure was rewritten: %v", err)
	}
}

func TestInspectAndReviewRequiresBothOwners(t *testing.T) {
	if _, err := (Service{}).InspectAndReview(
		context.Background(),
		"https://example/song",
	); err == nil {
		t.Fatal("missing inspector accepted")
	}

	if _, err := (Service{
		Inspector: &inspectorFixture{},
	}).InspectAndReview(
		context.Background(),
		"https://example/song",
	); err == nil {
		t.Fatal("missing Library snapshot accepted")
	}
}
