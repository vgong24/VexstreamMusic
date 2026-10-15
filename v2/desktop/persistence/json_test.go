package persistence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSONDistinguishesMissingFromCorrupt(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state.json")
	var state map[string]string
	found, err := LoadJSON(path, &state)
	if err != nil || found {
		t.Fatalf("missing contract drift: found=%v err=%v", found, err)
	}
	if err := os.WriteFile(path, []byte("{bad-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	found, err = LoadJSON(path, &state)
	if err == nil || !found {
		t.Fatalf("corrupt contract drift: found=%v err=%v", found, err)
	}
}

func TestSaveJSONAtomicRoundTripsAndCleansFailedTemporary(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "state.json")
	input := map[string][]string{"refs": {"a", "b"}}
	if err := SaveJSONAtomic(path, input); err != nil {
		t.Fatal(err)
	}
	var output map[string][]string
	found, err := LoadJSON(path, &output)
	if err != nil || !found || len(output["refs"]) != 2 {
		t.Fatalf("round trip drift: found=%v state=%+v err=%v", found, output, err)
	}

	bad := filepath.Join(root, "destination-directory")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSONAtomic(bad, input); err == nil {
		t.Fatal("rename-to-directory failure was not surfaced")
	}
	if _, err := os.Stat(bad + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("failed atomic write left temporary file: %v", err)
	}
}
