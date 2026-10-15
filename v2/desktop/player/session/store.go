// Package session owns Desktop V4 listening-session persistence.
//
// This is composition state, not Queue semantics: it remembers the durable listening context needed
// to reconstruct Queue truth after a clean desktop restart without auto-starting Playback.
package session

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

const SchemaVersion = "vexstream.desktop-player-session/v1"

var ErrUnsupportedSchema = errors.New("unsupported player session schema")

type Snapshot struct {
	SchemaVersion string   `json:"schemaVersion"`
	Items         []string `json:"items"`
	CurrentIndex  int      `json:"currentIndex"`
	CurrentRef    string   `json:"currentRef"`
	Shuffle       bool     `json:"shuffle"`
	Repeat        bool     `json:"repeat"`
	UpdatedAt     string   `json:"updatedAt,omitempty"`
}

type Store struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
}

func New(path string) *Store {
	return &Store{
		path: strings.TrimSpace(path),
		now:  time.Now,
	}
}

func Empty() Snapshot {
	return Snapshot{
		SchemaVersion: SchemaVersion,
		Items:         []string{},
		CurrentIndex:  -1,
	}
}

func (s *Store) Load() (Snapshot, bool, error) {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return Snapshot{}, false, errors.New("player session path required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnlocked()
}

func (s *Store) Save(input Snapshot) (Snapshot, error) {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return Snapshot{}, errors.New("player session path required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, found, err := s.loadUnlocked(); err != nil {
		// A protected older V4 must never overwrite a future schema it cannot understand.
		return Snapshot{}, err
	} else if !found {
		// Missing is the only absence that permits forming v1 from scratch.
	}

	next, err := normalize(input)
	if err != nil {
		return Snapshot{}, err
	}
	next.SchemaVersion = SchemaVersion
	next.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
	if err := persistence.SaveJSONAtomic(s.path, next); err != nil {
		return Snapshot{}, err
	}
	return clone(next), nil
}

func (s *Store) loadUnlocked() (Snapshot, bool, error) {
	var stored Snapshot
	found, err := persistence.LoadJSON(s.path, &stored)
	if err != nil || !found {
		return Snapshot{}, found, err
	}
	if stored.SchemaVersion != SchemaVersion {
		return Snapshot{}, true, fmt.Errorf(
			"%w: %q",
			ErrUnsupportedSchema,
			stored.SchemaVersion,
		)
	}
	normalized, err := normalize(stored)
	if err != nil {
		return Snapshot{}, true, err
	}
	normalized.SchemaVersion = SchemaVersion
	normalized.UpdatedAt = stored.UpdatedAt
	return clone(normalized), true, nil
}

func normalize(input Snapshot) (Snapshot, error) {
	next := Snapshot{
		SchemaVersion: SchemaVersion,
		Items:         make([]string, 0, len(input.Items)),
		CurrentIndex:  input.CurrentIndex,
		CurrentRef:    strings.TrimSpace(input.CurrentRef),
		Shuffle:       input.Shuffle,
		Repeat:        input.Repeat,
		UpdatedAt:     strings.TrimSpace(input.UpdatedAt),
	}
	for _, ref := range input.Items {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return Snapshot{}, errors.New("player session contains blank queue ref")
		}
		next.Items = append(next.Items, ref)
	}

	if len(next.Items) == 0 {
		next.CurrentIndex = -1
		next.CurrentRef = ""
		return next, nil
	}
	if next.CurrentIndex < 0 || next.CurrentIndex >= len(next.Items) {
		return Snapshot{}, errors.New("player session currentIndex does not address an item")
	}
	if next.CurrentRef == "" ||
		next.Items[next.CurrentIndex] != next.CurrentRef {
		return Snapshot{}, errors.New(
			"player session currentRef does not match currentIndex",
		)
	}
	return next, nil
}

func clone(input Snapshot) Snapshot {
	input.Items = append([]string(nil), input.Items...)
	return input
}
