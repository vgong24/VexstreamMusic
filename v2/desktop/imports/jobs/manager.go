// Package jobs owns in-process Import runtime coordination and restart visibility.
//
// It wraps the provider-neutral workflow with stable job identity, bounded concurrency, cancellation,
// retry/discard, and explicit commit recovery. It never auto-replays uncertain commit effects.
package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	importmodel "github.com/vgong24/VexstreamMusic/v2/desktop/imports"
	importcommit "github.com/vgong24/VexstreamMusic/v2/desktop/imports/commit"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/staging"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/workflow"
	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

var (
	ErrNotFound         = errors.New("Import job not found")
	ErrNotCancellable   = errors.New("Import job is not cancellable")
	ErrNotDiscardable   = errors.New("Import job is not discardable")
	ErrNotRetryable     = errors.New("Import job is not ready for commit retry")
	ErrRecoveryNotReady = errors.New("Import commit recovery is not safely resumable")
)

type Snapshot struct {
	Groups   importmodel.Groups                         `json:"groups"`
	Recovery map[string]importmodel.RestartDisposition `json:"recovery,omitempty"`
	Issues   []staging.Issue                           `json:"issues,omitempty"`
}

type Manager struct {
	mu       sync.RWMutex
	workflow workflow.Service
	jobs     map[string]importmodel.Job
	stages   map[string]string
	recovery map[string]importmodel.RestartDisposition
	issues   []staging.Issue
	cancels  map[string]context.CancelFunc
	rootCtx  context.Context
	close    context.CancelFunc
	now      func() time.Time
	newID    func() (string, error)
}

func New(
	service workflow.Service,
	maxConcurrent int,
) (*Manager, error) {
	if strings.TrimSpace(service.StagingRoot) == "" {
		return nil, errors.New("Import Manager requires staging root")
	}
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}
	if service.Slots == nil {
		service.Slots = make(chan struct{}, maxConcurrent)
	}

	rootCtx, closeManager := context.WithCancel(context.Background())
	manager := &Manager{
		workflow: service,
		jobs: map[string]importmodel.Job{},
		stages: map[string]string{},
		recovery: map[string]importmodel.RestartDisposition{},
		issues: []staging.Issue{},
		cancels: map[string]context.CancelFunc{},
		rootCtx: rootCtx,
		close: closeManager,
		now: time.Now,
		newID: randomID,
	}

	inspection, err := staging.Inspect(service.StagingRoot)
	if err != nil {
		closeManager()
		return nil, err
	}
	manager.issues = append(manager.issues, inspection.Issues...)
	for _, record := range inspection.Records {
		job := cloneJob(record.Job)
		manager.stages[job.ID] = record.StageDir
		manager.recovery[job.ID] = record.Disposition

		if record.Disposition == importmodel.RestartInterruptedPreparation {
			job.Status = importmodel.StatusFailed
			job.Stage = "Interrupted"
			job.Progress = 100
			job.Detail = "A prior VexStream process ended before preparation completed. Retry from the source or discard this staging."
			if job.CompletedAt == "" {
				job.CompletedAt = manager.now().UTC().Format(time.RFC3339Nano)
			}
			_ = persistJob(record.StageDir, job)
		}
		manager.jobs[job.ID] = job
	}

	return manager, nil
}

func (m *Manager) Close() {
	if m == nil || m.close == nil {
		return
	}
	m.close()
}

func (m *Manager) Start(request workflow.Request) (importmodel.Job, error) {
	if m == nil {
		return importmodel.Job{}, errors.New("Import Manager is unavailable")
	}

	id := strings.TrimSpace(request.ID)
	if id == "" {
		var err error
		id, err = m.newID()
		if err != nil {
			return importmodel.Job{}, err
		}
		id = strings.TrimSpace(id)
	}
	if id == "" {
		return importmodel.Job{}, errors.New("Import job id is empty")
	}
	request.ID = id

	m.mu.Lock()
	if _, exists := m.jobs[id]; exists {
		m.mu.Unlock()
		return importmodel.Job{}, fmt.Errorf("Import job %q already exists", id)
	}
	stage := filepath.Join(m.workflow.StagingRoot, id)
	m.stages[id] = stage
	ctx, cancel := context.WithCancel(m.rootCtx)
	m.cancels[id] = cancel
	m.mu.Unlock()

	first := make(chan importmodel.Job, 1)
	finished := make(chan struct {
		outcome workflow.Outcome
		err     error
	}, 1)

	go func() {
		outcome, err := m.workflow.Run(
			ctx,
			request,
			func(job importmodel.Job) {
				m.mu.Lock()
				m.jobs[id] = cloneJob(job)
				delete(m.recovery, id)
				m.mu.Unlock()
				select {
				case first <- cloneJob(job):
				default:
				}
			},
		)

		m.mu.Lock()
		if outcome.Job.ID != "" {
			m.jobs[id] = cloneJob(outcome.Job)
		}
		delete(m.cancels, id)
		m.mu.Unlock()
		finished <- struct {
			outcome workflow.Outcome
			err     error
		}{outcome: outcome, err: err}
	}()

	select {
	case job := <-first:
		return job, nil
	case result := <-finished:
		m.mu.Lock()
		delete(m.cancels, id)
		if result.outcome.Job.ID == "" {
			delete(m.stages, id)
		}
		m.mu.Unlock()
		if result.err != nil {
			return cloneJob(result.outcome.Job), result.err
		}
		return cloneJob(result.outcome.Job), nil
	case <-m.rootCtx.Done():
		return importmodel.Job{}, errors.New("Import Manager is closed")
	}
}

func (m *Manager) Snapshot() Snapshot {
	if m == nil {
		return Snapshot{
			Groups: importmodel.Classify(nil),
			Recovery: map[string]importmodel.RestartDisposition{},
			Issues: []staging.Issue{},
		}
	}

	m.mu.RLock()
	rows := make([]importmodel.Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		rows = append(rows, cloneJob(job))
	}
	recovery := make(map[string]importmodel.RestartDisposition, len(m.recovery))
	for id, disposition := range m.recovery {
		recovery[id] = disposition
	}
	issues := append([]staging.Issue(nil), m.issues...)
	m.mu.RUnlock()

	return Snapshot{
		Groups: importmodel.Classify(rows),
		Recovery: recovery,
		Issues: issues,
	}
}

func (m *Manager) Cancel(id string) error {
	id = strings.TrimSpace(id)
	m.mu.RLock()
	job, ok := m.jobs[id]
	cancel := m.cancels[id]
	stage := m.stages[id]
	m.mu.RUnlock()
	if !ok {
		return ErrNotFound
	}
	if !importmodel.CanCancel(job.Status) || cancel == nil {
		return ErrNotCancellable
	}

	if stage != "" {
		_ = os.MkdirAll(stage, 0o755)
		_ = os.WriteFile(
			filepath.Join(stage, ".cancel"),
			[]byte("cancel\n"),
			0o644,
		)
	}
	cancel()
	return nil
}

func (m *Manager) Discard(id string) (importmodel.Job, error) {
	id = strings.TrimSpace(id)
	m.mu.Lock()
	job, ok := m.jobs[id]
	stage := m.stages[id]
	if !ok {
		m.mu.Unlock()
		return importmodel.Job{}, ErrNotFound
	}
	if !importmodel.CanDiscard(job.Status) {
		m.mu.Unlock()
		return cloneJob(job), ErrNotDiscardable
	}

	job.Status = importmodel.StatusDiscarded
	job.Stage = "Discarded"
	job.Progress = 100
	job.Detail = "Temporary import staging was discarded."
	if job.CompletedAt == "" {
		job.CompletedAt = m.now().UTC().Format(time.RFC3339Nano)
	}
	m.jobs[id] = cloneJob(job)
	delete(m.recovery, id)
	m.mu.Unlock()

	if stage != "" {
		_ = os.RemoveAll(stage)
	}
	return cloneJob(job), nil
}

func (m *Manager) RetryCommit(id string) (importmodel.Job, error) {
	id = strings.TrimSpace(id)
	m.mu.RLock()
	job, ok := m.jobs[id]
	stage := m.stages[id]
	m.mu.RUnlock()
	if !ok {
		return importmodel.Job{}, ErrNotFound
	}
	if job.Status != importmodel.StatusReady {
		return cloneJob(job), ErrNotRetryable
	}
	return m.commit(id, stage, false)
}

func (m *Manager) RecoverCommit(id string) (importmodel.Job, error) {
	id = strings.TrimSpace(id)
	m.mu.RLock()
	job, ok := m.jobs[id]
	stage := m.stages[id]
	disposition := m.recovery[id]
	m.mu.RUnlock()
	if !ok {
		return importmodel.Job{}, ErrNotFound
	}
	if disposition != importmodel.RestartCommitUncertain ||
		job.Status != importmodel.StatusCommitting {
		return cloneJob(job), ErrRecoveryNotReady
	}

	if _, found, err := importcommit.Load(stage); err != nil {
		return cloneJob(job), err
	} else if !found {
		return cloneJob(job), ErrRecoveryNotReady
	}
	return m.commit(id, stage, true)
}

func (m *Manager) commit(
	id string,
	stage string,
	recovery bool,
) (importmodel.Job, error) {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return importmodel.Job{}, ErrNotFound
	}
	job.Status = importmodel.StatusCommitting
	job.Stage = "Adding to library"
	job.Progress = 100
	if recovery {
		job.Detail = "Resuming the existing crash-recovery commit journal."
	} else {
		job.Detail = "Retrying Library commit from prepared staging."
	}
	m.jobs[id] = cloneJob(job)
	m.mu.Unlock()
	if err := persistJob(stage, job); err != nil {
		return cloneJob(job), err
	}

	result, err := m.workflow.Commit(id, stage, job.Destination)
	if err != nil {
		m.mu.Lock()
		current := m.jobs[id]
		if recovery {
			current.Status = importmodel.StatusCommitting
			current.Stage = "Recovery held"
			current.Detail = "Commit recovery remains unresolved. " + err.Error()
		} else {
			current.Status = importmodel.StatusReady
			current.Stage = "Needs attention"
			current.Detail = "Library commit is still held. " + err.Error()
		}
		m.jobs[id] = cloneJob(current)
		m.mu.Unlock()
		_ = persistJob(stage, current)
		return cloneJob(current), err
	}

	m.mu.Lock()
	current := m.jobs[id]
	current.Status = importmodel.StatusDone
	current.Stage = "Added to library"
	current.Progress = 100
	current.OutputCount = result.Added
	current.TrackRefs = append([]string(nil), result.TrackRefs...)
	current.Detail = fmt.Sprintf("%d file(s) added to the Library.", result.Added)
	current.CompletedAt = m.now().UTC().Format(time.RFC3339Nano)
	m.jobs[id] = cloneJob(current)
	delete(m.recovery, id)
	m.mu.Unlock()
	if err := persistJob(stage, current); err != nil {
		return cloneJob(current), err
	}
	if err := os.RemoveAll(stage); err != nil {
		return cloneJob(current), err
	}
	return cloneJob(current), nil
}

func persistJob(stage string, job importmodel.Job) error {
	return persistence.SaveJSONAtomic(filepath.Join(stage, "job.json"), job)
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

func randomID() (string, error) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

