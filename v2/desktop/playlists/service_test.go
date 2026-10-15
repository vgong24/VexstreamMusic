package playlists

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func openTest(t *testing.T) *Service {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "playlists.json"))
	if err != nil {
		t.Fatal(err)
	}
	tick := 0
	s.now = func() time.Time { tick++; return time.Date(2026, 10, 14, 0, 0, tick, 0, time.UTC) }
	s.newID = func() (string, error) { return "playlist-test", nil }
	return s
}

func TestOpenPreservesV3RawArrayAndMissingTrackRefs(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "playlists.json")
	raw := `[{"id":"p1","name":" Road Trip ","trackIds":["known","missing","known",""],"createdAt":"c","updatedAt":"u"}]`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	rows := s.Snapshot()
	if len(rows) != 1 || rows[0].Name != "Road Trip" || strings.Join(rows[0].TrackRefs, ",") != "known,missing" {
		t.Fatalf("compatibility drift: %+v", rows)
	}
}

func TestOpenFailsClosedOnCorruptDurablePlaylistState(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "playlists.json")
	_ = os.WriteFile(path, []byte("{bad"), 0o644)
	if _, err := Open(path); err == nil {
		t.Fatal("corrupt playlist state was silently dropped")
	}
}

func TestCreateCleansNameDedupesRefsAndPersists(t *testing.T) {
	s := openTest(t)
	row, err := s.Create("  Road Trip  ", []string{"a", "a", " missing ", ""})
	if err != nil {
		t.Fatal(err)
	}
	if row.Name != "Road Trip" || strings.Join(row.TrackRefs, ",") != "a,missing" {
		t.Fatalf("create drift: %+v", row)
	}
	reopened, err := Open(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Snapshot(); len(got) != 1 || got[0].ID != "playlist-test" {
		t.Fatalf("persistence drift: %+v", got)
	}
}

func TestAddAndRemovePreserveOrderAndUnknownRefs(t *testing.T) {
	s := openTest(t)
	_, _ = s.Create("Mix", []string{"a", "missing"})
	row, added, err := s.Add("playlist-test", []string{"b", "a", "gone"})
	if err != nil || added != 2 || strings.Join(row.TrackRefs, ",") != "a,missing,b,gone" {
		t.Fatalf("add drift: row=%+v added=%d err=%v", row, added, err)
	}
	row, err = s.Remove("playlist-test", []string{"missing", "not-present"})
	if err != nil || strings.Join(row.TrackRefs, ",") != "a,b,gone" {
		t.Fatalf("remove drift: %+v err=%v", row, err)
	}
}

func TestReorderRequiresSameReferenceSetAndPreservesMissingRefs(t *testing.T) {
	s := openTest(t)
	_, _ = s.Create("Mix", []string{"a", "missing", "b"})
	if err := s.Reorder("playlist-test", []string{"missing", "b", "a"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(s.Snapshot()[0].TrackRefs, ","); got != "missing,b,a" {
		t.Fatalf("reorder drift: %s", got)
	}
	before := s.Snapshot()[0].UpdatedAt
	if err := s.Reorder("playlist-test", []string{"a", "b"}); !errors.Is(err, ErrOrderMismatch) {
		t.Fatalf("missing ref reorder accepted: %v", err)
	}
	if s.Snapshot()[0].UpdatedAt != before {
		t.Fatal("rejected reorder mutated playlist")
	}
}

func TestRenameCapsHumanNameAt120Runes(t *testing.T) {
	s := openTest(t)
	_, _ = s.Create("Mix", nil)
	row, err := s.Rename("playlist-test", strings.Repeat("界", 130))
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(row.Name)) != 120 {
		t.Fatalf("name cap drift: %d", len([]rune(row.Name)))
	}
}

func TestPersistenceFailureDoesNotReplaceLivePlaylistTruth(t *testing.T) {
	s := openTest(t)
	_, _ = s.Create("Mix", []string{"a"})
	before := s.Snapshot()[0]
	bad := filepath.Join(filepath.Dir(s.path), "blocked")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	s.path = bad
	if _, _, err := s.Add("playlist-test", []string{"b"}); err == nil {
		t.Fatal("write failure not surfaced")
	}
	after := s.Snapshot()[0]
	if strings.Join(after.TrackRefs, ",") != strings.Join(before.TrackRefs, ",") {
		t.Fatalf("failed write mutated live truth: before=%+v after=%+v", before, after)
	}
}

func TestDeleteRemovesOnlyPlaylistState(t *testing.T) {
	s := openTest(t)
	_, _ = s.Create("Mix", []string{"a", "missing"})
	if err := s.Delete("playlist-test"); err != nil {
		t.Fatal(err)
	}
	if len(s.Snapshot()) != 0 {
		t.Fatal("playlist delete retained row")
	}
	if err := s.Delete("playlist-test"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete drift: %v", err)
	}
}
