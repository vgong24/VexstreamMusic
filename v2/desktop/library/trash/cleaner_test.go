package trash

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/catalog"
)

type stateCleanerFixture struct {
	refs []string
	err error
}

func (s *stateCleanerFixture) ForgetTrackState(refs []string) error {
	s.refs = append([]string(nil), refs...)
	return s.err
}

func TestLibraryCleanerRemovesTrackStateAndSourceCatalogRows(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "music")
	path := filepath.Join(source, "_vexmedia", "library.jsonl")
	if err := catalog.WriteRowsAtomic(path, []map[string]any{
		{"provider": "youtube", "providerId": "a", "filename": "gone.mp3"},
		{"provider": "youtube", "providerId": "b", "filename": "keep.mp3"},
	}); err != nil {
		t.Fatal(err)
	}
	state := &stateCleanerFixture{}
	cleaner := LibraryCleaner{State: state}

	err := cleaner.CleanPurged([]Item{{
		Track: library.Track{
			ID: "track-a",
			Source: source,
			Filename: "gone.mp3",
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(state.refs) != 1 || state.refs[0] != "track-a" {
		t.Fatalf("track-state cleanup drift: %v", state.refs)
	}
	rows, err := catalog.ReadRows(path)
	if err != nil || len(rows) != 1 ||
		rows[0]["filename"] != "keep.mp3" {
		t.Fatalf("source catalog cleanup drift: %+v err=%v", rows, err)
	}
}

func TestLibraryCleanerDoesNotCreateMissingCatalog(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "music")
	state := &stateCleanerFixture{}
	cleaner := LibraryCleaner{State: state}

	if err := cleaner.CleanPurged([]Item{{
		Track: library.Track{
			ID: "track-a",
			Source: source,
			Filename: "gone.mp3",
		},
	}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source, "_vexmedia", "library.jsonl")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("missing source catalog was manufactured: %v", err)
	}
}
