// Package imports owns provider-independent VexStream import-job lifecycle meaning.
//
// It deliberately does not own:
// - provider inspection/search/download execution;
// - process supervision or cancellation mechanics;
// - staging filesystem custody;
// - duplicate scoring;
// - Library reconciliation;
// - cross-process/session routing.
//
// Those concrete owners may observe or realize this lifecycle, but they must not invent a different
// meaning for the same persisted job status.
package imports

import (
	"sort"
	"strings"
)

type Status string

const (
	StatusQueued     Status = "queued"
	StatusRunning    Status = "running"
	StatusCancelling Status = "cancelling"
	StatusReady      Status = "ready"
	StatusCommitting Status = "committing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
	StatusDiscarded  Status = "discarded"
)

type Group string

const (
	GroupActive    Group = "active"
	GroupReady     Group = "ready"
	GroupCompleted Group = "completed"
)

type RestartDisposition string

const (
	RestartInterruptedPreparation RestartDisposition = "INTERRUPTED_PREPARATION"
	RestartReadyStaging           RestartDisposition = "READY_STAGING"
	RestartCommitUncertain        RestartDisposition = "COMMIT_UNCERTAIN"
	RestartTerminal               RestartDisposition = "TERMINAL"
)

type Job struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Status      Status         `json:"status"`
	Stage       string         `json:"stage,omitempty"`
	Progress    int            `json:"progress,omitempty"`
	Detail      string         `json:"detail,omitempty"`
	StartedAt   string         `json:"startedAt,omitempty"`
	CompletedAt string         `json:"completedAt,omitempty"`
	Destination string         `json:"destination,omitempty"`
	OutputCount int            `json:"outputCount,omitempty"`
	TrackRefs   []string       `json:"trackIds,omitempty"`
	Problem     map[string]any `json:"problem,omitempty"`
}

type Groups struct {
	Active    []Job `json:"active"`
	Ready     []Job `json:"ready"`
	Completed []Job `json:"completed"`
}

func KnownStatus(status Status) bool {
	switch status {
	case StatusQueued,
		StatusRunning,
		StatusCancelling,
		StatusReady,
		StatusCommitting,
		StatusDone,
		StatusFailed,
		StatusCancelled,
		StatusDiscarded:
		return true
	default:
		return false
	}
}

func IsTerminal(status Status) bool {
	switch status {
	case StatusDone, StatusFailed, StatusCancelled, StatusDiscarded:
		return true
	default:
		return false
	}
}

// GroupFor preserves the V3 activity grouping contract.
//
// "committing" remains active because Library acceptance has not completed.
// "ready" is separate because staged bytes exist but require a human-visible retry/discard choice.
func GroupFor(status Status) Group {
	if status == StatusReady {
		return GroupReady
	}
	if IsTerminal(status) {
		return GroupCompleted
	}
	return GroupActive
}

// CanCancel answers whether the current local endpoint accepts a cancellation request.
//
// "cancelling" remains accepted/idempotent at the semantic admission layer while concrete process
// termination remains provider/runtime work.
func CanCancel(status Status) bool {
	switch status {
	case StatusQueued, StatusRunning, StatusCancelling:
		return true
	default:
		return false
	}
}

// CanDiscard answers whether staged artifacts may be explicitly abandoned without Library commit.
func CanDiscard(status Status) bool {
	return status == StatusReady || status == StatusFailed
}

// CanAttemptCommit preserves the V3 local commit admission.
//
// StatusCommitting is intentionally admitted here because an in-process automatic commit may already
// have set that status before calling the commit function. This does NOT mean an interrupted
// committing job is safe to auto-retry after restart.
func CanAttemptCommit(status Status) bool {
	return status == StatusReady || status == StatusCommitting
}

// RestartClass makes V3's previously implicit recovery ambiguity explicit.
//
// A persisted "committing" status is not equivalent to "ready": the process could have stopped after
// moving some outputs but before catalog metadata committed. V4 must inspect/reconcile that boundary
// before replay rather than auto-commit.
func RestartClass(status Status) RestartDisposition {
	switch status {
	case StatusReady:
		return RestartReadyStaging
	case StatusCommitting:
		return RestartCommitUncertain
	case StatusQueued, StatusRunning, StatusCancelling:
		return RestartInterruptedPreparation
	case StatusDone, StatusFailed, StatusCancelled, StatusDiscarded:
		return RestartTerminal
	default:
		return RestartInterruptedPreparation
	}
}

func Classify(jobs []Job) Groups {
	groups := Groups{
		Active:    []Job{},
		Ready:     []Job{},
		Completed: []Job{},
	}
	for _, job := range jobs {
		copy := cloneJob(job)
		switch GroupFor(job.Status) {
		case GroupReady:
			groups.Ready = append(groups.Ready, copy)
		case GroupCompleted:
			groups.Completed = append(groups.Completed, copy)
		default:
			groups.Active = append(groups.Active, copy)
		}
	}

	sort.SliceStable(groups.Active, func(i, j int) bool {
		left, right := activeRank(groups.Active[i].Status), activeRank(groups.Active[j].Status)
		if left != right {
			return left < right
		}
		return strings.Compare(groups.Active[i].StartedAt, groups.Active[j].StartedAt) < 0
	})
	sort.SliceStable(groups.Ready, func(i, j int) bool {
		return strings.Compare(groups.Ready[i].StartedAt, groups.Ready[j].StartedAt) > 0
	})
	sort.SliceStable(groups.Completed, func(i, j int) bool {
		return strings.Compare(groups.Completed[i].CompletedAt, groups.Completed[j].CompletedAt) > 0
	})
	return groups
}

func activeRank(status Status) int {
	switch status {
	case StatusRunning, StatusCancelling, StatusCommitting:
		return 0
	case StatusQueued:
		return 1
	default:
		return 2
	}
}

func cloneJob(job Job) Job {
	job.TrackRefs = append([]string(nil), job.TrackRefs...)
	if job.Problem != nil {
		job.Problem = cloneMap(job.Problem)
	}
	return job
}

func cloneMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
