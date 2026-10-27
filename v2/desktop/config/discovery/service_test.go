package discovery

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultRootsPreserveV3CommonLocations(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"Music", "Desktop", "Downloads"} {
		if err := os.MkdirAll(filepath.Join(home, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got := DefaultRoots(home, false)
	want := []string{
		filepath.Join(home, "Music"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Downloads"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default root drift: got=%v want=%v", got, want)
	}
	if deep := DefaultRoots(home, true); !reflect.DeepEqual(deep, []string{home}) {
		t.Fatalf("deep root drift: %v", deep)
	}
}

func TestScanFindsMP3FoldersRanksCountAndExcludesIncludedIgnored(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "A")
	b := filepath.Join(root, "B")
	c := filepath.Join(root, "C")
	touch(t, filepath.Join(a, "one.mp3"))
	touch(t, filepath.Join(a, "two.MP3"))
	touch(t, filepath.Join(b, "one.mp3"))
	touch(t, filepath.Join(c, "one.mp3"))

	got := Scan(
		[]string{root},
		[]string{b},
		[]string{c},
		false,
	)
	if len(got.Candidates) != 1 ||
		got.Candidates[0].Path != a ||
		got.Candidates[0].MP3Count != 2 {
		t.Fatalf("candidate projection drift: %+v", got)
	}
}

func TestScanSkipsImplementationAndSymlinkTerrain(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, ".git", "hidden.mp3"))
	touch(t, filepath.Join(root, "node_modules", "hidden.mp3"))
	real := filepath.Join(root, "real")
	touch(t, filepath.Join(real, "song.mp3"))
	link := filepath.Join(root, "linked")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	got := Scan([]string{root}, nil, nil, false)
	if len(got.Candidates) != 1 ||
		got.Candidates[0].Path != real {
		t.Fatalf("ignored terrain leaked into candidates: %+v", got)
	}
}

func TestScanDoesNotPersistCandidateMeaning(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "music")
	touch(t, filepath.Join(candidate, "song.mp3"))

	first := Scan([]string{root}, nil, nil, false)
	if len(first.Candidates) != 1 {
		t.Fatalf("candidate missing: %+v", first)
	}
	second := Scan([]string{root}, []string{candidate}, nil, false)
	if len(second.Candidates) != 0 {
		t.Fatalf("included candidate remained discovery truth: %+v", second)
	}
}

func TestDepthStopsUnboundedNestedTraversal(t *testing.T) {
	root := t.TempDir()
	path := root
	for index := 0; index < 9; index++ {
		path = filepath.Join(path, "d")
	}
	touch(t, filepath.Join(path, "too-deep.mp3"))

	got := Scan([]string{root}, nil, nil, false)
	if len(got.Candidates) != 0 {
		t.Fatalf("over-depth candidate leaked: %+v", got)
	}
}
