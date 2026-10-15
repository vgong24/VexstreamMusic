package provider

import (
	"context"
	"errors"
	"reflect"
	"testing"

	youtubeprovider "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube"
)

type prepareFixture struct {
	plan   youtubeprovider.PreparePlan
	result youtubeprovider.PrepareResult
	err    error
}

func (f *prepareFixture) Prepare(
	_ context.Context,
	plan youtubeprovider.PreparePlan,
	onEvent func(youtubeprovider.PrepareEvent),
) (youtubeprovider.PrepareResult, error) {
	f.plan = plan
	if onEvent != nil {
		onEvent(youtubeprovider.PrepareEvent{
			Type: "progress",
			Stage: "Downloading source once",
			Progress: 10,
		})
	}
	return f.result, f.err
}

func TestYouTubePreparationMapsImportPlanEventsAndOutputs(t *testing.T) {
	bridge := &prepareFixture{
		result: youtubeprovider.PrepareResult{
			Staged: true,
			Outputs: []youtubeprovider.PreparedOutput{{
				RelativePath: "outputs/Song.mp3",
				Filename: "Song.mp3",
				CatalogEntry: map[string]any{"providerId": "abc"},
			}},
		},
	}
	adapter := YouTubePreparation{Bridge: bridge}
	events := []PrepareEvent{}

	got, err := adapter.Prepare(
		context.Background(),
		PreparePlan{
			URL: "https://example/song",
			Destination: "/music",
			StagingDirectory: "/stage",
			Mode: "both",
			Sections: []PrepareSection{{
				Index: 1,
				Title: "Part 1",
				Start: 0,
				End: 30,
				Selected: true,
			}},
		},
		func(event PrepareEvent) {
			events = append(events, event)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bridge.plan.URL != "https://example/song" ||
		bridge.plan.Mode != "both" ||
		len(bridge.plan.Sections) != 1 ||
		!bridge.plan.Sections[0].Selected {
		t.Fatalf("plan mapping drift: %+v", bridge.plan)
	}
	if !got.Staged || len(got.Outputs) != 1 ||
		got.Outputs[0].CatalogEntry["providerId"] != "abc" {
		t.Fatalf("result mapping drift: %+v", got)
	}
	if !reflect.DeepEqual(
		events,
		[]PrepareEvent{{
			Type: "progress",
			Stage: "Downloading source once",
			Progress: 10,
		}},
	) {
		t.Fatalf("event mapping drift: %+v", events)
	}
}

func TestYouTubePreparationPreservesBridgeFailure(t *testing.T) {
	expected := errors.New("prepare failed")
	adapter := YouTubePreparation{
		Bridge: &prepareFixture{err: expected},
	}
	_, err := adapter.Prepare(
		context.Background(),
		PreparePlan{},
		nil,
	)
	if !errors.Is(err, expected) {
		t.Fatalf("prepare failure was rewritten: %v", err)
	}
}

func TestYouTubePreparationRequiresBridge(t *testing.T) {
	_, err := (YouTubePreparation{}).Prepare(
		context.Background(),
		PreparePlan{},
		nil,
	)
	if err == nil {
		t.Fatal("missing bridge accepted")
	}
}
