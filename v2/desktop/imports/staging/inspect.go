// Package staging inspects persisted VexStream import staging without replaying or mutating it.
//
// Inspection is intentionally conservative. A previous process may have stopped between moving
// prepared media and committing catalog metadata, so a persisted "committing" job is evidence to
// reconcile, never an instruction to auto-replay.
package staging

const ManifestSchema = "vexstream.import-stage/v1"

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	importmodel "github.com/vgong24/VexstreamMusic/v2/desktop/imports"
	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

type Output struct {
	RelativePath string         `json:"relativePath"`
	Filename     string         `json:"filename"`
	CatalogEntry map[string]any `json:"catalogEntry"`
}

type Manifest struct {
	SchemaVersion string   `json:"schemaVersion"`
	Destination   string   `json:"destination"`
	Outputs       []Output `json:"outputs"`
}

type Issue struct {
	StageDir string `json:"stageDir"`
	Code     string `json:"code"`
	Detail   string `json:"detail"`
}

type Record struct {
	StageDir          string                         `json:"stageDir"`
	Job               importmodel.Job                `json:"job"`
	Disposition       importmodel.RestartDisposition `json:"disposition"`
	Manifest          *Manifest                      `json:"manifest,omitempty"`
	Prepared          bool                           `json:"prepared"`
	MissingOutputRefs []string                       `json:"missingOutputRefs,omitempty"`
}

type Inspection struct {
	Records []Record `json:"records"`
	Issues  []Issue  `json:"issues"`
}

func Inspect(root string) (Inspection, error) {
	inspection := Inspection{Records: []Record{}, Issues: []Issue{}}
	rows, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return inspection, nil
	}
	if err != nil {
		return inspection, err
	}

	for _, row := range rows {
		if !row.IsDir() {
			continue
		}
		stage := filepath.Join(root, row.Name())
		record, issues, ok := inspectStage(stage)
		inspection.Issues = append(inspection.Issues, issues...)
		if ok {
			inspection.Records = append(inspection.Records, record)
		}
	}
	return inspection, nil
}

func inspectStage(stage string) (Record, []Issue, bool) {
	issues := []Issue{}
	var job importmodel.Job
	found, err := persistence.LoadJSON(filepath.Join(stage, "job.json"), &job)
	if err != nil {
		issues = append(issues, issue(stage, "JOB_STATE_INVALID", err.Error()))
		return Record{}, issues, false
	}
	if !found {
		issues = append(issues, issue(stage, "JOB_STATE_MISSING", "job.json is missing"))
		return Record{}, issues, false
	}
	job.ID = strings.TrimSpace(job.ID)
	if job.ID == "" {
		issues = append(issues, issue(stage, "JOB_ID_MISSING", "persisted import job has no id"))
		return Record{}, issues, false
	}

	record := Record{
		StageDir:    stage,
		Job:         job,
		Disposition: importmodel.RestartClass(job.Status),
		Prepared:    false,
	}

	var manifest Manifest
	manifestFound, manifestErr := persistence.LoadJSON(filepath.Join(stage, "manifest.json"), &manifest)
	if manifestErr != nil {
		issues = append(issues, issue(stage, "MANIFEST_INVALID", manifestErr.Error()))
	} else if manifestFound {
		record.Manifest = &manifest
		validateManifest(stage, job, &record, &issues)
	} else if job.Status == importmodel.StatusReady || job.Status == importmodel.StatusCommitting {
		issues = append(issues, issue(stage, "MANIFEST_MISSING", "commit-capable staged job has no manifest.json"))
	}

	if importmodel.KnownStatus(job.Status) == false {
		issues = append(issues, issue(stage, "JOB_STATUS_UNKNOWN", fmt.Sprintf("unknown persisted status %q", job.Status)))
	}
	if importmodel.IsTerminal(job.Status) {
		issues = append(issues, issue(stage, "TERMINAL_STAGE_REMAINS", "terminal import job still has staging custody"))
	}
	if job.Status == importmodel.StatusCommitting {
		issues = append(issues, issue(stage, "COMMIT_RECONCILIATION_REQUIRED", "previous process stopped at or after commit admission; do not auto-replay"))
	}
	if record.Disposition == importmodel.RestartInterruptedPreparation {
		issues = append(issues, issue(stage, "PREPARATION_INTERRUPTED", "provider preparation did not reach a safely resumable ready state"))
	}

	return record, issues, true
}

func validateManifest(stage string, job importmodel.Job, record *Record, issues *[]Issue) {
	manifest := record.Manifest
	if manifest == nil {
		return
	}
	if manifest.SchemaVersion != ManifestSchema {
		*issues = append(*issues, issue(
			stage,
			"MANIFEST_SCHEMA_UNSUPPORTED",
			fmt.Sprintf("unsupported staging manifest schema %q", manifest.SchemaVersion),
		))
		return
	}
	if len(manifest.Outputs) == 0 {
		*issues = append(*issues, issue(stage, "MANIFEST_EMPTY", "staging manifest has no outputs"))
		return
	}
	if job.Destination != "" && manifest.Destination != "" &&
		!samePath(job.Destination, manifest.Destination) {
		*issues = append(*issues, issue(stage, "DESTINATION_MISMATCH", "job and manifest destination differ"))
	}

	missing := []string{}
	unsafe := false
	for _, output := range manifest.Outputs {
		relative := filepath.FromSlash(strings.TrimSpace(output.RelativePath))
		if !safeRelative(relative) {
			unsafe = true
			*issues = append(*issues, issue(stage, "OUTPUT_PATH_UNSAFE", output.RelativePath))
			continue
		}
		path := filepath.Join(stage, relative)
		if !contained(stage, path) {
			unsafe = true
			*issues = append(*issues, issue(stage, "OUTPUT_PATH_ESCAPE", output.RelativePath))
			continue
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			missing = append(missing, output.RelativePath)
		}
	}
	record.MissingOutputRefs = append([]string(nil), missing...)
	record.Prepared = !unsafe && len(missing) == 0

	if len(missing) > 0 {
		code := "STAGED_OUTPUT_MISSING"
		detail := "one or more manifest outputs are no longer present in staging"
		if job.Status == importmodel.StatusCommitting {
			code = "COMMIT_OUTPUTS_PARTIALLY_ABSENT"
			detail = "one or more outputs are absent from staging during uncertain commit recovery; they may already have moved"
		}
		*issues = append(*issues, issue(stage, code, detail))
	}
}

func issue(stage, code, detail string) Issue {
	return Issue{StageDir: stage, Code: code, Detail: detail}
}

func safeRelative(path string) bool {
	if path == "" || filepath.IsAbs(path) {
		return false
	}
	cleaned := filepath.Clean(path)
	return cleaned != "." &&
		cleaned != ".." &&
		!strings.HasPrefix(cleaned, ".."+string(os.PathSeparator))
}

func contained(root, path string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false
	}
	return relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	return strings.EqualFold(left, right)
}
