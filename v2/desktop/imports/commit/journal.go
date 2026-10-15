// Package commit owns crash-recoverable promotion of prepared import outputs into one VexStream
// Library source.
//
// A journal is persisted before destination writes. Exact byte identity and per-destination claims
// make replay observable rather than assumptive. Catalog writes are idempotent through a stable
// vexImportCommitRef carried in each committed provenance row.
//
// This package does not mark import jobs "done", remove the staging directory, or trigger Library
// reconciliation. Those are higher lifecycle effects after CATALOG_COMMITTED is observed.
package commit

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/staging"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/catalog"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/filesystem"
	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

const (
	JournalSchema = "vexstream.import-commit/v1"
	JournalName   = "commit-journal.json"
)

type State string

const (
	StatePlanned          State = "PLANNED"
	StateMoving           State = "MOVING"
	StateFilesReady       State = "FILES_READY"
	StateCatalogCommitted State = "CATALOG_COMMITTED"
)

type Entry struct {
	CommitRef       string         `json:"commitRef"`
	RelativePath    string         `json:"relativePath"`
	Filename        string         `json:"filename"`
	DestinationPath string         `json:"destinationPath"`
	Size            int64          `json:"size"`
	SHA256          string         `json:"sha256"`
	CatalogEntry    map[string]any `json:"catalogEntry,omitempty"`
}

type Journal struct {
	SchemaVersion string  `json:"schemaVersion"`
	JobID         string  `json:"jobId"`
	Destination   string  `json:"destination"`
	State         State   `json:"state"`
	Entries       []Entry `json:"entries"`
}

type Result struct {
	Added        int      `json:"added"`
	TrackRefs    []string `json:"trackIds"`
	Destinations []string `json:"destinations"`
}

func Prepare(jobID, stageDir, destination string, manifest staging.Manifest) (Journal, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return Journal{}, errors.New("commit journal requires job id")
	}
	stageDir = filepath.Clean(stageDir)
	destination = filepath.Clean(destination)
	if stageDir == "." || destination == "." {
		return Journal{}, errors.New("commit journal requires stage and destination")
	}
	if manifest.Destination != "" && !samePath(manifest.Destination, destination) {
		return Journal{}, errors.New("staging manifest destination does not match import job")
	}
	if len(manifest.Outputs) == 0 {
		return Journal{}, errors.New("staging manifest has no outputs")
	}

	if existing, found, err := Load(stageDir); err != nil {
		return Journal{}, err
	} else if found {
		if existing.JobID != jobID || !samePath(existing.Destination, destination) {
			return Journal{}, errors.New("existing commit journal belongs to another import")
		}
		return existing, nil
	}

	reserved := map[string]bool{}
	entries := make([]Entry, 0, len(manifest.Outputs))
	for index, output := range manifest.Outputs {
		relative := filepath.FromSlash(strings.TrimSpace(output.RelativePath))
		if !safeRelative(relative) {
			return Journal{}, fmt.Errorf("invalid staged output path: %q", output.RelativePath)
		}
		source := filepath.Join(stageDir, relative)
		if !contained(stageDir, source) {
			return Journal{}, fmt.Errorf("staged output escapes stage: %q", output.RelativePath)
		}
		size, digest, err := exactIdentity(source)
		if err != nil {
			return Journal{}, fmt.Errorf("inspect staged output %q: %w", output.RelativePath, err)
		}

		filename := strings.TrimSpace(output.Filename)
		if filename == "" {
			filename = filepath.Base(source)
		}
		filename = filepath.Base(filename)
		if filename == "." || filename == "" {
			return Journal{}, fmt.Errorf("staged output %q has no filename", output.RelativePath)
		}
		destinationPath, err := chooseAvailable(destination, filename, reserved)
		if err != nil {
			return Journal{}, err
		}
		reserved[pathKey(destinationPath)] = true

		refHash := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%s\x00%s", jobID, index, digest, destinationPath)))
		commitRef := hex.EncodeToString(refHash[:16])
		entries = append(entries, Entry{
			CommitRef:       commitRef,
			RelativePath:    filepath.ToSlash(relative),
			Filename:        filepath.Base(destinationPath),
			DestinationPath: destinationPath,
			Size:            size,
			SHA256:          digest,
			CatalogEntry:    cloneMap(output.CatalogEntry),
		})
	}

	journal := Journal{
		SchemaVersion: JournalSchema,
		JobID:         jobID,
		Destination:   destination,
		State:         StatePlanned,
		Entries:       entries,
	}
	if err := save(stageDir, journal); err != nil {
		return Journal{}, err
	}
	return cloneJournal(journal), nil
}

func Load(stageDir string) (Journal, bool, error) {
	var journal Journal
	found, err := persistence.LoadJSON(filepath.Join(stageDir, JournalName), &journal)
	if err != nil || !found {
		return Journal{}, found, err
	}
	if journal.SchemaVersion != JournalSchema {
		return Journal{}, true, fmt.Errorf("unsupported import commit journal schema: %q", journal.SchemaVersion)
	}
	if journal.JobID == "" || journal.Destination == "" || len(journal.Entries) == 0 {
		return Journal{}, true, errors.New("import commit journal is incomplete")
	}
	return cloneJournal(journal), true, nil
}

// Resume advances an already-prepared V4 commit journal idempotently.
//
// It is safe to call after process restart. It never creates a journal from an unjournaled legacy
// "committing" job; callers must hold those for explicit recovery.
func Resume(stageDir string) (Result, error) {
	journal, found, err := Load(stageDir)
	if err != nil {
		return Result{}, err
	}
	if !found {
		return Result{}, errors.New("import commit journal is missing")
	}

	if journal.State != StateCatalogCommitted {
		journal.State = StateMoving
		if err := save(stageDir, journal); err != nil {
			return Result{}, err
		}
		if err := moveOutputs(stageDir, &journal); err != nil {
			return Result{}, err
		}
		journal.State = StateFilesReady
		if err := save(stageDir, journal); err != nil {
			return Result{}, err
		}
	}

	if err := commitCatalog(stageDir, &journal); err != nil {
		return Result{}, err
	}
	journal.State = StateCatalogCommitted
	if err := save(stageDir, journal); err != nil {
		return Result{}, err
	}
	releaseClaims(journal)

	result := Result{
		Added:        len(journal.Entries),
		TrackRefs:    make([]string, 0, len(journal.Entries)),
		Destinations: make([]string, 0, len(journal.Entries)),
	}
	for _, entry := range journal.Entries {
		result.TrackRefs = append(result.TrackRefs, filesystem.TrackID(entry.DestinationPath))
		result.Destinations = append(result.Destinations, entry.DestinationPath)
	}
	return result, nil
}

func moveOutputs(stageDir string, journal *Journal) error {
	for _, entry := range journal.Entries {
		source := filepath.Join(stageDir, filepath.FromSlash(entry.RelativePath))
		if !contained(stageDir, source) {
			return fmt.Errorf("journal source escapes stage: %q", entry.RelativePath)
		}
		if !contained(journal.Destination, entry.DestinationPath) {
			return fmt.Errorf("journal destination escapes source folder: %q", entry.DestinationPath)
		}

		sourceExists, sourceExact, err := identityMatches(source, entry.Size, entry.SHA256)
		if err != nil {
			return err
		}
		destinationExists, destinationExact, err := identityMatches(entry.DestinationPath, entry.Size, entry.SHA256)
		if err != nil {
			return err
		}

		if destinationExact {
			if sourceExact {
				// Exact destination bytes are sufficient evidence. The staged duplicate can be
				// removed even if a previous process already released/lost its claim.
				if err := os.Remove(source); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("remove duplicate staged output: %w", err)
				}
			}
			continue
		}

		if destinationExists {
			owned, err := existingClaimOwned(entry)
			if err != nil {
				return err
			}
			if !owned {
				return fmt.Errorf("destination collision during import commit: %s", entry.DestinationPath)
			}
			// A previous copy from this same journal may have stopped before completion. The staged
			// source must still match before we discard the incomplete destination.
			if !sourceExact {
				return fmt.Errorf("claimed destination is incomplete and staged source is unavailable: %s", entry.DestinationPath)
			}
			if err := os.Remove(entry.DestinationPath); err != nil {
				return fmt.Errorf("remove incomplete claimed destination: %w", err)
			}
		} else if err := ensureClaim(entry); err != nil {
			return err
		}

		if !sourceExists || !sourceExact {
			return fmt.Errorf("staged output no longer matches commit journal: %s", entry.RelativePath)
		}
		if err := copyExclusive(source, entry.DestinationPath); err != nil {
			return err
		}
		_, destinationExact, err = identityMatches(entry.DestinationPath, entry.Size, entry.SHA256)
		if err != nil {
			return err
		}
		if !destinationExact {
			return fmt.Errorf("destination bytes do not match commit journal: %s", entry.DestinationPath)
		}
		if err := os.Remove(source); err != nil {
			return fmt.Errorf("remove staged output after exact destination commit: %w", err)
		}
	}
	return nil
}

// ensureClaim creates this journal's destination claim only while the destination is absent.
//
// Callers must inspect an already-existing destination before invoking this function; creating a
// claim after seeing foreign bytes would manufacture false custody.
func ensureClaim(entry Entry) error {
	path := claimPath(entry.DestinationPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create destination claim directory: %w", err)
	}
	expected := []byte(entry.CommitRef + "\n")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		if _, writeErr := file.Write(expected); writeErr != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return writeErr
		}
		if closeErr := file.Close(); closeErr != nil {
			_ = os.Remove(path)
			return closeErr
		}
		return nil
	}
	if !os.IsExist(err) {
		return fmt.Errorf("create destination claim: %w", err)
	}
	owned, readErr := existingClaimOwned(entry)
	if readErr != nil {
		return readErr
	}
	if !owned {
		return fmt.Errorf("destination is claimed by another import: %s", entry.DestinationPath)
	}
	return nil
}

func existingClaimOwned(entry Entry) (bool, error) {
	data, err := os.ReadFile(claimPath(entry.DestinationPath))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read destination claim: %w", err)
	}
	return strings.TrimSpace(string(data)) == entry.CommitRef, nil
}

func commitCatalog(stageDir string, journal *Journal) error {
	entries := make([]map[string]any, 0, len(journal.Entries))
	for _, entry := range journal.Entries {
		_, exact, err := identityMatches(entry.DestinationPath, entry.Size, entry.SHA256)
		if err != nil {
			return err
		}
		if !exact {
			return fmt.Errorf("cannot commit catalog before destination bytes are exact: %s", entry.DestinationPath)
		}
		row := cloneMap(entry.CatalogEntry)
		row["filename"] = filepath.Base(entry.DestinationPath)
		row["relativePath"] = filepath.Base(entry.DestinationPath)
		row["bytes"] = entry.Size
		row["vexImportCommitRef"] = entry.CommitRef
		entries = append(entries, row)
	}

	path := filepath.Join(journal.Destination, "_vexmedia", "library.jsonl")
	rows, err := catalog.ReadRows(path)
	if err != nil {
		return fmt.Errorf("read source catalog: %w", err)
	}
	rows = upsertCommitRows(rows, entries)
	if err := catalog.WriteRowsAtomic(path, rows); err != nil {
		return fmt.Errorf("commit source catalog: %w", err)
	}
	return nil
}

func upsertCommitRows(rows, entries []map[string]any) []map[string]any {
	out := cloneRows(rows)
	for _, entry := range entries {
		ref, _ := entry["vexImportCommitRef"].(string)
		replaced := false
		if ref != "" {
			for index, row := range out {
				if existing, _ := row["vexImportCommitRef"].(string); existing == ref {
					out[index] = cloneMap(entry)
					replaced = true
					break
				}
			}
		}
		if !replaced {
			out = catalog.UpsertProviderRows(out, []map[string]any{entry})
		}
	}
	return out
}

func copyExclusive(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create import destination: %w", err)
	}
	_, copyErr := io.Copy(output, input)
	syncErr := output.Sync()
	closeErr := output.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(destination)
		if copyErr != nil {
			return copyErr
		}
		if syncErr != nil {
			return syncErr
		}
		return closeErr
	}
	return nil
}

func chooseAvailable(destination, filename string, reserved map[string]bool) (string, error) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)
	for index := 1; index < 10000; index++ {
		name := filename
		if index > 1 {
			name = fmt.Sprintf("%s (%d)%s", base, index, ext)
		}
		candidate := filepath.Join(destination, name)
		key := pathKey(candidate)
		if reserved[key] {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if _, err := os.Stat(claimPath(candidate)); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return "", err
		}
		return candidate, nil
	}
	return "", errors.New("could not allocate import destination filename")
}

func exactIdentity(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, "", err
	}
	if !info.Mode().IsRegular() {
		return 0, "", errors.New("not a regular file")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return 0, "", err
	}
	return info.Size(), hex.EncodeToString(hash.Sum(nil)), nil
}

func identityMatches(path string, size int64, digest string) (exists bool, exact bool, err error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if !info.Mode().IsRegular() {
		return true, false, nil
	}
	if info.Size() != size {
		return true, false, nil
	}
	actualSize, actualDigest, err := exactIdentity(path)
	if err != nil {
		return true, false, err
	}
	return true, actualSize == size && actualDigest == digest, nil
}

func save(stageDir string, journal Journal) error {
	return persistence.SaveJSONAtomic(filepath.Join(stageDir, JournalName), journal)
}

func releaseClaims(journal Journal) {
	for _, entry := range journal.Entries {
		path := claimPath(entry.DestinationPath)
		data, err := os.ReadFile(path)
		if err == nil && strings.TrimSpace(string(data)) == entry.CommitRef {
			_ = os.Remove(path)
		}
	}
}

func claimPath(destination string) string {
	return destination + ".vex-import-claim"
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
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}

func pathKey(path string) string {
	return strings.ToLower(filepath.Clean(path))
}

func cloneJournal(journal Journal) Journal {
	journal.Entries = append([]Entry(nil), journal.Entries...)
	for index := range journal.Entries {
		journal.Entries[index].CatalogEntry = cloneMap(journal.Entries[index].CatalogEntry)
	}
	return journal
}

func cloneRows(rows []map[string]any) []map[string]any {
	out := make([]map[string]any, len(rows))
	for index, row := range rows {
		out[index] = cloneMap(row)
	}
	return out
}

func cloneMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
