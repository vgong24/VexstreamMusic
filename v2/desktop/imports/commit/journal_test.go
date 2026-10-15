package commit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/staging"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/catalog"
	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

func prepareFixture(t *testing.T, names ...string) (stage, destination string, manifest staging.Manifest) {
	t.Helper()
	root := t.TempDir()
	stage = filepath.Join(root, "stage")
	destination = filepath.Join(root, "music")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest.SchemaVersion = staging.ManifestSchema
	manifest.Destination = destination
	for index, name := range names {
		relative := filepath.Join("out", name)
		path := filepath.Join(stage, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		data := []byte(strings.Repeat(string(rune('A'+index)), 32+index))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		manifest.Outputs = append(manifest.Outputs, staging.Output{
			RelativePath: filepath.ToSlash(relative),
			Filename: name,
			CatalogEntry: map[string]any{
				"provider": "youtube",
				"providerId": strings.TrimSuffix(name, filepath.Ext(name)),
				"metadata": map[string]any{"title": name},
			},
		})
	}
	return
}

func TestPreparePersistsExactByteIdentityAndStableDestinations(t *testing.T) {
	stage, destination, manifest := prepareFixture(t, "song.mp3", "song.mp3")
	journal, err := Prepare("job-1", stage, destination, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if journal.State != StatePlanned || len(journal.Entries) != 2 {
		t.Fatalf("journal formation drift: %+v", journal)
	}
	if filepath.Base(journal.Entries[0].DestinationPath) != "song.mp3" ||
		filepath.Base(journal.Entries[1].DestinationPath) != "song (2).mp3" {
		t.Fatalf("destination allocation drift: %+v", journal.Entries)
	}
	for _, entry := range journal.Entries {
		if entry.Size <= 0 || len(entry.SHA256) != 64 || entry.CommitRef == "" {
			t.Fatalf("byte identity missing: %+v", entry)
		}
	}
	loaded, found, err := Load(stage)
	if err != nil || !found || loaded.Entries[1].DestinationPath != journal.Entries[1].DestinationPath {
		t.Fatalf("journal persistence drift: %+v found=%v err=%v", loaded, found, err)
	}

	again, err := Prepare("job-1", stage, destination, manifest)
	if err != nil || again.Entries[1].DestinationPath != journal.Entries[1].DestinationPath {
		t.Fatalf("prepare replay changed destination: %+v err=%v", again, err)
	}
}

func TestResumeMovesExactBytesCommitsCatalogAndIsIdempotent(t *testing.T) {
	stage, destination, manifest := prepareFixture(t, "one.mp3", "two.mp3")
	journal, err := Prepare("job-2", stage, destination, manifest)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Resume(stage)
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 2 || len(result.TrackRefs) != 2 {
		t.Fatalf("commit result drift: %+v", result)
	}
	for index, entry := range journal.Entries {
		if _, exact, err := identityMatches(entry.DestinationPath, entry.Size, entry.SHA256); err != nil || !exact {
			t.Fatalf("destination %d not exact: exact=%v err=%v", index, exact, err)
		}
		source := filepath.Join(stage, filepath.FromSlash(entry.RelativePath))
		if _, err := os.Stat(source); !os.IsNotExist(err) {
			t.Fatalf("staged source still present after commit: %s err=%v", source, err)
		}
		if _, err := os.Stat(claimPath(entry.DestinationPath)); !os.IsNotExist(err) {
			t.Fatalf("claim not released: %s err=%v", claimPath(entry.DestinationPath), err)
		}
	}

	rows, err := catalog.ReadRows(filepath.Join(destination, "_vexmedia", "library.jsonl"))
	if err != nil || len(rows) != 2 {
		t.Fatalf("catalog commit drift: %+v err=%v", rows, err)
	}
	for _, row := range rows {
		if row["vexImportCommitRef"] == "" {
			t.Fatalf("catalog row missing commit identity: %+v", row)
		}
	}

	second, err := Resume(stage)
	if err != nil {
		t.Fatal(err)
	}
	if second.Added != 2 {
		t.Fatalf("idempotent resume result drift: %+v", second)
	}
	rows, err = catalog.ReadRows(filepath.Join(destination, "_vexmedia", "library.jsonl"))
	if err != nil || len(rows) != 2 {
		t.Fatalf("idempotent catalog replay duplicated rows: %+v err=%v", rows, err)
	}
}

func TestResumeRecoversCrashAfterOneOutputMovedBeforeJournalProgress(t *testing.T) {
	stage, _, manifest := prepareFixture(t, "one.mp3", "two.mp3")
	journal, err := Prepare("job-3", stage, manifest.Destination, manifest)
	if err != nil {
		t.Fatal(err)
	}
	first := journal.Entries[0]
	source := filepath.Join(stage, filepath.FromSlash(first.RelativePath))
	if err := os.MkdirAll(filepath.Dir(first.DestinationPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ensureClaim(first); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, first.DestinationPath); err != nil {
		t.Fatal(err)
	}

	result, err := Resume(stage)
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 2 {
		t.Fatalf("partial-move recovery drift: %+v", result)
	}
	for _, entry := range journal.Entries {
		if _, exact, _ := identityMatches(entry.DestinationPath, entry.Size, entry.SHA256); !exact {
			t.Fatalf("recovery lost destination: %+v", entry)
		}
	}
}

func TestClaimedIncompleteDestinationCanBeRebuiltFromExactStage(t *testing.T) {
	stage, _, manifest := prepareFixture(t, "one.mp3")
	journal, err := Prepare("job-4", stage, manifest.Destination, manifest)
	if err != nil {
		t.Fatal(err)
	}
	entry := journal.Entries[0]
	if err := os.MkdirAll(filepath.Dir(entry.DestinationPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ensureClaim(entry); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry.DestinationPath, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Resume(stage); err != nil {
		t.Fatal(err)
	}
	if _, exact, _ := identityMatches(entry.DestinationPath, entry.Size, entry.SHA256); !exact {
		t.Fatal("claimed partial destination was not rebuilt exactly")
	}
}

func TestForeignDestinationCollisionFailsClosed(t *testing.T) {
	stage, _, manifest := prepareFixture(t, "one.mp3")
	journal, err := Prepare("job-5", stage, manifest.Destination, manifest)
	if err != nil {
		t.Fatal(err)
	}
	entry := journal.Entries[0]
	if err := os.MkdirAll(filepath.Dir(entry.DestinationPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry.DestinationPath, []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Resume(stage); err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("foreign collision was not held: %v", err)
	}
	source := filepath.Join(stage, filepath.FromSlash(entry.RelativePath))
	if _, exact, _ := identityMatches(source, entry.Size, entry.SHA256); !exact {
		t.Fatal("collision damaged staged source")
	}
}

func TestSourceMutationAfterJournalFailsBeforeLibraryCommit(t *testing.T) {
	stage, destination, manifest := prepareFixture(t, "one.mp3")
	journal, err := Prepare("job-6", stage, destination, manifest)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(stage, filepath.FromSlash(journal.Entries[0].RelativePath))
	if err := os.WriteFile(source, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resume(stage); err == nil || !strings.Contains(err.Error(), "no longer matches") {
		t.Fatalf("mutated staged output was accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "_vexmedia", "library.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("catalog mutated after source mismatch: %v", err)
	}
}

func TestCatalogReplayIsIdempotentEvenWithoutProviderID(t *testing.T) {
	stage, destination, manifest := prepareFixture(t, "anonymous.mp3")
	delete(manifest.Outputs[0].CatalogEntry, "providerId")
	if _, err := Prepare("job-7", stage, destination, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := Resume(stage); err != nil {
		t.Fatal(err)
	}

	journal, _, err := Load(stage)
	if err != nil {
		t.Fatal(err)
	}
	journal.State = StateFilesReady
	if err := save(stage, journal); err != nil {
		t.Fatal(err)
	}
	if _, err := Resume(stage); err != nil {
		t.Fatal(err)
	}
	rows, err := catalog.ReadRows(filepath.Join(destination, "_vexmedia", "library.jsonl"))
	if err != nil || len(rows) != 1 {
		t.Fatalf("anonymous catalog replay duplicated row: %+v err=%v", rows, err)
	}
}

func TestPrepareRejectsTraversalBeforeJournalCreation(t *testing.T) {
	root := t.TempDir()
	stage := filepath.Join(root, "stage")
	destination := filepath.Join(root, "music")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := staging.Manifest{
		Destination: destination,
		Outputs: []staging.Output{{RelativePath: "../escape.mp3"}},
	}
	if _, err := Prepare("job-8", stage, destination, manifest); err == nil {
		t.Fatal("traversal manifest was accepted")
	}
	if _, err := os.Stat(filepath.Join(stage, JournalName)); !os.IsNotExist(err) {
		t.Fatalf("journal created for unsafe manifest: %v", err)
	}
}


func TestCommitPreparedLoadsKnownManifestAndRejectsUnknownSchema(t *testing.T) {
	stage, destination, manifest := prepareFixture(t, "one.mp3")
	if err := persistence.SaveJSONAtomic(
		filepath.Join(stage, "manifest.json"),
		manifest,
	); err != nil {
		t.Fatal(err)
	}
	result, err := CommitPrepared("job-prepared", stage, destination)
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 || len(result.TrackRefs) != 1 {
		t.Fatalf("prepared commit drift: %+v", result)
	}

	stage2, destination2, future := prepareFixture(t, "future.mp3")
	future.SchemaVersion = "vexstream.import-stage/v99"
	if err := persistence.SaveJSONAtomic(
		filepath.Join(stage2, "manifest.json"),
		future,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitPrepared("job-future", stage2, destination2); err == nil ||
		!strings.Contains(err.Error(), "unsupported staging manifest schema") {
		t.Fatalf("future manifest was admitted: %v", err)
	}
}

func TestCommitPreparedRequiresPersistedManifest(t *testing.T) {
	root := t.TempDir()
	stage := filepath.Join(root, "stage")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitPrepared(
		"job-missing",
		stage,
		filepath.Join(root, "music"),
	); err == nil || !strings.Contains(err.Error(), "manifest is missing") {
		t.Fatalf("missing manifest was admitted: %v", err)
	}
}
