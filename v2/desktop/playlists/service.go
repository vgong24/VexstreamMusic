// Package playlists owns durable desktop playlist identity, order, naming, and mutation semantics.
// A playlist references catalog track identities; it does not own Library membership or Queue state.
package playlists

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

var (
	ErrNotFound      = errors.New("playlist not found")
	ErrNameRequired  = errors.New("playlist name required")
	ErrOrderMismatch = errors.New("playlist order must contain the same tracks exactly once")
)

type Playlist struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	TrackRefs []string `json:"trackIds"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

type Service struct {
	mu    sync.RWMutex
	path  string
	rows  []Playlist
	now   func() time.Time
	newID func() (string, error)
}

func Open(path string) (*Service, error) {
	rows := []Playlist{}
	found, err := persistence.LoadJSON(path, &rows)
	if err != nil {
		return nil, err
	}
	if !found || rows == nil {
		rows = []Playlist{}
	}
	rows = normalizeRows(rows)
	return &Service{path: path, rows: rows, now: time.Now, newID: randomID}, nil
}

func (s *Service) Snapshot() []Playlist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneRows(s.rows)
}

func (s *Service) Create(name string, refs []string) (Playlist, error) {
	name = cleanName(name)
	if name == "" {
		return Playlist{}, ErrNameRequired
	}
	id, err := s.newID()
	if err != nil {
		return Playlist{}, err
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	row := Playlist{ID: id, Name: name, TrackRefs: normalizeRefs(refs), CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := append(cloneRows(s.rows), cloneRow(row))
	if err := s.commitLocked(next); err != nil {
		return Playlist{}, err
	}
	return cloneRow(row), nil
}

func (s *Service) Add(id string, refs []string) (Playlist, int, error) {
	incoming := normalizeRefs(refs)
	s.mu.Lock()
	defer s.mu.Unlock()
	index := indexOf(s.rows, id)
	if index < 0 {
		return Playlist{}, 0, ErrNotFound
	}
	next := cloneRows(s.rows)
	seen := map[string]bool{}
	for _, ref := range next[index].TrackRefs {
		seen[ref] = true
	}
	added := 0
	for _, ref := range incoming {
		if !seen[ref] {
			next[index].TrackRefs = append(next[index].TrackRefs, ref)
			seen[ref] = true
			added++
		}
	}
	if added == 0 {
		return cloneRow(next[index]), 0, nil
	}
	next[index].UpdatedAt = s.timestamp()
	if err := s.commitLocked(next); err != nil {
		return Playlist{}, 0, err
	}
	return cloneRow(next[index]), added, nil
}

func (s *Service) Reorder(id string, refs []string) error {
	ordered := normalizeRefs(refs)
	s.mu.Lock()
	defer s.mu.Unlock()
	index := indexOf(s.rows, id)
	if index < 0 {
		return ErrNotFound
	}
	current := normalizeRefs(s.rows[index].TrackRefs)
	if !sameRefSet(current, ordered) {
		return ErrOrderMismatch
	}
	next := cloneRows(s.rows)
	next[index].TrackRefs = ordered
	next[index].UpdatedAt = s.timestamp()
	return s.commitLocked(next)
}

func (s *Service) Remove(id string, refs []string) (Playlist, error) {
	wanted := map[string]bool{}
	for _, ref := range normalizeRefs(refs) {
		wanted[ref] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := indexOf(s.rows, id)
	if index < 0 {
		return Playlist{}, ErrNotFound
	}
	next := cloneRows(s.rows)
	kept := make([]string, 0, len(next[index].TrackRefs))
	for _, ref := range next[index].TrackRefs {
		if !wanted[ref] {
			kept = append(kept, ref)
		}
	}
	if len(kept) == len(next[index].TrackRefs) {
		return cloneRow(next[index]), nil
	}
	next[index].TrackRefs = kept
	next[index].UpdatedAt = s.timestamp()
	if err := s.commitLocked(next); err != nil {
		return Playlist{}, err
	}
	return cloneRow(next[index]), nil
}

func (s *Service) Rename(id, name string) (Playlist, error) {
	name = cleanName(name)
	if name == "" {
		return Playlist{}, ErrNameRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := indexOf(s.rows, id)
	if index < 0 {
		return Playlist{}, ErrNotFound
	}
	next := cloneRows(s.rows)
	next[index].Name = name
	next[index].UpdatedAt = s.timestamp()
	if err := s.commitLocked(next); err != nil {
		return Playlist{}, err
	}
	return cloneRow(next[index]), nil
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := indexOf(s.rows, id)
	if index < 0 {
		return ErrNotFound
	}
	next := cloneRows(s.rows)
	next = append(next[:index], next[index+1:]...)
	return s.commitLocked(next)
}

func (s *Service) timestamp() string { return s.now().UTC().Format(time.RFC3339Nano) }
func (s *Service) commitLocked(next []Playlist) error {
	if err := persistence.SaveJSONAtomic(s.path, next); err != nil {
		return err
	}
	s.rows = cloneRows(next)
	return nil
}

func cleanName(name string) string {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) <= 120 {
		return name
	}
	r := []rune(name)
	return string(r[:120])
}
func normalizeRefs(refs []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		out = append(out, ref)
	}
	return out
}
func normalizeRows(rows []Playlist) []Playlist {
	out := make([]Playlist, 0, len(rows))
	for _, row := range rows {
		row.Name = cleanName(row.Name)
		row.TrackRefs = normalizeRefs(row.TrackRefs)
		out = append(out, row)
	}
	return out
}
func indexOf(rows []Playlist, id string) int {
	for i := range rows {
		if rows[i].ID == id {
			return i
		}
	}
	return -1
}
func sameRefSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[string]bool{}
	for _, v := range a {
		set[v] = true
	}
	for _, v := range b {
		if !set[v] {
			return false
		}
	}
	return true
}
func cloneRow(row Playlist) Playlist {
	row.TrackRefs = append([]string(nil), row.TrackRefs...)
	return row
}
func cloneRows(rows []Playlist) []Playlist {
	out := make([]Playlist, len(rows))
	for i, row := range rows {
		out[i] = cloneRow(row)
	}
	return out
}
func randomID() (string, error) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
