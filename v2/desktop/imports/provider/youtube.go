package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/duplicates"
	youtubeprovider "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube"
)

type Inspector interface {
	Inspect(context.Context, string) (youtubeprovider.Inspection, error)
}

type Section struct {
	Index    int
	Start    float64
	End      float64
	Title    string
	Artist   string
	Selected bool
}

type Inspection struct {
	ProviderID    string
	URL           string
	Title         string
	Channel       string
	Duration      float64
	Thumbnail     string
	Description   string
	Tags          []string
	Categories    []string
	Artist        string
	Album         string
	Genre         string
	UploadDate    string
	Sections      []Section
	HasSections   bool
	SourceQuality duplicates.AudioQuality
}

func (i Inspection) DuplicateReviewRequest() duplicates.ReviewRequest {
	sections := make([]duplicates.Section, 0, len(i.Sections))
	for _, section := range i.Sections {
		if !section.Selected {
			continue
		}
		sections = append(sections, duplicates.Section{
			Index: section.Index,
			Title: section.Title,
			Start: section.Start,
			End:   section.End,
		})
	}

	return duplicates.ReviewRequest{
		Incoming: duplicates.Incoming{
			ProviderID:    i.ProviderID,
			URL:           i.URL,
			Title:         i.Title,
			Artist:        i.Artist,
			SourceCreator: i.Channel,
			Duration:      i.Duration,
			SourceQuality: i.SourceQuality,
		},
		Sections: sections,
	}
}

type YouTube struct {
	Inspector Inspector
}

func (y YouTube) Inspect(
	ctx context.Context,
	url string,
) (Inspection, error) {
	if y.Inspector == nil {
		return Inspection{}, errors.New("Import YouTube inspector is unavailable")
	}

	row, err := y.Inspector.Inspect(ctx, url)
	if err != nil {
		return Inspection{}, err
	}

	sections := make([]Section, 0, len(row.Sections))
	for _, section := range row.Sections {
		sections = append(sections, Section{
			Index:    section.Index,
			Start:    section.Start,
			End:      section.End,
			Title:    strings.TrimSpace(section.Title),
			Artist:   strings.TrimSpace(section.Artist),
			Selected: section.Selected,
		})
	}

	return Inspection{
		ProviderID:  strings.TrimSpace(row.ProviderID),
		URL:         strings.TrimSpace(row.URL),
		Title:       strings.TrimSpace(row.Title),
		Channel:     strings.TrimSpace(row.Channel),
		Duration:    row.Duration,
		Thumbnail:   strings.TrimSpace(row.Thumbnail),
		Description: row.Description,
		Tags:        append([]string(nil), row.Tags...),
		Categories:  append([]string(nil), row.Categories...),
		Artist:      strings.TrimSpace(row.Artist),
		Album:       strings.TrimSpace(row.Album),
		Genre:       strings.TrimSpace(row.Genre),
		UploadDate:  strings.TrimSpace(row.UploadDate),
		Sections:    sections,
		HasSections: row.HasSections,
		SourceQuality: duplicates.AudioQuality{
			Codec:        row.SourceQuality.Codec,
			Container:    row.SourceQuality.Container,
			BitrateKbps:  row.SourceQuality.BitrateKbps,
			SampleRateHz: row.SourceQuality.SampleRateHz,
			Channels:     row.SourceQuality.Channels,
			Basis:        row.SourceQuality.Basis,
		},
	}, nil
}
