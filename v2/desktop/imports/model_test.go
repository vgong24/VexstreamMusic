package imports

import (
	"reflect"
	"testing"
)

func TestV3StatusVocabularyIsExplicit(t *testing.T) {
	for _, status := range []Status{
		StatusQueued,
		StatusRunning,
		StatusCancelling,
		StatusReady,
		StatusCommitting,
		StatusDone,
		StatusFailed,
		StatusCancelled,
		StatusDiscarded,
	} {
		if !KnownStatus(status) {
			t.Fatalf("known status rejected: %q", status)
		}
	}
	if KnownStatus("mystery") {
		t.Fatal("unknown status accepted")
	}
}

func TestCapabilityAdmissionPreservesV3LifecycleMeaning(t *testing.T) {
	for _, status := range []Status{StatusQueued, StatusRunning, StatusCancelling} {
		if !CanCancel(status) {
			t.Fatalf("%q should be cancellable", status)
		}
	}
	for _, status := range []Status{StatusReady, StatusFailed} {
		if !CanDiscard(status) {
			t.Fatalf("%q should be discardable", status)
		}
	}
	for _, status := range []Status{StatusReady, StatusCommitting} {
		if !CanAttemptCommit(status) {
			t.Fatalf("%q should be accepted by local commit", status)
		}
	}
	for _, status := range []Status{StatusDone, StatusCancelled, StatusDiscarded} {
		if CanCancel(status) || CanDiscard(status) || CanAttemptCommit(status) {
			t.Fatalf("terminal status exposes invalid action: %q", status)
		}
	}
}

func TestRestartDispositionKeepsCommittingDistinctFromReady(t *testing.T) {
	if got := RestartClass(StatusReady); got != RestartReadyStaging {
		t.Fatalf("ready restart drift: %s", got)
	}
	if got := RestartClass(StatusCommitting); got != RestartCommitUncertain {
		t.Fatalf("committing restart must stay uncertain: %s", got)
	}
	for _, status := range []Status{StatusQueued, StatusRunning, StatusCancelling} {
		if got := RestartClass(status); got != RestartInterruptedPreparation {
			t.Fatalf("%q restart drift: %s", status, got)
		}
	}
	for _, status := range []Status{StatusDone, StatusFailed, StatusCancelled, StatusDiscarded} {
		if got := RestartClass(status); got != RestartTerminal {
			t.Fatalf("%q terminal restart drift: %s", status, got)
		}
	}
}

func TestClassificationPreservesV3GroupsAndSortOrder(t *testing.T) {
	jobs := []Job{
		{ID: "queued-late", Status: StatusQueued, StartedAt: "2026-10-14T00:00:09Z"},
		{ID: "running-late", Status: StatusRunning, StartedAt: "2026-10-14T00:00:08Z"},
		{ID: "committing-early", Status: StatusCommitting, StartedAt: "2026-10-14T00:00:01Z"},
		{ID: "ready-old", Status: StatusReady, StartedAt: "2026-10-14T00:00:02Z"},
		{ID: "ready-new", Status: StatusReady, StartedAt: "2026-10-14T00:00:07Z"},
		{ID: "done-old", Status: StatusDone, CompletedAt: "2026-10-14T00:00:03Z"},
		{ID: "failed-new", Status: StatusFailed, CompletedAt: "2026-10-14T00:00:10Z"},
		{ID: "cancelled", Status: StatusCancelled, CompletedAt: "2026-10-14T00:00:05Z"},
	}
	got := Classify(jobs)

	ids := func(rows []Job) []string {
		out := make([]string, len(rows))
		for i, row := range rows {
			out[i] = row.ID
		}
		return out
	}

	if want := []string{"committing-early", "running-late", "queued-late"}; !reflect.DeepEqual(ids(got.Active), want) {
		t.Fatalf("active grouping/sort drift: got=%v want=%v", ids(got.Active), want)
	}
	if want := []string{"ready-new", "ready-old"}; !reflect.DeepEqual(ids(got.Ready), want) {
		t.Fatalf("ready grouping/sort drift: got=%v want=%v", ids(got.Ready), want)
	}
	if want := []string{"failed-new", "cancelled", "done-old"}; !reflect.DeepEqual(ids(got.Completed), want) {
		t.Fatalf("completed grouping/sort drift: got=%v want=%v", ids(got.Completed), want)
	}
}

func TestClassificationReturnsIndependentCopies(t *testing.T) {
	input := []Job{{
		ID:        "ready",
		Status:    StatusReady,
		TrackRefs: []string{"a"},
		Problem:   map[string]any{"code": "X"},
	}}
	got := Classify(input)
	got.Ready[0].TrackRefs[0] = "mutated"
	got.Ready[0].Problem["code"] = "Y"

	if input[0].TrackRefs[0] != "a" || input[0].Problem["code"] != "X" {
		t.Fatalf("classification mutated caller truth: %+v", input[0])
	}
}

func TestUnknownStatusStaysActiveAndRecoveryHeldInsteadOfInventingTerminalMeaning(t *testing.T) {
	status := Status("future-status")
	if GroupFor(status) != GroupActive {
		t.Fatalf("unknown status should remain visible/active, got %s", GroupFor(status))
	}
	if RestartClass(status) != RestartInterruptedPreparation {
		t.Fatalf("unknown restart state should stay held, got %s", RestartClass(status))
	}
}
