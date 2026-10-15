package provider

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/duplicates"
	youtubeprovider "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube"
)

type inspectFixture struct {
	row  youtubeprovider.Inspection
	err  error
	seen string
}

func (f *inspectFixture) Inspect(
	_ context.Context,
	url string,
) (youtubeprovider.Inspection, error) {
	f.seen = url
	return f.row, f.err
}

func TestYouTubeMapsProviderInspectionIntoImportMeaning(t *testing.T) {
	fixture := &inspectFixture{
		row: youtubeprovider.Inspection{
			ProviderID: " abc ",
			URL:        " https://example ",
			Title:      " Song ",
			Channel:    " Creator ",
			Duration:   123,
			Artist:     " Artist ",
			Album:      " Album ",
			Genre:      " Dream Pop ",
			Sections: []youtubeprovider.Section{
				{
					Index: 1,
					Start: 0,
					End: 30,
					Title: " Part 1 ",
					Selected: true,
				},
				{
					Index: 2,
					Start: 30,
					End: 60,
					Title: " Part 2 ",
					Selected: false,
				},
			},
			HasSections: true,
			SourceQuality: youtubeprovider.AudioQuality{
				Codec: "opus",
				BitrateKbps: 160,
			},
		},
	}

	adapter := YouTube{Inspector: fixture}
	got, err := adapter.Inspect(context.Background(), "https://example")
	if err != nil {
		t.Fatal(err)
	}
	if fixture.seen != "https://example" ||
		got.ProviderID != "abc" ||
		got.Title != "Song" ||
		got.Artist != "Artist" ||
		got.Album != "Album" ||
		got.Genre != "Dream Pop" ||
		len(got.Sections) != 2 ||
		got.Sections[0].Title != "Part 1" ||
		got.SourceQuality.Codec != "opus" {
		t.Fatalf("inspection mapping drift: %+v", got)
	}
}

func TestDuplicateReviewRequestUsesSelectedSectionsAndKeepsCreatorDistinctFromArtist(t *testing.T) {
	inspection := Inspection{
		ProviderID: "abc",
		URL: "https://example",
		Title: "Song",
		Artist: "",
		Channel: "Uploader",
		Duration: 120,
		SourceQuality: duplicates.AudioQuality{
			Codec: "opus",
		},
		Sections: []Section{
			{Index: 1, Title: "Part 1", Start: 0, End: 30, Selected: true},
			{Index: 2, Title: "Part 2", Start: 30, End: 60, Selected: false},
		},
	}
	got := inspection.DuplicateReviewRequest()

	if got.ProviderID != "abc" ||
		got.Artist != "" ||
		got.SourceCreator != "Uploader" ||
		got.SourceQuality.Codec != "opus" ||
		len(got.Sections) != 1 ||
		!reflect.DeepEqual(
			got.Sections[0],
			duplicates.Section{
				Index: 1,
				Title: "Part 1",
				Start: 0,
				End: 30,
			},
		) {
		t.Fatalf("duplicate review mapping drift: %+v", got)
	}
}

func TestYouTubePreservesInspectorFailure(t *testing.T) {
	expected := errors.New("inspect failed")
	adapter := YouTube{
		Inspector: &inspectFixture{err: expected},
	}
	_, err := adapter.Inspect(context.Background(), "https://example")
	if !errors.Is(err, expected) {
		t.Fatalf("inspector failure was rewritten: %v", err)
	}
}

func TestYouTubeRequiresInspector(t *testing.T) {
	_, err := (YouTube{}).Inspect(context.Background(), "https://example")
	if err == nil {
		t.Fatal("missing inspector accepted")
	}
}
