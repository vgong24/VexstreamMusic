package session

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

func TestSaveAndLoadPreserveExactCurrentEntryWithDuplicateRefs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player-session.json")
	store := New(path)
	store.now = func() time.Time {
		return time.Date(2026, 10, 14, 22, 45, 1, 2, time.UTC)
	}

	saved, err := store.Save(Snapshot{
		Items:        []string{"a", "b", "a"},
		CurrentIndex: 2,
		CurrentRef:   "a",
		Shuffle:      true,
		Repeat:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.SchemaVersion != SchemaVersion ||
		saved.CurrentIndex != 2 ||
		saved.CurrentRef != "a" ||
		!saved.Shuffle ||
		!saved.Repeat ||
		saved.UpdatedAt != "2026-10-14T22:45:01.000000002Z" {
		t.Fatalf("saved session drift: %+v", saved)
	}

	loaded, found, err := store.Load()
	if err != nil || !found {
		t.Fatalf("load failed: found=%v err=%v", found, err)
	}
	if len(loaded.Items) != 3 ||
		loaded.Items[2] != "a" ||
		loaded.CurrentIndex != 2 {
		t.Fatalf("loaded session drift: %+v", loaded)
	}
}

func TestEmptySessionNormalizesCurrentWitness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player-session.json")
	store := New(path)
	saved, err := store.Save(Snapshot{
		Items:        nil,
		CurrentIndex: 9,
		CurrentRef:   "stale",
		Shuffle:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.CurrentIndex != -1 ||
		saved.CurrentRef != "" ||
		len(saved.Items) != 0 ||
		!saved.Shuffle {
		t.Fatalf("empty normalization drift: %+v", saved)
	}
}

func TestNonEmptySessionRequiresCurrentIdentityAndIndexAgreement(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "player-session.json"))
	cases := []Snapshot{
		{Items: []string{"a"}, CurrentIndex: -1, CurrentRef: "a"},
		{Items: []string{"a"}, CurrentIndex: 1, CurrentRef: "a"},
		{Items: []string{"a", "b"}, CurrentIndex: 0, CurrentRef: "b"},
		{Items: []string{"a", ""}, CurrentIndex: 0, CurrentRef: "a"},
	}
	for _, input := range cases {
		if _, err := store.Save(input); err == nil {
			t.Fatalf("invalid session accepted: %+v", input)
		}
	}
}

func TestFutureSchemaIsHeldAndCannotBeOverwrittenByOlderV4(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player-session.json")
	if err := persistence.SaveJSONAtomic(path, map[string]any{
		"schemaVersion": "vexstream.desktop-player-session/v99",
		"items": []string{"future"},
		"currentIndex": 0,
		"currentRef": "future",
	}); err != nil {
		t.Fatal(err)
	}

	store := New(path)
	if _, found, err := store.Load(); !found ||
		!errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("future schema load drift: found=%v err=%v", found, err)
	}
	if _, err := store.Save(Snapshot{
		Items: []string{"old"},
		CurrentIndex: 0,
		CurrentRef: "old",
	}); !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("future schema was overwritten: %v", err)
	}

	var raw map[string]any
	found, err := persistence.LoadJSON(path, &raw)
	if err != nil || !found ||
		raw["schemaVersion"] != "vexstream.desktop-player-session/v99" {
		t.Fatalf("future schema bytes lost: %+v found=%v err=%v", raw, found, err)
	}
}

func TestMissingSessionLoadsAsAbsent(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "missing.json"))
	got, found, err := store.Load()
	if err != nil || found || len(got.Items) != 0 {
		t.Fatalf("missing session drift: %+v found=%v err=%v", got, found, err)
	}
}
