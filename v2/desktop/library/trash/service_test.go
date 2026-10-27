package trash

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

type cleanerFixture struct {
	calls [][]Item
	err   error
}

func (c *cleanerFixture) CleanPurged(items []Item) error {
	c.calls = append(c.calls, cloneItems(items))
	return c.err
}

func testPaths(root string) Paths {
	return Paths{
		TrashState:     filepath.Join(root, "trash.json"),
		DeletedHistory: filepath.Join(root, "deleted-history.json"),
		FilesRoot:      filepath.Join(root, "trash", "files"),
		PurgeJournal:   filepath.Join(root, "trash-purge-journal.json"),
	}
}

func deterministicIDs(values ...string) func() (string, error) {
	index := 0
	return func() (string, error) {
		if index >= len(values) {
			return "", errors.New("no more ids")
		}
		value := values[index]
		index++
		return value, nil
	}
}

func writeTrack(t *testing.T, root, name string) (library.Track, string) {
	t.Helper()
	path := filepath.Join(root, "music", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("audio-"+name), 0o644); err != nil {
		t.Fatal(err)
	}
	return library.Track{
		ID:       "track-" + name,
		Path:     path,
		Filename: name,
		Title:    name,
		Source:   filepath.Dir(path),
	}, path
}

func TestMovePersistsV3TrashShapeAndRemovesOriginalBytes(t *testing.T) {
	root := t.TempDir()
	cleaner := &cleanerFixture{}
	service, err := Open(testPaths(root), cleaner)
	if err != nil {
		t.Fatal(err)
	}
	service.newID = deterministicIDs("trash-a")
	service.now = func() time.Time {
		return time.Date(2026, 10, 14, 1, 2, 3, 0, time.UTC)
	}
	track, path := writeTrack(t, root, "song.mp3")

	moved, err := service.Move([]MoveInput{{Track: track, Path: path}})
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 1 ||
		moved[0].ID != "trash-a" ||
		moved[0].OriginalPath != path {
		t.Fatalf("trash move drift: %+v", moved)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("original still present after Trash move: %v", err)
	}
	if _, err := os.Stat(moved[0].TrashPath); err != nil {
		t.Fatalf("Trash bytes missing: %v", err)
	}

	data, err := os.ReadFile(testPaths(root).TrashState)
	if err != nil {
		t.Fatal(err)
	}
	var v3 []struct {
		ID           string        `json:"id"`
		Track        library.Track `json:"track"`
		OriginalPath string        `json:"originalPath"`
		TrashPath    string        `json:"trashPath"`
		TrashedAt    string        `json:"trashedAt"`
	}
	if err := json.Unmarshal(data, &v3); err != nil ||
		len(v3) != 1 ||
		v3[0].ID != "trash-a" {
		t.Fatalf("V3 Trash shape drift: %s err=%v", data, err)
	}
}

func TestMoveRollsBackBytesWhenTrashStateCannotPersist(t *testing.T) {
	root := t.TempDir()
	paths := testPaths(root)
	service, err := Open(paths, &cleanerFixture{})
	if err != nil {
		t.Fatal(err)
	}
	service.newID = deterministicIDs("trash-a")
	track, path := writeTrack(t, root, "song.mp3")

	if err := os.MkdirAll(paths.TrashState, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Move([]MoveInput{{Track: track, Path: path}}); err == nil {
		t.Fatal("Trash persistence failure was swallowed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("audio was not rolled back to original path: %v", err)
	}
	if len(service.Snapshot().Trash) != 0 {
		t.Fatalf("in-memory Trash moved ahead of persistence: %+v", service.Snapshot())
	}
}

func TestRestoreRollsBackBytesWhenUpdatedTrashStateCannotPersist(t *testing.T) {
	root := t.TempDir()
	paths := testPaths(root)
	service, err := Open(paths, &cleanerFixture{})
	if err != nil {
		t.Fatal(err)
	}
	service.newID = deterministicIDs("trash-a")
	track, path := writeTrack(t, root, "song.mp3")
	moved, err := service.Move([]MoveInput{{Track: track, Path: path}})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(paths.TrashState); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(paths.TrashState, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Restore([]string{"trash-a"}); err == nil {
		t.Fatal("restore persistence failure was swallowed")
	}
	if _, err := os.Stat(moved[0].TrashPath); err != nil {
		t.Fatalf("restore failure did not return bytes to Trash: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("restore failure left original bytes visible: %v", err)
	}
	if len(service.Snapshot().Trash) != 1 {
		t.Fatalf("in-memory Trash moved ahead of persistence: %+v", service.Snapshot())
	}
}

func TestRestoreBlocksOccupiedOriginalPathBeforeMovingBytes(t *testing.T) {
	root := t.TempDir()
	service, err := Open(testPaths(root), &cleanerFixture{})
	if err != nil {
		t.Fatal(err)
	}
	service.newID = deterministicIDs("trash-a")
	track, path := writeTrack(t, root, "song.mp3")
	moved, err := service.Move([]MoveInput{{Track: track, Path: path}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Restore([]string{"trash-a"}); !errors.Is(err, ErrRestoreCollision) {
		t.Fatalf("restore collision drift: %v", err)
	}
	if _, err := os.Stat(moved[0].TrashPath); err != nil {
		t.Fatalf("Trash bytes moved despite collision: %v", err)
	}
}

func TestPurgePersistsIntentBeforeDeletionAndCompletesV3Stores(t *testing.T) {
	root := t.TempDir()
	paths := testPaths(root)
	cleaner := &cleanerFixture{}
	service, err := Open(paths, cleaner)
	if err != nil {
		t.Fatal(err)
	}
	service.newID = deterministicIDs("trash-a", "purge-a", "history-a")
	track, path := writeTrack(t, root, "song.mp3")
	moved, err := service.Move([]MoveInput{{Track: track, Path: path}})
	if err != nil {
		t.Fatal(err)
	}

	count, err := service.Purge([]string{"trash-a"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(cleaner.calls) != 1 {
		t.Fatalf("purge cleanup drift: count=%d calls=%d", count, len(cleaner.calls))
	}
	if _, err := os.Stat(moved[0].TrashPath); !os.IsNotExist(err) {
		t.Fatalf("purged bytes remain: %v", err)
	}
	snapshot := service.Snapshot()
	if len(snapshot.Trash) != 0 ||
		len(snapshot.DeletedHistory) != 1 ||
		snapshot.DeletedHistory[0].ID != "history-a" ||
		snapshot.PendingPurge != nil {
		t.Fatalf("purge final state drift: %+v", snapshot)
	}
	if _, err := os.Stat(paths.PurgeJournal); !os.IsNotExist(err) {
		t.Fatalf("completed purge journal remains: %v", err)
	}

	var v3History []DeletedHistoryItem
	found, err := persistence.LoadJSON(paths.DeletedHistory, &v3History)
	if err != nil || !found || len(v3History) != 1 {
		t.Fatalf("V3 deleted-history shape drift: %+v found=%v err=%v", v3History, found, err)
	}
}

func TestPurgeCleanupFailureLeavesRecoverableJournalAfterBytesAndStoresAdvance(t *testing.T) {
	root := t.TempDir()
	paths := testPaths(root)
	cleaner := &cleanerFixture{err: errors.New("catalog busy")}
	service, err := Open(paths, cleaner)
	if err != nil {
		t.Fatal(err)
	}
	service.newID = deterministicIDs("trash-a", "purge-a", "history-a")
	track, path := writeTrack(t, root, "song.mp3")
	moved, err := service.Move([]MoveInput{{Track: track, Path: path}})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.Purge([]string{"trash-a"}, true); err == nil ||
		!strings.Contains(err.Error(), "catalog busy") {
		t.Fatalf("cleanup failure not surfaced: %v", err)
	}
	if _, err := os.Stat(moved[0].TrashPath); !os.IsNotExist(err) {
		t.Fatalf("purge cleanup failure resurrected bytes: %v", err)
	}
	snapshot := service.Snapshot()
	if snapshot.PendingPurge == nil ||
		snapshot.PendingPurge.State != PurgeStateStoresUpdated ||
		len(snapshot.Trash) != 0 ||
		len(snapshot.DeletedHistory) != 1 {
		t.Fatalf("recoverable purge checkpoint drift: %+v", snapshot)
	}

	cleaner.err = nil
	count, err := service.RecoverPurge()
	if err != nil || count != 1 {
		t.Fatalf("purge recovery failed: count=%d err=%v", count, err)
	}
	if len(cleaner.calls) != 2 ||
		service.Snapshot().PendingPurge != nil {
		t.Fatalf("purge cleanup was not retried exactly: calls=%d snapshot=%+v", len(cleaner.calls), service.Snapshot())
	}
}

func TestOpenRecoversPendingJournalWithoutInventingCompletion(t *testing.T) {
	root := t.TempDir()
	paths := testPaths(root)
	item := Item{
		ID:           "trash-a",
		Track:        library.Track{ID: "track-a", Filename: "song.mp3", Source: filepath.Join(root, "music")},
		OriginalPath: filepath.Join(root, "music", "song.mp3"),
		TrashPath:    filepath.Join(paths.FilesRoot, "trash-a.mp3"),
		TrashedAt:    "earlier",
	}
	if err := os.MkdirAll(filepath.Dir(item.TrashPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(item.TrashPath, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := persistence.SaveJSONAtomic(paths.TrashState, []Item{item}); err != nil {
		t.Fatal(err)
	}
	journal := PurgeJournal{
		SchemaVersion: PurgeJournalSchema,
		PurgeRef:      "purge-a",
		State:         PurgeStatePlanned,
		Entries:       []PurgeEntry{{Item: item}},
	}
	if err := persistence.SaveJSONAtomic(paths.PurgeJournal, journal); err != nil {
		t.Fatal(err)
	}

	cleaner := &cleanerFixture{}
	service, err := Open(paths, cleaner)
	if err != nil {
		t.Fatal(err)
	}
	if service.Snapshot().PendingPurge == nil {
		t.Fatal("pending purge journal was silently dropped")
	}
	count, err := service.RecoverPurge()
	if err != nil || count != 1 {
		t.Fatalf("pending purge recovery drift: count=%d err=%v", count, err)
	}
	if _, err := os.Stat(item.TrashPath); !os.IsNotExist(err) {
		t.Fatalf("journal recovery did not delete admitted bytes: %v", err)
	}
}

func TestMoveInputNormalizationDoesNotDuplicateTrackCustody(t *testing.T) {
	track := library.Track{ID: "track-a", Tags: []string{"tag"}}
	got := normalizeMoveInputs([]MoveInput{
		{Track: track, Path: "/one.mp3"},
		{Track: track, Path: "/two.mp3"},
		{Track: library.Track{}, Path: "/missing-id.mp3"},
	})
	if len(got) != 1 ||
		got[0].Path != filepath.Clean("/one.mp3") ||
		!reflect.DeepEqual(got[0].Track.Tags, []string{"tag"}) {
		t.Fatalf("move normalization drift: %+v", got)
	}
}
