package jobs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	importmodel "github.com/vgong24/VexstreamMusic/v2/desktop/imports"
	importcommit "github.com/vgong24/VexstreamMusic/v2/desktop/imports/commit"
	importprovider "github.com/vgong24/VexstreamMusic/v2/desktop/imports/provider"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/staging"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/workflow"
	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

type preparerFixture struct {
	mu     sync.Mutex
	calls  int
	result importprovider.PrepareResult
	err    error
	events []importprovider.PrepareEvent
}

func (f *preparerFixture) Prepare(
	_ context.Context,
	_ importprovider.PreparePlan,
	onEvent func(importprovider.PrepareEvent),
) (importprovider.PrepareResult, error) {
	f.mu.Lock()
	f.calls++
	events := append([]importprovider.PrepareEvent(nil), f.events...)
	result := f.result
	err := f.err
	f.mu.Unlock()

	for _, event := range events {
		if onEvent != nil {
			onEvent(event)
		}
	}
	return result, err
}

func (f *preparerFixture) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func testWorkflow(root string, preparer importproviderPreparer) workflow.Service {
	return workflow.Service{
		StagingRoot: root,
		Preparer: preparer,
		Commit: func(
			_ string,
			_ string,
			_ string,
		) (importcommit.Result, error) {
			return importcommit.Result{
				Added: 1,
				TrackRefs: []string{"track-a"},
			}, nil
		},
		DestinationAllowed: func(path string) bool {
			return path == filepath.Join(root, "music")
		},
		Now: func() time.Time {
			return time.Date(2026, 10, 14, 1, 2, 3, 0, time.UTC)
		},
	}
}

type importproviderPreparer interface {
	Prepare(
		context.Context,
		importprovider.PreparePlan,
		func(importprovider.PrepareEvent),
	) (importprovider.PrepareResult, error)
}

func persistStage(
	t *testing.T,
	root string,
	job importmodel.Job,
	manifest *staging.Manifest,
	outputs map[string][]byte,
) string {
	t.Helper()
	stage := filepath.Join(root, job.ID)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := persistence.SaveJSONAtomic(
		filepath.Join(stage, "job.json"),
		job,
	); err != nil {
		t.Fatal(err)
	}
	if manifest != nil {
		if manifest.SchemaVersion == "" {
			manifest.SchemaVersion = staging.ManifestSchema
		}
		if err := persistence.SaveJSONAtomic(
			filepath.Join(stage, "manifest.json"),
			manifest,
		); err != nil {
			t.Fatal(err)
		}
	}
	for relative, data := range outputs {
		path := filepath.Join(stage, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return stage
}

func findJob(snapshot Snapshot, id string) (importmodel.Job, bool) {
	for _, rows := range [][]importmodel.Job{
		snapshot.Groups.Active,
		snapshot.Groups.Ready,
		snapshot.Groups.Completed,
	} {
		for _, job := range rows {
			if job.ID == id {
				return job, true
			}
		}
	}
	return importmodel.Job{}, false
}

func waitJob(
	t *testing.T,
	manager *Manager,
	id string,
	predicate func(importmodel.Job) bool,
) importmodel.Job {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if job, ok := findJob(manager.Snapshot(), id); ok && predicate(job) {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	job, _ := findJob(manager.Snapshot(), id)
	t.Fatalf("job %s did not reach expected state: %+v", id, job)
	return importmodel.Job{}
}

func TestNewMakesInterruptedPreparationVisibleAsFailedInsteadOfGhostRunning(t *testing.T) {
	root := t.TempDir()
	stage := persistStage(
		t,
		root,
		importmodel.Job{
			ID: "interrupted",
			Title: "Song",
			Status: importmodel.StatusRunning,
			Stage: "Downloading",
			Destination: filepath.Join(root, "music"),
		},
		nil,
		nil,
	)

	manager, err := New(workflow.Service{StagingRoot: root}, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	snapshot := manager.Snapshot()
	job, ok := findJob(snapshot, "interrupted")
	if !ok ||
		job.Status != importmodel.StatusFailed ||
		job.Stage != "Interrupted" ||
		snapshot.Recovery["interrupted"] != importmodel.RestartInterruptedPreparation {
		t.Fatalf("interrupted recovery drift: job=%+v snapshot=%+v", job, snapshot)
	}

	var persisted importmodel.Job
	found, err := persistence.LoadJSON(
		filepath.Join(stage, "job.json"),
		&persisted,
	)
	if err != nil || !found || persisted.Status != importmodel.StatusFailed {
		t.Fatalf("interrupted state was not durably currentized: %+v found=%v err=%v", persisted, found, err)
	}
}

func TestNewRestoresReadyStagingWithoutReplayingProviderOrCommit(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "music")
	persistStage(
		t,
		root,
		importmodel.Job{
			ID: "ready",
			Title: "Song",
			Status: importmodel.StatusReady,
			Destination: destination,
		},
		&staging.Manifest{
			Destination: destination,
			Outputs: []staging.Output{{
				RelativePath: "outputs/song.mp3",
				Filename: "song.mp3",
			}},
		},
		map[string][]byte{
			"outputs/song.mp3": []byte("audio"),
		},
	)

	manager, err := New(workflow.Service{StagingRoot: root}, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	snapshot := manager.Snapshot()
	if len(snapshot.Groups.Ready) != 1 ||
		snapshot.Groups.Ready[0].ID != "ready" ||
		snapshot.Recovery["ready"] != importmodel.RestartReadyStaging {
		t.Fatalf("ready recovery drift: %+v", snapshot)
	}
}

func TestRecoverCommitRequiresExistingJournalAndNeverAutoReplaysUnjournaledCommit(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "music")
	manifest := staging.Manifest{
		SchemaVersion: staging.ManifestSchema,
		Destination: destination,
		Outputs: []staging.Output{{
			RelativePath: "outputs/song.mp3",
			Filename: "song.mp3",
			CatalogEntry: map[string]any{
				"provider": "youtube",
				"providerId": "song",
			},
		}},
	}
	stage := persistStage(
		t,
		root,
		importmodel.Job{
			ID: "commit-uncertain",
			Title: "Song",
			Status: importmodel.StatusCommitting,
			Destination: destination,
		},
		&manifest,
		map[string][]byte{
			"outputs/song.mp3": []byte("audio"),
		},
	)

	service := workflow.Service{
		StagingRoot: root,
		Commit: func(
			_ string,
			stageDir string,
			_ string,
		) (importcommit.Result, error) {
			return importcommit.Resume(stageDir)
		},
	}
	manager, err := New(service, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	if _, err := manager.RecoverCommit("commit-uncertain"); !errors.Is(err, ErrRecoveryNotReady) {
		t.Fatalf("unjournaled uncertain commit was replayed: %v", err)
	}

	if _, err := importcommit.Prepare(
		"commit-uncertain",
		stage,
		destination,
		manifest,
	); err != nil {
		t.Fatal(err)
	}

	job, err := manager.RecoverCommit("commit-uncertain")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != importmodel.StatusDone ||
		len(job.TrackRefs) != 1 {
		t.Fatalf("journaled recovery drift: %+v", job)
	}
	if _, err := os.Stat(stage); !os.IsNotExist(err) {
		t.Fatalf("recovered staging not cleaned: %v", err)
	}
}

func TestQueuedCancellationCanStopBeforeProviderExecution(t *testing.T) {
	root := t.TempDir()
	preparer := &preparerFixture{
		result: importprovider.PrepareResult{
			Staged: true,
			Outputs: []importprovider.PreparedOutput{{
				Filename: "song.mp3",
			}},
		},
	}
	slots := make(chan struct{}, 1)
	slots <- struct{}{}
	service := testWorkflow(root, preparer)
	service.Slots = slots

	manager, err := New(service, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	manager.newID = func() (string, error) { return "queued", nil }

	queued, err := manager.Start(workflow.Request{
		Title: "Song",
		Destination: filepath.Join(root, "music"),
		URL: "https://example/song",
	})
	if err != nil {
		t.Fatal(err)
	}
	if queued.Status != importmodel.StatusQueued {
		t.Fatalf("first visible state drift: %+v", queued)
	}
	if err := manager.Cancel("queued"); err != nil {
		t.Fatal(err)
	}

	job := waitJob(t, manager, "queued", func(job importmodel.Job) bool {
		return job.Status == importmodel.StatusCancelled
	})
	if job.Status != importmodel.StatusCancelled {
		t.Fatalf("cancelled job drift: %+v", job)
	}
	if preparer.callCount() != 0 {
		t.Fatalf("provider executed despite queued cancellation: %d", preparer.callCount())
	}
}

func TestRetryCommitReturnsToReadyWhenLibraryCommitRemainsHeld(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "music")
	stage := persistStage(
		t,
		root,
		importmodel.Job{
			ID: "ready",
			Title: "Song",
			Status: importmodel.StatusReady,
			Destination: destination,
		},
		&staging.Manifest{
			Destination: destination,
			Outputs: []staging.Output{{
				RelativePath: "outputs/song.mp3",
				Filename: "song.mp3",
			}},
		},
		map[string][]byte{
			"outputs/song.mp3": []byte("audio"),
		},
	)

	service := workflow.Service{
		StagingRoot: root,
		Commit: func(
			_ string,
			_ string,
			_ string,
		) (importcommit.Result, error) {
			return importcommit.Result{}, errors.New("destination busy")
		},
	}
	manager, err := New(service, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	job, err := manager.RetryCommit("ready")
	if err == nil || !stringsContains(err.Error(), "destination busy") {
		t.Fatalf("commit hold not surfaced: job=%+v err=%v", job, err)
	}
	if job.Status != importmodel.StatusReady ||
		job.Stage != "Needs attention" {
		t.Fatalf("ready retry hold drift: %+v", job)
	}
	if _, statErr := os.Stat(stage); statErr != nil {
		t.Fatalf("held staging removed: %v", statErr)
	}
}

func TestDiscardRemovesOnlyDiscardableStaging(t *testing.T) {
	root := t.TempDir()
	failedStage := persistStage(
		t,
		root,
		importmodel.Job{
			ID: "failed",
			Title: "Song",
			Status: importmodel.StatusFailed,
		},
		nil,
		nil,
	)

	manager, err := New(workflow.Service{StagingRoot: root}, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	job, err := manager.Discard("failed")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != importmodel.StatusDiscarded {
		t.Fatalf("discarded state drift: %+v", job)
	}
	if _, err := os.Stat(failedStage); !os.IsNotExist(err) {
		t.Fatalf("discard did not remove staging: %v", err)
	}

	if _, err := manager.Discard("failed"); !errors.Is(err, ErrNotDiscardable) {
		t.Fatalf("terminal discarded job accepted again: %v", err)
	}
}

func TestStartRejectsDuplicateJobIdentity(t *testing.T) {
	root := t.TempDir()
	preparer := &preparerFixture{}
	service := testWorkflow(root, preparer)
	slots := make(chan struct{}, 1)
	slots <- struct{}{}
	service.Slots = slots

	manager, err := New(service, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	request := workflow.Request{
		ID: "same",
		Title: "Song",
		Destination: filepath.Join(root, "music"),
		URL: "https://example/song",
	}
	if _, err := manager.Start(request); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Start(request); err == nil {
		t.Fatal("duplicate job identity accepted")
	}
	if err := manager.Cancel("same"); err != nil {
		t.Fatal(err)
	}
}

func stringsContains(value, fragment string) bool {
	return strings.Contains(value, fragment)
}
