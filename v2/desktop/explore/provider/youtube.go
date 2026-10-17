package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/explore"
	youtubeprovider "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube"
)

type Searcher interface {
	Search(context.Context, string, int) ([]youtubeprovider.SearchResult, error)
}

type YouTube struct {
	Searcher Searcher
}

func (y YouTube) Search(
	ctx context.Context,
	query string,
	limit int,
) ([]explore.Candidate, error) {
	if y.Searcher == nil {
		return nil, errors.New("Explore YouTube provider is unavailable")
	}

	rows, err := y.Searcher.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	out := make([]explore.Candidate, 0, len(rows))
	for _, row := range rows {
		title := strings.TrimSpace(row.Title)
		if title == "" {
			continue
		}
		out = append(out, explore.Candidate{
			ProviderID: strings.TrimSpace(row.ProviderID),
			Title:      title,
			Channel:    strings.TrimSpace(row.Channel),
			URL:        strings.TrimSpace(row.URL),
			Thumbnail:  strings.TrimSpace(row.Thumbnail),
			Duration:   row.Duration,
		})
	}
	return out, nil
}
