package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingCatalogIsEmpty(t *testing.T) {
	rows, err := ReadRows(filepath.Join(t.TempDir(), "missing.jsonl"))
	if err != nil || len(rows) != 0 {
		t.Fatalf("missing catalog drift: rows=%+v err=%v", rows, err)
	}
}

func TestReadRowsPreservesValidRowsAroundMalformedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.jsonl")
	data := strings.Join([]string{
		`{"filename":"a.mp3","provider":"youtube","providerId":"a"}`,
		"{bad",
		"",
		`{"filename":"b.mp3","provider":"youtube","providerId":"b"}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadRows(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0]["filename"] != "a.mp3" || rows[1]["filename"] != "b.mp3" {
		t.Fatalf("lenient read drift: %+v", rows)
	}
}

func TestWriteRowsAtomicRoundTripsAndDoesNotHTMLEscapeProviderMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "_vexmedia", "library.jsonl")
	rows := []map[string]any{{
		"filename":   "song.mp3",
		"provider":   "youtube",
		"providerId": "id",
		"metadata":   map[string]any{"description": "A < B & C > D"},
	}}
	if err := WriteRowsAtomic(path, rows); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "\\u003c") || !strings.Contains(string(raw), "A < B & C > D") {
		t.Fatalf("unexpected JSONL escaping: %s", raw)
	}
	loaded, err := ReadRows(path)
	if err != nil || len(loaded) != 1 || loaded[0]["filename"] != "song.mp3" {
		t.Fatalf("round trip drift: %+v %v", loaded, err)
	}
}

func TestUpsertProviderRowsReplacesExactProviderIdentityOnly(t *testing.T) {
	rows := []map[string]any{
		{"provider": "youtube", "providerId": "same", "filename": "old.mp3"},
		{"provider": "other", "providerId": "same", "filename": "other.mp3"},
		{"provider": "youtube", "providerId": "", "filename": "anonymous.mp3"},
	}
	got := UpsertProviderRows(rows, []map[string]any{
		{"provider": "youtube", "providerId": "same", "filename": "new.mp3"},
		{"provider": "youtube", "providerId": "", "filename": "new-anonymous.mp3"},
	})
	if len(got) != 4 {
		t.Fatalf("upsert count drift: %+v", got)
	}
	if got[0]["filename"] != "new.mp3" || got[1]["filename"] != "other.mp3" {
		t.Fatalf("provider identity replacement drift: %+v", got)
	}
	if got[3]["filename"] != "new-anonymous.mp3" {
		t.Fatalf("anonymous row should append: %+v", got)
	}
	if rows[0]["filename"] != "old.mp3" {
		t.Fatal("upsert mutated caller rows")
	}
}
