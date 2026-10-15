package staging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	importmodel "github.com/vgong24/VexstreamMusic/v2/desktop/imports"
	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

func stageJob(t *testing.T, root, name string, job importmodel.Job, manifest *Manifest, outputs map[string][]byte) string {
	t.Helper()
	stage := filepath.Join(root, name)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := persistence.SaveJSONAtomic(filepath.Join(stage, "job.json"), job); err != nil {
		t.Fatal(err)
	}
	if manifest != nil {
		if err := persistence.SaveJSONAtomic(filepath.Join(stage, "manifest.json"), manifest); err != nil {
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

func issueCodes(inspection Inspection) []string {
	out := make([]string, len(inspection.Issues))
	for i, item := range inspection.Issues {
		out[i] = item.Code
	}
	return out
}

func TestMissingStagingRootIsEmptyNotFailure(t *testing.T) {
	got, err := Inspect(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Records) != 0 || len(got.Issues) != 0 {
		t.Fatalf("missing root drift: %+v", got)
	}
}

func TestReadyJobWithCompleteManifestIsPreparedForHumanRetryOrDiscard(t *testing.T) {
	root := t.TempDir()
	stageJob(
		t,
		root,
		"ready",
		importmodel.Job{ID: "ready", Status: importmodel.StatusReady, Destination: filepath.Join(root, "music")},
		&Manifest{
			Destination: filepath.Join(root, "music"),
			Outputs: []Output{{RelativePath: "out/one.mp3", Filename: "one.mp3"}},
		},
		map[string][]byte{"out/one.mp3": []byte("audio")},
	)

	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Records) != 1 || !got.Records[0].Prepared {
		t.Fatalf("ready staged inspection drift: %+v", got)
	}
	if got.Records[0].Disposition != importmodel.RestartReadyStaging {
		t.Fatalf("ready disposition drift: %s", got.Records[0].Disposition)
	}
}

func TestCommittingJobIsNeverPromotedToReadyEvenWithCompleteManifest(t *testing.T) {
	root := t.TempDir()
	stageJob(
		t,
		root,
		"committing",
		importmodel.Job{ID: "c", Status: importmodel.StatusCommitting, Destination: filepath.Join(root, "music")},
		&Manifest{
			Destination: filepath.Join(root, "music"),
			Outputs: []Output{{RelativePath: "one.mp3"}},
		},
		map[string][]byte{"one.mp3": []byte("audio")},
	)
	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	record := got.Records[0]
	if record.Disposition != importmodel.RestartCommitUncertain || !record.Prepared {
		t.Fatalf("committing recovery drift: %+v", record)
	}
	if !strings.Contains(strings.Join(issueCodes(got), ","), "COMMIT_RECONCILIATION_REQUIRED") {
		t.Fatalf("commit uncertainty not surfaced: %+v", got.Issues)
	}
}

func TestMissingCommittingOutputStrengthensUncertaintyInsteadOfAutoReplay(t *testing.T) {
	root := t.TempDir()
	stageJob(
		t,
		root,
		"committing",
		importmodel.Job{ID: "c", Status: importmodel.StatusCommitting},
		&Manifest{Outputs: []Output{{RelativePath: "moved.mp3"}, {RelativePath: "still.mp3"}}},
		map[string][]byte{"still.mp3": []byte("remaining")},
	)
	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	record := got.Records[0]
	if record.Prepared || len(record.MissingOutputRefs) != 1 || record.MissingOutputRefs[0] != "moved.mp3" {
		t.Fatalf("missing output evidence drift: %+v", record)
	}
	codes := strings.Join(issueCodes(got), ",")
	if !strings.Contains(codes, "COMMIT_OUTPUTS_PARTIALLY_ABSENT") {
		t.Fatalf("partial commit evidence not surfaced: %s", codes)
	}
}

func TestInterruptedPreparationRemainsHeld(t *testing.T) {
	root := t.TempDir()
	for _, status := range []importmodel.Status{
		importmodel.StatusQueued,
		importmodel.StatusRunning,
		importmodel.StatusCancelling,
	} {
		stageJob(t, root, string(status), importmodel.Job{ID: string(status), Status: status}, nil, nil)
	}
	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Records) != 3 {
		t.Fatalf("interrupted records missing: %+v", got)
	}
	for _, record := range got.Records {
		if record.Disposition != importmodel.RestartInterruptedPreparation {
			t.Fatalf("interrupted disposition drift: %+v", record)
		}
	}
}

func TestUnsafeManifestPathsFailPreparedInspection(t *testing.T) {
	root := t.TempDir()
	stageJob(
		t,
		root,
		"unsafe",
		importmodel.Job{ID: "unsafe", Status: importmodel.StatusReady},
		&Manifest{Outputs: []Output{{RelativePath: "../escape.mp3"}}},
		nil,
	)
	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Records[0].Prepared {
		t.Fatal("unsafe manifest was marked prepared")
	}
	if !strings.Contains(strings.Join(issueCodes(got), ","), "OUTPUT_PATH_UNSAFE") {
		t.Fatalf("unsafe path issue missing: %+v", got.Issues)
	}
}

func TestCorruptJobAndMissingManifestAreVisibleRatherThanSilentlySkipped(t *testing.T) {
	root := t.TempDir()
	corrupt := filepath.Join(root, "corrupt")
	if err := os.MkdirAll(corrupt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corrupt, "job.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	stageJob(t, root, "ready-no-manifest", importmodel.Job{ID: "r", Status: importmodel.StatusReady}, nil, nil)

	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	codes := strings.Join(issueCodes(got), ",")
	if !strings.Contains(codes, "JOB_STATE_INVALID") || !strings.Contains(codes, "MANIFEST_MISSING") {
		t.Fatalf("staging visibility drift: %+v", got.Issues)
	}
	if len(got.Records) != 1 || got.Records[0].Job.ID != "r" {
		t.Fatalf("valid persisted job lost: %+v", got.Records)
	}
}

func TestTerminalStagingResidueIsVisibleForLaterCleanupPolicy(t *testing.T) {
	root := t.TempDir()
	stageJob(t, root, "done", importmodel.Job{ID: "d", Status: importmodel.StatusDone}, nil, nil)
	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Records[0].Disposition != importmodel.RestartTerminal {
		t.Fatalf("terminal disposition drift: %+v", got.Records[0])
	}
	if !strings.Contains(strings.Join(issueCodes(got), ","), "TERMINAL_STAGE_REMAINS") {
		t.Fatalf("terminal residue issue missing: %+v", got.Issues)
	}
}
