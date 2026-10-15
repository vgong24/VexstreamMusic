package provider

import (
	"context"
	"errors"

	youtubeprovider "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube"
)

var ErrPrepareCancelled = errors.New("Import preparation cancelled")

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

type Problem struct {
	SchemaVersion   string
	Provider        string
	Stage           string
	Code            string
	Title           string
	Message         string
	Retryable       bool
	SuggestedAction string
	Technical       string
	ObservedAt      string
}

type PrepareEvent struct {
	Type     string
	Stage    string
	Progress int
	Message  string
	Problem  *Problem
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
			var problem *Problem
			if event.Problem != nil {
				problem = &Problem{
					SchemaVersion: event.Problem.SchemaVersion,
					Provider: event.Problem.Provider,
					Stage: event.Problem.Stage,
					Code: event.Problem.Code,
					Title: event.Problem.Title,
					Message: event.Problem.Message,
					Retryable: event.Problem.Retryable,
					SuggestedAction: event.Problem.SuggestedAction,
					Technical: event.Problem.Technical,
					ObservedAt: event.Problem.ObservedAt,
				}
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
		if errors.Is(err, youtubeprovider.ErrPrepareCancelled) {
			return PrepareResult{}, ErrPrepareCancelled
		}
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

func ProblemFromError(err error) (Problem, bool) {
	var failure *youtubeprovider.Failure
	if !errors.As(err, &failure) || failure == nil {
		return Problem{}, false
	}
	return Problem{
		SchemaVersion: failure.Problem.SchemaVersion,
		Provider: failure.Problem.Provider,
		Stage: failure.Problem.Stage,
		Code: failure.Problem.Code,
		Title: failure.Problem.Title,
		Message: failure.Problem.Message,
		Retryable: failure.Problem.Retryable,
		SuggestedAction: failure.Problem.SuggestedAction,
		Technical: failure.Problem.Technical,
		ObservedAt: failure.Problem.ObservedAt,
	}, true
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
