// Package workflow composes one provider-neutral VexStream import from admitted request through
// staged preparation and crash-recoverable Library commit.
//
// It does not own provider runtime discovery/repair, HTTP routing, source-folder configuration,
// cross-process routing, or long-lived UI state.
package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	importmodel "github.com/vgong24/VexstreamMusic/v2/desktop/imports"
	importcommit "github.com/vgong24/VexstreamMusic/v2/desktop/imports/commit"
	importprovider "github.com/vgong24/VexstreamMusic/v2/desktop/imports/provider"
	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

type Preparer interface {
	Prepare(
		context.Context,
		importprovider.PreparePlan,
		func(importprovider.PrepareEvent),
	) (importprovider.PrepareResult, error)
}

type CommitFunc func(
	jobID string,
	stageDir string,
	destination string,
) (importcommit.Result, error)

type DestinationAllowed func(string) bool
type AfterCommit func(importcommit.Result) error

type Request struct {
	ID             string
	Title          string
	Destination    string
	URL            string
	Mode           string
	WholeTitle     string
	ArtistOverride string
	Collection     string
	Genre          string
	Sections       []importprovider.PrepareSection
}

type Outcome struct {
	Job      importmodel.Job
	StageDir string
	Commit   *importcommit.Result
}

type UpdateFunc func(importmodel.Job)

type CommitHeldError struct {
	Err error
}

func (e *CommitHeldError) Error() string {
	if e == nil || e.Err == nil {
		return "Import commit is held"
	}
	return "Import commit is held: " + e.Err.Error()
}

func (e *CommitHeldError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type Service struct {
	StagingRoot        string
	Preparer           Preparer
	Commit             CommitFunc
	DestinationAllowed DestinationAllowed
	AfterCommit        AfterCommit
	Slots              chan struct{}
	Now                func() time.Time
	NewID              func() (string, error)
}

func (s Service) Run(
	ctx context.Context,
	request Request,
	onUpdate UpdateFunc,
) (Outcome, error) {
	if s.Preparer == nil {
		return Outcome{}, errors.New("Import workflow preparer is unavailable")
	}
	if s.Commit == nil {
		return Outcome{}, errors.New("Import workflow committer is unavailable")
	}
	root := strings.TrimSpace(s.StagingRoot)
	if root == "" {
		return Outcome{}, errors.New("Import workflow staging root is unavailable")
	}

	request.Destination = strings.TrimSpace(request.Destination)
	request.URL = strings.TrimSpace(request.URL)
	request.Title = strings.TrimSpace(request.Title)
	if request.Destination == "" {
		return Outcome{}, errors.New("Import destination is required")
	}
	if request.URL == "" {
		return Outcome{}, errors.New("Import source URL is required")
	}
	if s.DestinationAllowed == nil || !s.DestinationAllowed(request.Destination) {
		return Outcome{}, errors.New("Import destination is not an included VexStream music folder")
	}

	now := s.Now
	if now == nil {
		now = time.Now
	}
	newID := s.NewID
	if newID == nil {
		newID = randomID
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		var err error
		id, err = newID()
		if err != nil {
			return Outcome{}, fmt.Errorf("form import job id: %w", err)
		}
		id = strings.TrimSpace(id)
	}
	if id == "" {
		return Outcome{}, errors.New("Import job id is empty")
	}

	stage := filepath.Join(root, id)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return Outcome{}, fmt.Errorf("create import staging: %w", err)
	}
	cancelPath := filepath.Join(stage, ".cancel")
	_ = os.Remove(cancelPath)

	title := request.Title
	if title == "" {
		title = strings.TrimSpace(request.WholeTitle)
	}
	if title == "" {
		title = "Media import"
	}

	job := importmodel.Job{
		ID:          id,
		Title:       title,
		Status:      importmodel.StatusQueued,
		Stage:       "Waiting for preparation",
		Progress:    0,
		Detail:      "Queued — media will remain in reversible staging until Library commit.",
		StartedAt:   now().UTC().Format(time.RFC3339Nano),
		Destination: request.Destination,
	}
	if err := persist(stage, job); err != nil {
		_ = os.RemoveAll(stage)
		return Outcome{}, fmt.Errorf("persist queued import job: %w", err)
	}
	notify(onUpdate, job)

	update := func(mutator func(*importmodel.Job)) error {
		mutator(&job)
		if err := persist(stage, job); err != nil {
			return err
		}
		notify(onUpdate, job)
		return nil
	}

	if s.Slots != nil {
		select {
		case s.Slots <- struct{}{}:
			defer func() { <-s.Slots }()
		case <-ctx.Done():
			_ = update(func(job *importmodel.Job) {
				job.Status = importmodel.StatusCancelled
				job.Stage = "Cancelled"
				job.Progress = 100
				job.Detail = "Import was cancelled before provider preparation started."
				job.CompletedAt = now().UTC().Format(time.RFC3339Nano)
			})
			outcome := Outcome{Job: cloneJob(job), StageDir: stage}
			_ = os.RemoveAll(stage)
			return outcome, importprovider.ErrPrepareCancelled
		}
	}

	if err := update(func(job *importmodel.Job) {
		job.Status = importmodel.StatusRunning
		job.Stage = "Preparing media"
		job.Progress = 1
		job.Detail = "Downloading and preparing media inside temporary staging."
	}); err != nil {
		return Outcome{Job: cloneJob(job), StageDir: stage}, err
	}

	watchDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = os.WriteFile(cancelPath, []byte("cancel\n"), 0o644)
		case <-watchDone:
		}
	}()
	defer close(watchDone)

	var eventPersistenceErr error
	result, prepareErr := s.Preparer.Prepare(
		ctx,
		importprovider.PreparePlan{
			URL:              request.URL,
			Destination:      request.Destination,
			StagingDirectory: stage,
			CancelPath:       cancelPath,
			Mode:             request.Mode,
			WholeTitle:       request.WholeTitle,
			ArtistOverride:   request.ArtistOverride,
			Collection:       request.Collection,
			Genre:            request.Genre,
			Sections:         append([]importprovider.PrepareSection(nil), request.Sections...),
		},
		func(event importprovider.PrepareEvent) {
			if eventPersistenceErr != nil {
				return
			}
			eventPersistenceErr = update(func(job *importmodel.Job) {
				switch event.Type {
				case "progress":
					job.Status = importmodel.StatusRunning
					if strings.TrimSpace(event.Stage) != "" {
						job.Stage = strings.TrimSpace(event.Stage)
						job.Detail = strings.TrimSpace(event.Stage)
					}
					job.Progress = clamp(event.Progress, 0, 100)
				case "error":
					if event.Problem != nil {
						job.Problem = problemMap(*event.Problem)
					}
				case "cancelled":
					job.Status = importmodel.StatusCancelling
					job.Stage = "Cancelling…"
					job.Detail = strings.TrimSpace(event.Message)
				}
			})
		},
	)

	if eventPersistenceErr != nil {
		return Outcome{Job: cloneJob(job), StageDir: stage},
			fmt.Errorf("persist import progress: %w", eventPersistenceErr)
	}

	if ctx.Err() != nil || errors.Is(prepareErr, importprovider.ErrPrepareCancelled) {
		_ = update(func(job *importmodel.Job) {
			job.Status = importmodel.StatusCancelled
			job.Stage = "Cancelled"
			job.Progress = 100
			job.Detail = "Import preparation was cancelled."
			job.CompletedAt = now().UTC().Format(time.RFC3339Nano)
		})
		outcome := Outcome{Job: cloneJob(job), StageDir: stage}
		_ = os.RemoveAll(stage)
		return outcome, importprovider.ErrPrepareCancelled
	}
	if prepareErr != nil {
		_ = update(func(job *importmodel.Job) {
			job.Status = importmodel.StatusFailed
			job.Stage = "Import failed"
			job.Progress = 100
			job.Detail = prepareErr.Error()
			job.CompletedAt = now().UTC().Format(time.RFC3339Nano)
			if problem, ok := importprovider.ProblemFromError(prepareErr); ok {
				job.Problem = problemMap(problem)
			}
		})
		return Outcome{Job: cloneJob(job), StageDir: stage}, prepareErr
	}
	if !result.Staged || len(result.Outputs) == 0 {
		err := errors.New("Import preparation returned no staged outputs")
		_ = update(func(job *importmodel.Job) {
			job.Status = importmodel.StatusFailed
			job.Stage = "Import failed"
			job.Progress = 100
			job.Detail = err.Error()
			job.CompletedAt = now().UTC().Format(time.RFC3339Nano)
		})
		return Outcome{Job: cloneJob(job), StageDir: stage}, err
	}

	if err := update(func(job *importmodel.Job) {
		job.Status = importmodel.StatusCommitting
		job.Stage = "Adding to library"
		job.Progress = 100
		job.Detail = fmt.Sprintf(
			"%d prepared file(s). Finishing Library commit…",
			len(result.Outputs),
		)
		job.OutputCount = len(result.Outputs)
	}); err != nil {
		return Outcome{Job: cloneJob(job), StageDir: stage}, err
	}

	committed, commitErr := s.Commit(id, stage, request.Destination)
	if commitErr != nil {
		_ = update(func(job *importmodel.Job) {
			job.Status = importmodel.StatusReady
			job.Stage = "Needs attention"
			job.Progress = 100
			job.Detail = "Preparation finished, but Library commit is held. Retry or discard the staged files. " + commitErr.Error()
		})
		return Outcome{Job: cloneJob(job), StageDir: stage},
			&CommitHeldError{Err: commitErr}
	}

	refreshWarning := ""
	if s.AfterCommit != nil {
		if err := s.AfterCommit(committed); err != nil {
			refreshWarning = err.Error()
		}
	}
	if err := update(func(job *importmodel.Job) {
		job.Status = importmodel.StatusDone
		job.Stage = "Added to library"
		job.Progress = 100
		job.Detail = fmt.Sprintf("%d file(s) added to the Library.", committed.Added)
		if refreshWarning != "" {
			job.Detail += " Library refresh is pending: " + refreshWarning
		}
		job.OutputCount = committed.Added
		job.TrackRefs = append([]string(nil), committed.TrackRefs...)
		job.CompletedAt = now().UTC().Format(time.RFC3339Nano)
	}); err != nil {
		return Outcome{
			Job:      cloneJob(job),
			StageDir: stage,
			Commit:   &committed,
		}, err
	}

	outcome := Outcome{
		Job:      cloneJob(job),
		StageDir: stage,
		Commit:   &committed,
	}
	if err := os.RemoveAll(stage); err != nil {
		return outcome, fmt.Errorf("clean committed import staging: %w", err)
	}
	return outcome, nil
}

func persist(stage string, job importmodel.Job) error {
	return persistence.SaveJSONAtomic(filepath.Join(stage, "job.json"), job)
}

func notify(callback UpdateFunc, job importmodel.Job) {
	if callback != nil {
		callback(cloneJob(job))
	}
}

func problemMap(problem importprovider.Problem) map[string]any {
	return map[string]any{
		"schemaVersion":   problem.SchemaVersion,
		"provider":        problem.Provider,
		"stage":           problem.Stage,
		"code":            problem.Code,
		"title":           problem.Title,
		"message":         problem.Message,
		"retryable":       problem.Retryable,
		"suggestedAction": problem.SuggestedAction,
		"technical":       problem.Technical,
		"observedAt":      problem.ObservedAt,
	}
}

func cloneJob(job importmodel.Job) importmodel.Job {
	job.TrackRefs = append([]string(nil), job.TrackRefs...)
	if job.Problem != nil {
		copy := make(map[string]any, len(job.Problem))
		for key, value := range job.Problem {
			copy[key] = value
		}
		job.Problem = copy
	}
	return job
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func randomID() (string, error) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
