package review

import (
	"context"
	"errors"

	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/duplicates"
	importprovider "github.com/vgong24/VexstreamMusic/v2/desktop/imports/provider"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

type Inspector interface {
	Inspect(context.Context, string) (importprovider.Inspection, error)
}

type LibrarySnapshot func() []library.Track

type Service struct {
	Inspector Inspector
	Library   LibrarySnapshot
}

type Result struct {
	Inspection importprovider.Inspection
	Items      []duplicates.Item
	MatchCount int
	Advisory   bool
}

func (s Service) InspectAndReview(
	ctx context.Context,
	url string,
) (Result, error) {
	if s.Inspector == nil {
		return Result{}, errors.New("Import inspector is unavailable")
	}
	if s.Library == nil {
		return Result{}, errors.New("Import Library snapshot is unavailable")
	}

	inspection, err := s.Inspector.Inspect(ctx, url)
	if err != nil {
		return Result{}, err
	}
	items := duplicates.Review(
		inspection.DuplicateReviewRequest(),
		s.Library(),
	)
	return Result{
		Inspection: inspection,
		Items:      items,
		MatchCount: len(items),
		Advisory:   true,
	}, nil
}
