package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func prepareBridge(t *testing.T) (*Bridge, *[][]string) {
	t.Helper()
	seen := [][]string{}
	bridge := New("python3", "/tmp/media_bridge.py", nil)
	bridge.stream = func(
		_ context.Context,
		command []string,
		onStdout func([]byte),
		_ func([]byte),
	) error {
		seen = append(seen, append([]string(nil), command...))
		planBytes, err := os.ReadFile(command[3])
		if err != nil {
			t.Fatal(err)
		}
		var plan PreparePlan
		if err := json.Unmarshal(planBytes, &plan); err != nil {
			t.Fatal(err)
		}
		if plan.URL != "https://example/song" ||
			plan.Mode != "both" ||
			plan.CancelPath == "" ||
			len(plan.Sections) != 1 {
			t.Fatalf("prepare plan drift: %+v", plan)
		}

		onStdout([]byte(`{"type":"progress","stage":"Downloading source once","progress":10}`))
		onStdout([]byte(`{"type":"progress","stage":"Prepared — waiting for you to add it","progress":100}`))
		onStdout([]byte(`{"ok":true,"outputs":[{"relativePath":"outputs/Song.mp3","filename":"Song.mp3","catalogEntry":{"provider":"youtube","providerId":"abc"}}],"staged":true}`))
		return nil
	}
	return bridge, &seen
}

func TestPrepareWritesTypedPlanStreamsProgressAndReturnsStagedOutputs(t *testing.T) {
	bridge, seen := prepareBridge(t)
	stage := filepath.Join(t.TempDir(), "stage")
	events := []PrepareEvent{}

	got, err := bridge.Prepare(
		context.Background(),
		PreparePlan{
			URL: " https://example/song ",
			Destination: "/music",
			StagingDirectory: stage,
			Mode: "both",
			Sections: []PrepareSection{{
				Index: 1,
				SourceTitle: "Part 1",
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
	if !got.Staged || len(got.Outputs) != 1 ||
		got.Outputs[0].Filename != "Song.mp3" ||
		got.Outputs[0].CatalogEntry["providerId"] != "abc" {
		t.Fatalf("prepare result drift: %+v", got)
	}
	if len(events) != 2 ||
		events[0].Type != "progress" ||
		events[0].Progress != 10 ||
		events[1].Progress != 100 {
		t.Fatalf("progress event drift: %+v", events)
	}
	wantCommand := []string{
		"python3",
		"/tmp/media_bridge.py",
		"import",
		filepath.Join(stage, "request.json"),
	}
	if len(*seen) != 1 || !reflect.DeepEqual((*seen)[0], wantCommand) {
		t.Fatalf("prepare command drift: got=%v want=%v", *seen, wantCommand)
	}
}

func TestPrepareCancellationWinsOverProcessExitFailure(t *testing.T) {
	bridge := New("python3", "media_bridge.py", nil)
	bridge.stream = func(
		_ context.Context,
		_ []string,
		onStdout func([]byte),
		_ func([]byte),
	) error {
		onStdout([]byte(`{"type":"cancelled","message":"Import cancelled while creating MP3 output"}`))
		return errors.New("exit status 2")
	}

	_, err := bridge.Prepare(
		context.Background(),
		PreparePlan{
			URL: "https://example/song",
			Destination: "/music",
			StagingDirectory: t.TempDir(),
		},
		nil,
	)
	if !errors.Is(err, ErrPrepareCancelled) {
		t.Fatalf("cancellation drift: %T %v", err, err)
	}
}

func TestPreparePreservesStructuredProviderProblem(t *testing.T) {
	bridge := New("python3", "media_bridge.py", nil)
	bridge.stream = func(
		_ context.Context,
		_ []string,
		onStdout func([]byte),
		_ func([]byte),
	) error {
		onStdout([]byte(`{"type":"error","problem":{"schemaVersion":"vexstream.provider-problem/v1","provider":"youtube","stage":"import.download","code":"RATE_LIMIT","title":"Provider busy","message":"Retry later","retryable":true}}`))
		return errors.New("exit status 1")
	}

	_, err := bridge.Prepare(
		context.Background(),
		PreparePlan{
			URL: "https://example/song",
			Destination: "/music",
			StagingDirectory: t.TempDir(),
		},
		nil,
	)
	var failure *Failure
	if !errors.As(err, &failure) ||
		failure.Problem.Code != "RATE_LIMIT" ||
		failure.Problem.Stage != "import.download" {
		t.Fatalf("provider problem drift: %T %+v", err, err)
	}
}

func TestPrepareProcessFailureUsesStderrFallbackProblem(t *testing.T) {
	bridge := New("python3", "media_bridge.py", nil)
	bridge.stream = func(
		_ context.Context,
		_ []string,
		_ func([]byte),
		onStderr func([]byte),
	) error {
		onStderr([]byte("provider exploded"))
		return errors.New("exit status 1")
	}

	_, err := bridge.Prepare(
		context.Background(),
		PreparePlan{
			URL: "https://example/song",
			Destination: "/music",
			StagingDirectory: t.TempDir(),
		},
		nil,
	)
	var failure *Failure
	if !errors.As(err, &failure) ||
		failure.Problem.Code != "YOUTUBE_PROVIDER_ERROR" ||
		failure.Problem.Technical != "provider exploded" {
		t.Fatalf("fallback problem drift: %T %+v", err, err)
	}
}

func TestPrepareRejectsInvalidPlanBeforeProviderExecution(t *testing.T) {
	bridge := New("python3", "media_bridge.py", nil)
	calls := 0
	bridge.stream = func(
		_ context.Context,
		_ []string,
		_ func([]byte),
		_ func([]byte),
	) error {
		calls++
		return nil
	}

	cases := []PreparePlan{
		{},
		{URL: "x", Destination: "/music"},
		{URL: "x", StagingDirectory: t.TempDir()},
		{URL: "x", Destination: "/music", StagingDirectory: t.TempDir(), Mode: "mystery"},
	}
	for _, plan := range cases {
		if _, err := bridge.Prepare(context.Background(), plan, nil); err == nil {
			t.Fatalf("invalid prepare plan accepted: %+v", plan)
		}
	}
	if calls != 0 {
		t.Fatalf("provider executed for invalid plan: %d", calls)
	}
}
