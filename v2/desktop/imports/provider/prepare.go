package provider

import (
	"context"
	"errors"

	youtubeprovider "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube"
)

type PrepareSection struct {
	Index       int
	SourceTitle string
	Title       string
	Artist      string
	Start       float64
	End         float64
	Selected    bool
}

type PreparePlan struct {
	URL              string
	Destination      string
	StagingDirectory string
	CancelPath       string
	Mode             string
	WholeTitle       string
	ArtistOverride   string
	Collection       string
	Genre            string
	Sections         []PrepareSection
}

type PreparedOutput struct {
	RelativePath string
	Filename     string
	CatalogEntry map[string]any
}

type PrepareResult struct {
	Outputs []PreparedOutput
	Staged  bool
}

type PrepareEvent struct {
	Type     string
	Stage    string
	Progress int
	Message  string
	Problem  *youtubeprovider.Problem
}

type PrepareBridge interface {
	Prepare(
		context.Context,
		youtubeprovider.PreparePlan,
		func(youtubeprovider.PrepareEvent),
	) (youtubeprovider.PrepareResult, error)
}

type YouTubePreparation struct {
	Bridge PrepareBridge
}

func (y YouTubePreparation) Prepare(
	ctx context.Context,
	plan PreparePlan,
	onEvent func(PrepareEvent),
) (PrepareResult, error) {
	if y.Bridge == nil {
		return PrepareResult{}, errors.New("Import YouTube preparer is unavailable")
	}

	sections := make([]youtubeprovider.PrepareSection, 0, len(plan.Sections))
	for _, section := range plan.Sections {
		sections = append(sections, youtubeprovider.PrepareSection{
			Index: section.Index,
			SourceTitle: section.SourceTitle,
			Title: section.Title,
			Artist: section.Artist,
			Start: section.Start,
			End: section.End,
			Selected: section.Selected,
		})
	}

	result, err := y.Bridge.Prepare(
		ctx,
		youtubeprovider.PreparePlan{
			URL: plan.URL,
			Destination: plan.Destination,
			StagingDirectory: plan.StagingDirectory,
			CancelPath: plan.CancelPath,
			Mode: plan.Mode,
			WholeTitle: plan.WholeTitle,
			ArtistOverride: plan.ArtistOverride,
			Collection: plan.Collection,
			Genre: plan.Genre,
			Sections: sections,
		},
		func(event youtubeprovider.PrepareEvent) {
			if onEvent == nil {
				return
			}
			var problem *youtubeprovider.Problem
			if event.Problem != nil {
				copy := *event.Problem
				problem = &copy
			}
			onEvent(PrepareEvent{
				Type: event.Type,
				Stage: event.Stage,
				Progress: event.Progress,
				Message: event.Message,
				Problem: problem,
			})
		},
	)
	if err != nil {
		return PrepareResult{}, err
	}

	outputs := make([]PreparedOutput, 0, len(result.Outputs))
	for _, output := range result.Outputs {
		outputs = append(outputs, PreparedOutput{
			RelativePath: output.RelativePath,
			Filename: output.Filename,
			CatalogEntry: cloneMap(output.CatalogEntry),
		})
	}
	return PrepareResult{
		Outputs: outputs,
		Staged: result.Staged,
	}, nil
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
