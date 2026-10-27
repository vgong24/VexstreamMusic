package provider

import (
	"context"
	"errors"
	"reflect"
	"testing"

	youtubeprovider "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube"
)

type searchFixture struct {
	rows []youtubeprovider.SearchResult
	err  error
	seen []any
}

func (f *searchFixture) Search(
	_ context.Context,
	query string,
	limit int,
) ([]youtubeprovider.SearchResult, error) {
	f.seen = append(f.seen, query, limit)
	return append([]youtubeprovider.SearchResult(nil), f.rows...), f.err
}

func TestYouTubeMapsProviderRowsIntoExploreCandidates(t *testing.T) {
	search := &searchFixture{
		rows: []youtubeprovider.SearchResult{
			{
				ProviderID: " abc ",
				Title:      " Song ",
				Channel:    " Creator ",
				URL:        " https://example ",
				Thumbnail:  " https://img ",
				Duration:   123,
			},
			{ProviderID: "blank", Title: "   "},
		},
	}
	adapter := YouTube{Searcher: search}

	rows, err := adapter.Search(context.Background(), "dream pop", 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("candidate filtering drift: %+v", rows)
	}
	got := rows[0]
	if got.ProviderID != "abc" ||
		got.Title != "Song" ||
		got.Channel != "Creator" ||
		got.URL != "https://example" ||
		got.Thumbnail != "https://img" ||
		got.Duration != 123 {
		t.Fatalf("candidate mapping drift: %+v", got)
	}
	if !reflect.DeepEqual(search.seen, []any{"dream pop", 6}) {
		t.Fatalf("search forwarding drift: %+v", search.seen)
	}
}

func TestYouTubePreservesProviderFailure(t *testing.T) {
	expected := errors.New("provider failed")
	adapter := YouTube{
		Searcher: &searchFixture{err: expected},
	}
	_, err := adapter.Search(context.Background(), "song", 8)
	if !errors.Is(err, expected) {
		t.Fatalf("provider failure was rewritten: %v", err)
	}
}

func TestYouTubeRequiresSearcher(t *testing.T) {
	_, err := (YouTube{}).Search(context.Background(), "song", 8)
	if err == nil {
		t.Fatal("missing searcher accepted")
	}
}
