// Package trash owns reversible desktop media custody and journaled permanent deletion.
//
// trash.json and deleted-history.json retain the V3 array shapes so the protected V3 desktop can
// still read them. Irreversible V4 purge progress lives in a separate sidecar journal.
package trash

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

const (
	PurgeJournalSchema      = "vexstream.trash-purge/v1"
	PurgeStatePlanned       = "PLANNED"
	PurgeStateFilesDeleted  = "FILES_DELETED"
	PurgeStateStoresUpdated = "STORES_UPDATED"
)

var (
	ErrNotFound              = errors.New("trash item not found")
	ErrRestoreCollision      = errors.New("trash restore original path is occupied")
	ErrPurgeRecoveryRequired = errors.New("pending trash purge recovery required")
)

type Item struct {
	ID           string        `json:"id"`
	Track        library.Track `json:"track"`
	OriginalPath string        `json:"originalPath"`
	TrashPath    string        `json:"trashPath"`
	TrashedAt    string        `json:"trashedAt"`
}

type DeletedHistoryItem struct {
	ID           string        `json:"id"`
	Track        library.Track `json:"track"`
	OriginalPath string        `json:"originalPath"`
	DeletedAt    string        `json:"deletedAt"`
}

type Paths struct {
	TrashState     string
	DeletedHistory string
	FilesRoot      string
	PurgeJournal   string
}

type MoveInput struct {
	Track library.Track
	Path  string
}

type Cleaner interface {
	CleanPurged([]Item) error
}

type PurgeEntry struct {
	Item    Item                `json:"item"`
	History *DeletedHistoryItem `json:"history,omitempty"`
}

type PurgeJournal struct {
	SchemaVersion string       `json:"schemaVersion"`
	PurgeRef      string       `json:"purgeRef"`
	State         string       `json:"state"`
	Entries       []PurgeEntry `json:"entries"`
}

type Snapshot struct {
	Trash          []Item               `json:"trash"`
	DeletedHistory []DeletedHistoryItem `json:"deletedHistory"`
	PendingPurge   *PurgeJournal        `json:"pendingPurge,omitempty"`
}

type Service struct {
	mu      sync.Mutex
	paths   Paths
	trash   []Item
	history []DeletedHistoryItem
	pending *PurgeJournal
	cleaner Cleaner
	now     func() time.Time
	newID   func() (string, error)
}

func Open(paths Paths, cleaner Cleaner) (*Service, error) {
	if strings.TrimSpace(paths.TrashState) == "" ||
		strings.TrimSpace(paths.DeletedHistory) == "" ||
		strings.TrimSpace(paths.FilesRoot) == "" ||
		strings.TrimSpace(paths.PurgeJournal) == "" {
		return nil, errors.New("trash paths are incomplete")
	}
	items := []Item{}
	if _, err := persistence.LoadJSON(paths.TrashState, &items); err != nil {
		return nil, fmt.Errorf("load trash state: %w", err)
	}
	history := []DeletedHistoryItem{}
	if _, err := persistence.LoadJSON(paths.DeletedHistory, &history); err != nil {
		return nil, fmt.Errorf("load deleted history: %w", err)
	}

	var pending *PurgeJournal
	var journal PurgeJournal
	if found, err := persistence.LoadJSON(paths.PurgeJournal, &journal); err != nil {
		return nil, fmt.Errorf("load trash purge journal: %w", err)
	} else if found {
		if journal.SchemaVersion != PurgeJournalSchema ||
			strings.TrimSpace(journal.PurgeRef) == "" ||
			len(journal.Entries) == 0 {
			return nil, errors.New("trash purge journal is invalid")
		}
		copy := cloneJournal(journal)
		pending = &copy
	}

	return &Service{
		paths:   paths,
		trash:   cloneItems(items),
		history: cloneHistory(history),
		pending: pending,
		cleaner: cleaner,
		now:     time.Now,
		newID:   randomID,
	}, nil
}

func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Snapshot{
		Trash:          cloneItems(s.trash),
		DeletedHistory: cloneHistory(s.history),
	}
	if s.pending != nil {
		copy := cloneJournal(*s.pending)
		out.PendingPurge = &copy
	}
	sort.Slice(out.Trash, func(i, j int) bool {
		return out.Trash[i].TrashedAt > out.Trash[j].TrashedAt
	})
	sort.Slice(out.DeletedHistory, func(i, j int) bool {
		return out.DeletedHistory[i].DeletedAt > out.DeletedHistory[j].DeletedAt
	})
	return out
}

func (s *Service) Move(inputs []MoveInput) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending != nil {
		return nil, ErrPurgeRecoveryRequired
	}

	normalized := normalizeMoveInputs(inputs)
	if len(normalized) == 0 {
		return nil, errors.New("no tracks selected")
	}

	now := s.now().UTC().Format(time.RFC3339Nano)
	moved := make([]Item, 0, len(normalized))
	for _, input := range normalized {
		id, err := s.newID()
		if err != nil {
			rollbackMoves(moved)
			return nil, err
		}
		ext := filepath.Ext(input.Path)
		if ext == "" {
			ext = ".mp3"
		}
		item := Item{
			ID:           id,
			Track:        cloneTrack(input.Track),
			OriginalPath: filepath.Clean(input.Path),
			TrashPath:    filepath.Join(s.paths.FilesRoot, id+ext),
			TrashedAt:    now,
		}
		if err := moveFileSafely(item.OriginalPath, item.TrashPath); err != nil {
			rollbackMoves(moved)
			return nil, fmt.Errorf("move track to trash: %w", err)
		}
		moved = append(moved, item)
	}

	next := append(cloneItems(s.trash), cloneItems(moved)...)
	if err := persistence.SaveJSONAtomic(s.paths.TrashState, next); err != nil {
		rollbackMoves(moved)
		return nil, fmt.Errorf("persist trash custody: %w", err)
	}
	s.trash = next
	return cloneItems(moved), nil
}

func (s *Service) Restore(ids []string) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending != nil {
		return nil, ErrPurgeRecoveryRequired
	}

	wanted := normalizedIDs(ids)
	chosen, err := chooseItems(s.trash, wanted)
	if err != nil {
		return nil, err
	}
	for _, item := range chosen {
		if _, err := os.Stat(item.OriginalPath); err == nil {
			return nil, fmt.Errorf("%w: %s", ErrRestoreCollision, item.OriginalPath)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	restored := make([]Item, 0, len(chosen))
	for _, item := range chosen {
		if err := moveFileSafely(item.TrashPath, item.OriginalPath); err != nil {
			rollbackRestores(restored)
			return nil, fmt.Errorf("restore trash item: %w", err)
		}
		restored = append(restored, item)
	}

	next := removeItems(s.trash, wanted)
	if err := persistence.SaveJSONAtomic(s.paths.TrashState, next); err != nil {
		rollbackRestores(restored)
		return nil, fmt.Errorf("persist restored trash custody: %w", err)
	}
	s.trash = next
	return cloneItems(restored), nil
}

func (s *Service) Purge(
	ids []string,
	preserveMetadata bool,
) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending != nil {
		return 0, ErrPurgeRecoveryRequired
	}

	wanted := normalizedIDs(ids)
	chosen, err := chooseItems(s.trash, wanted)
	if err != nil {
		return 0, err
	}
	purgeRef, err := s.newID()
	if err != nil {
		return 0, err
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	journal := PurgeJournal{
		SchemaVersion: PurgeJournalSchema,
		PurgeRef:      purgeRef,
		State:         PurgeStatePlanned,
		Entries:       make([]PurgeEntry, 0, len(chosen)),
	}
	for _, item := range chosen {
		entry := PurgeEntry{Item: cloneItem(item)}
		if preserveMetadata {
			historyID, err := s.newID()
			if err != nil {
				return 0, err
			}
			entry.History = &DeletedHistoryItem{
				ID:           historyID,
				Track:        cloneTrack(item.Track),
				OriginalPath: item.OriginalPath,
				DeletedAt:    now,
			}
		}
		journal.Entries = append(journal.Entries, entry)
	}
	if err := persistence.SaveJSONAtomic(s.paths.PurgeJournal, journal); err != nil {
		return 0, fmt.Errorf("persist purge intent: %w", err)
	}
	copy := cloneJournal(journal)
	s.pending = &copy
	return s.resumePurgeLocked()
}

func (s *Service) RecoverPurge() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil {
		return 0, nil
	}
	return s.resumePurgeLocked()
}

func (s *Service) resumePurgeLocked() (int, error) {
	if s.pending == nil {
		return 0, nil
	}
	journal := cloneJournal(*s.pending)

	if journal.State == PurgeStatePlanned {
		for _, entry := range journal.Entries {
			if err := os.Remove(entry.Item.TrashPath); err != nil && !os.IsNotExist(err) {
				return 0, fmt.Errorf(
					"delete purged media %s: %w",
					entry.Item.TrashPath,
					err,
				)
			}
		}
		journal.State = PurgeStateFilesDeleted
		if err := persistence.SaveJSONAtomic(
			s.paths.PurgeJournal,
			journal,
		); err != nil {
			return 0, fmt.Errorf("persist purge file-deletion state: %w", err)
		}
		copy := cloneJournal(journal)
		s.pending = &copy
	}

	if journal.State == PurgeStateFilesDeleted {
		nextHistory := cloneHistory(s.history)
		for _, entry := range journal.Entries {
			if entry.History != nil &&
				!historyContains(nextHistory, entry.History.ID) {
				nextHistory = append(nextHistory, cloneHistoryItem(*entry.History))
			}
		}
		if err := persistence.SaveJSONAtomic(
			s.paths.DeletedHistory,
			nextHistory,
		); err != nil {
			return 0, fmt.Errorf("persist deleted history: %w", err)
		}
		s.history = nextHistory

		ids := make([]string, 0, len(journal.Entries))
		for _, entry := range journal.Entries {
			ids = append(ids, entry.Item.ID)
		}
		nextTrash := removeItems(s.trash, normalizedIDs(ids))
		if err := persistence.SaveJSONAtomic(
			s.paths.TrashState,
			nextTrash,
		); err != nil {
			return 0, fmt.Errorf("persist purged trash state: %w", err)
		}
		s.trash = nextTrash

		journal.State = PurgeStateStoresUpdated
		if err := persistence.SaveJSONAtomic(
			s.paths.PurgeJournal,
			journal,
		); err != nil {
			return 0, fmt.Errorf("persist purge store state: %w", err)
		}
		copy := cloneJournal(journal)
		s.pending = &copy
	}

	if s.cleaner == nil {
		return 0, errors.New("purge cleanup owner unavailable")
	}
	items := make([]Item, 0, len(journal.Entries))
	for _, entry := range journal.Entries {
		items = append(items, cloneItem(entry.Item))
	}
	if err := s.cleaner.CleanPurged(items); err != nil {
		return 0, fmt.Errorf("clean purged Library state: %w", err)
	}
	if err := os.Remove(s.paths.PurgeJournal); err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("remove completed purge journal: %w", err)
	}
	s.pending = nil
	return len(items), nil
}

func normalizeMoveInputs(inputs []MoveInput) []MoveInput {
	seen := map[string]bool{}
	out := make([]MoveInput, 0, len(inputs))
	for _, input := range inputs {
		input.Path = strings.TrimSpace(input.Path)
		input.Track.ID = strings.TrimSpace(input.Track.ID)
		if input.Path == "" || input.Track.ID == "" || seen[input.Track.ID] {
			continue
		}
		seen[input.Track.ID] = true
		out = append(out, MoveInput{
			Track: cloneTrack(input.Track),
			Path:  filepath.Clean(input.Path),
		})
	}
	return out
}

func normalizedIDs(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out[value] = true
		}
	}
	return out
}

func chooseItems(items []Item, wanted map[string]bool) ([]Item, error) {
	if len(wanted) == 0 {
		return nil, errors.New("no trash items selected")
	}
	chosen := make([]Item, 0, len(wanted))
	for _, item := range items {
		if wanted[item.ID] {
			chosen = append(chosen, cloneItem(item))
		}
	}
	if len(chosen) != len(wanted) {
		return nil, ErrNotFound
	}
	return chosen, nil
}

func removeItems(items []Item, wanted map[string]bool) []Item {
	out := make([]Item, 0, len(items))
	for _, item := range items {
		if !wanted[item.ID] {
			out = append(out, cloneItem(item))
		}
	}
	return out
}

func historyContains(items []DeletedHistoryItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func rollbackMoves(items []Item) {
	for index := len(items) - 1; index >= 0; index-- {
		_ = moveFileSafely(items[index].TrashPath, items[index].OriginalPath)
	}
}

func rollbackRestores(items []Item) {
	for index := len(items) - 1; index >= 0; index-- {
		_ = moveFileSafely(items[index].OriginalPath, items[index].TrashPath)
	}
}

func moveFileSafely(source, destination string) error {
	source = strings.TrimSpace(source)
	destination = strings.TrimSpace(destination)
	if source == "" || destination == "" {
		return errors.New("missing source or destination")
	}
	if _, err := os.Stat(destination); err == nil {
		return errors.New("destination already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := os.Rename(source, destination); err == nil {
		return nil
	}

	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	temporary := destination + ".partial"
	output, err := os.OpenFile(
		temporary,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	syncErr := output.Sync()
	closeErr := output.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(temporary)
		if copyErr != nil {
			return copyErr
		}
		if syncErr != nil {
			return syncErr
		}
		return closeErr
	}
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	if err := os.Remove(source); err != nil {
		_ = os.Remove(destination)
		return err
	}
	return nil
}

func cloneTrack(track library.Track) library.Track {
	track.Tags = append([]string(nil), track.Tags...)
	track.Categories = append([]string(nil), track.Categories...)
	if track.PlaybackEnd != nil {
		value := *track.PlaybackEnd
		track.PlaybackEnd = &value
	}
	return track
}

func cloneItem(item Item) Item {
	item.Track = cloneTrack(item.Track)
	return item
}

func cloneItems(items []Item) []Item {
	out := make([]Item, len(items))
	for index, item := range items {
		out[index] = cloneItem(item)
	}
	return out
}

func cloneHistoryItem(item DeletedHistoryItem) DeletedHistoryItem {
	item.Track = cloneTrack(item.Track)
	return item
}

func cloneHistory(items []DeletedHistoryItem) []DeletedHistoryItem {
	out := make([]DeletedHistoryItem, len(items))
	for index, item := range items {
		out[index] = cloneHistoryItem(item)
	}
	return out
}

func cloneJournal(journal PurgeJournal) PurgeJournal {
	journal.Entries = append([]PurgeEntry(nil), journal.Entries...)
	for index := range journal.Entries {
		journal.Entries[index].Item = cloneItem(journal.Entries[index].Item)
		if journal.Entries[index].History != nil {
			copy := cloneHistoryItem(*journal.Entries[index].History)
			journal.Entries[index].History = &copy
		}
	}
	return journal
}

func randomID() (string, error) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
