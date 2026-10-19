package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	importmodel "github.com/vgong24/VexstreamMusic/v2/desktop/imports"
	importcommit "github.com/vgong24/VexstreamMusic/v2/desktop/imports/commit"
	importprovider "github.com/vgong24/VexstreamMusic/v2/desktop/imports/provider"
)

type preparerFixture struct {
	result importprovider.PrepareResult
	err    error
	plan   importprovider.PreparePlan
	events []importprovider.PrepareEvent
}

func (f *preparerFixture) Prepare(
	_ context.Context,
	plan importprovider.PreparePlan,
	onEvent func(importprovider.PrepareEvent),
) (importprovider.PrepareResult, error) {
	f.plan = plan
	for _, event := range f.events {
		onEvent(event)
	}
	return f.result, f.err
}

func serviceFixture(t *testing.T, preparer Preparer) Service {
	t.Helper()
	tick := 0
	return Service{
		StagingRoot: t.TempDir(),
		Preparer:    preparer,
		Commit: func(
			_ string,
			_ string,
			_ string,
		) (importcommit.Result, error) {
			return importcommit.Result{
				Added:        1,
				TrackRefs:    []string{"track-a"},
				Destinations: []string{"/music/Song.mp3"},
			}, nil
		},
		DestinationAllowed: func(path string) bool {
			return path == "/music"
		},
		Now: func() time.Time {
			tick++
			return time.Date(2026, 10, 14, 0, 0, tick, 0, time.UTC)
		},
		NewID: func() (string, error) {
			return "job-a", nil
		},
	}
}

func TestRunPersistsLifecycleAndCommitsPreparedOutput(t *testing.T) {
	preparer := &preparerFixture{
		result: importprovider.PrepareResult{
			Staged: true,
			Outputs: []importprovider.PreparedOutput{{
				RelativePath: "outputs/Song.mp3",
				Filename:     "Song.mp3",
			}},
		},
		events: []importprovider.PrepareEvent{
			{Type: "progress", Stage: "Downloading source once", Progress: 10},
			{Type: "progress", Stage: "Prepared — waiting for you to add it", Progress: 100},
		},
	}
	service := serviceFixture(t, preparer)
	seen := []importmodel.Status{}

	outcome, err := service.Run(
		context.Background(),
		Request{
			Title:          "Song",
			Creator:        "Uploader",
			Thumbnail:      "https://example/song.jpg",
			SourceDuration: 123,
			Destination:    "/music",
			URL:            "https://example/song",
			Mode:           "whole",
		},
		func(job importmodel.Job) {
			seen = append(seen, job.Status)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Job.Status != importmodel.StatusDone ||
		outcome.Job.Creator != "Uploader" ||
		outcome.Job.Thumbnail != "https://example/song.jpg" ||
		outcome.Job.SourceURL != "https://example/song" ||
		outcome.Job.SourceDuration != 123 ||
		outcome.Job.OutputCount != 1 ||
		!reflect.DeepEqual(outcome.Job.TrackRefs, []string{"track-a"}) ||
		outcome.Commit == nil ||
		outcome.Commit.Added != 1 {
		t.Fatalf("successful workflow drift: %+v", outcome)
	}
	if preparer.plan.StagingDirectory == "" ||
		preparer.plan.CancelPath == "" ||
		preparer.plan.Destination != "/music" {
		t.Fatalf("prepare custody drift: %+v", preparer.plan)
	}
	wantStatuses := []importmodel.Status{
		importmodel.StatusQueued,
		importmodel.StatusRunning,
		importmodel.StatusRunning,
		importmodel.StatusRunning,
		importmodel.StatusCommitting,
		importmodel.StatusDone,
	}
	if !reflect.DeepEqual(seen, wantStatuses) {
		t.Fatalf("lifecycle drift: got=%v want=%v", seen, wantStatuses)
	}
	if _, err := os.Stat(outcome.StageDir); !os.IsNotExist(err) {
		t.Fatalf("committed staging was not cleaned: %v", err)
	}
}

func TestCommitFailureLeavesReadyStagingForRetryOrDiscard(t *testing.T) {
	preparer := &preparerFixture{
		result: importprovider.PrepareResult{
			Staged:  true,
			Outputs: []importprovider.PreparedOutput{{Filename: "Song.mp3"}},
		},
	}
	service := serviceFixture(t, preparer)
	service.Commit = func(
		_ string,
		stage string,
		_ string,
	) (importcommit.Result, error) {
		if _, err := os.Stat(filepath.Join(stage, "job.json")); err != nil {
			t.Fatalf("job state missing before commit: %v", err)
		}
		return importcommit.Result{}, errors.New("destination busy")
	}

	outcome, err := service.Run(
		context.Background(),
		Request{
			Title:       "Song",
			Destination: "/music",
			URL:         "https://example/song",
		},
		nil,
	)
	var held *CommitHeldError
	if !errors.As(err, &held) {
		t.Fatalf("commit hold type drift: %T %v", err, err)
	}
	if outcome.Job.Status != importmodel.StatusReady ||
		!stringsContains(outcome.Job.Detail, "destination busy") {
		t.Fatalf("ready hold drift: %+v", outcome.Job)
	}
	if _, statErr := os.Stat(outcome.StageDir); statErr != nil {
		t.Fatalf("ready staging was removed: %v", statErr)
	}
}

func TestProviderFailureLeavesFailedStagingAndProblemEvidence(t *testing.T) {
	problem := importprovider.Problem{
		SchemaVersion: "vexstream.provider-problem/v1",
		Provider:      "youtube",
		Stage:         "import.download",
		Code:          "RATE_LIMIT",
		Title:         "Provider busy",
		Message:       "Retry later",
		Retryable:     true,
	}
	preparer := &preparerFixture{
		err: errors.New("provider failed"),
		events: []importprovider.PrepareEvent{{
			Type:    "error",
			Problem: &problem,
		}},
	}
	service := serviceFixture(t, preparer)

	outcome, err := service.Run(
		context.Background(),
		Request{
			Title:       "Song",
			Destination: "/music",
			URL:         "https://example/song",
		},
		nil,
	)
	if err == nil {
		t.Fatal("provider failure was swallowed")
	}
	if outcome.Job.Status != importmodel.StatusFailed ||
		outcome.Job.Problem["code"] != "RATE_LIMIT" {
		t.Fatalf("failed provider evidence drift: %+v", outcome.Job)
	}
	if _, statErr := os.Stat(outcome.StageDir); statErr != nil {
		t.Fatalf("failed staging was removed: %v", statErr)
	}
}

func TestCancellationRemovesStagingAndReturnsCancelledJob(t *testing.T) {
	preparer := &preparerFixture{
		err: importprovider.ErrPrepareCancelled,
	}
	service := serviceFixture(t, preparer)

	outcome, err := service.Run(
		context.Background(),
		Request{
			Title:       "Song",
			Destination: "/music",
			URL:         "https://example/song",
		},
		nil,
	)
	if !errors.Is(err, importprovider.ErrPrepareCancelled) {
		t.Fatalf("cancellation error drift: %T %v", err, err)
	}
	if outcome.Job.Status != importmodel.StatusCancelled {
		t.Fatalf("cancelled job drift: %+v", outcome.Job)
	}
	if _, statErr := os.Stat(outcome.StageDir); !os.IsNotExist(statErr) {
		t.Fatalf("cancelled staging was not cleaned: %v", statErr)
	}
}

func TestDestinationIsAdmittedBeforeStagingOrProviderWork(t *testing.T) {
	preparer := &preparerFixture{}
	service := serviceFixture(t, preparer)

	_, err := service.Run(
		context.Background(),
		Request{
			Title:       "Song",
			Destination: "/outside",
			URL:         "https://example/song",
		},
		nil,
	)
	if err == nil {
		t.Fatal("unadmitted destination accepted")
	}
	if preparer.plan.URL != "" {
		t.Fatalf("provider ran for unadmitted destination: %+v", preparer.plan)
	}
	entries, readErr := os.ReadDir(service.StagingRoot)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("staging formed before destination admission: %+v", entries)
	}
}

func stringsContains(value, fragment string) bool {
	return len(value) >= len(fragment) &&
		reflect.ValueOf(value).String() != "" &&
		contains(value, fragment)
}

func contains(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
