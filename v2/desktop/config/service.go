// Package config owns VexStream desktop source-folder configuration.
//
// The on-disk JSON shape deliberately remains V3-compatible while V3 is the protected rollback
// product:
//
//	{"sources":[...],"ignored":[...]}
//
// PERSISTED_STATE_UPGRADE != PERMISSION_TO_BREAK_ROLLBACK.
// New product-only fields belong in another store rather than widening this shared file.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

type State struct {
	Sources []string `json:"sources"`
	Ignored []string `json:"ignored"`
}

type Service struct {
	mu         sync.RWMutex
	path       string
	backupPath string
	state      State
}

func Open(path, backupPath string) (*Service, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("config path required")
	}
	state := State{Sources: []string{}, Ignored: []string{}}
	found, err := persistence.LoadJSON(path, &state)
	if err != nil {
		return nil, fmt.Errorf("load desktop config: %w", err)
	}
	if !found {
		state = State{Sources: []string{}, Ignored: []string{}}
	}
	state = normalize(state)
	return &Service{
		path:       path,
		backupPath: strings.TrimSpace(backupPath),
		state:      state,
	}, nil
}

func (s *Service) Snapshot() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneState(s.state)
}

func (s *Service) Sources() []string {
	return s.Snapshot().Sources
}

func (s *Service) IsSource(path string) bool {
	path = cleanPath(path)
	if path == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return containsPath(s.state.Sources, path)
}

func (s *Service) AddSource(path string) (State, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return State{}, errors.New("source path required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return State{}, err
	}
	info, err := os.Stat(absolute)
	if err != nil || !info.IsDir() {
		return State{}, errors.New("folder does not exist")
	}
	absolute = filepath.Clean(absolute)

	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneState(s.state)
	if !containsPath(next.Sources, absolute) {
		next.Sources = append(next.Sources, absolute)
	}
	next.Ignored = removePath(next.Ignored, absolute)
	if err := s.commitLocked(next); err != nil {
		return State{}, err
	}
	return cloneState(s.state), nil
}

func (s *Service) RemoveSource(path string) (State, error) {
	path = cleanPath(path)
	if path == "" {
		return State{}, errors.New("source path required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneState(s.state)
	next.Sources = removePath(next.Sources, path)
	if err := s.commitLocked(next); err != nil {
		return State{}, err
	}
	return cloneState(s.state), nil
}

func (s *Service) Ignore(path string) (State, error) {
	path = cleanPath(path)
	if path == "" {
		return State{}, errors.New("ignored path required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneState(s.state)
	if !containsPath(next.Ignored, path) {
		next.Ignored = append(next.Ignored, path)
	}
	if err := s.commitLocked(next); err != nil {
		return State{}, err
	}
	return cloneState(s.state), nil
}

func (s *Service) Unignore(path string) (State, error) {
	path = cleanPath(path)
	if path == "" {
		return State{}, errors.New("ignored path required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneState(s.state)
	next.Ignored = removePath(next.Ignored, path)
	if err := s.commitLocked(next); err != nil {
		return State{}, err
	}
	return cloneState(s.state), nil
}

func (s *Service) ClearIgnored() (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneState(s.state)
	next.Ignored = []string{}
	if err := s.commitLocked(next); err != nil {
		return State{}, err
	}
	return cloneState(s.state), nil
}

func (s *Service) RestoreBackup() (State, error) {
	if s.backupPath == "" {
		return State{}, errors.New("config backup path unavailable")
	}
	var recovered State
	found, err := persistence.LoadJSON(s.backupPath, &recovered)
	if err != nil {
		return State{}, fmt.Errorf("load config backup: %w", err)
	}
	if !found {
		return State{}, errors.New("config backup not found")
	}
	recovered = normalize(recovered)

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := persistence.SaveJSONAtomic(s.path, recovered); err != nil {
		return State{}, fmt.Errorf("restore config backup: %w", err)
	}
	s.state = recovered
	return cloneState(s.state), nil
}

func (s *Service) commitLocked(next State) error {
	next = normalize(next)
	if s.backupPath != "" {
		if current, err := os.ReadFile(s.path); err == nil {
			if !validV3Shape(current) {
				return errors.New("current config is not V3-compatible JSON")
			}
			if err := writeBytesAtomic(s.backupPath, current); err != nil {
				return fmt.Errorf("backup desktop config: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("read desktop config for backup: %w", err)
		}
	}
	if err := persistence.SaveJSONAtomic(s.path, next); err != nil {
		return fmt.Errorf("persist desktop config: %w", err)
	}
	s.state = cloneState(next)
	return nil
}

func normalize(state State) State {
	return State{
		Sources: normalizePaths(state.Sources),
		Ignored: normalizePaths(state.Ignored),
	}
}

func normalizePaths(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = cleanPath(value)
		if value == "" || containsPath(out, value) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func cleanPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}

func containsPath(values []string, wanted string) bool {
	wanted = cleanPath(wanted)
	for _, value := range values {
		if strings.EqualFold(cleanPath(value), wanted) {
			return true
		}
	}
	return false
}

func removePath(values []string, wanted string) []string {
	wanted = cleanPath(wanted)
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !strings.EqualFold(cleanPath(value), wanted) {
			out = append(out, value)
		}
	}
	return out
}

func cloneState(state State) State {
	return State{
		Sources: append([]string(nil), state.Sources...),
		Ignored: append([]string(nil), state.Ignored...),
	}
}

func validV3Shape(data []byte) bool {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return false
	}
	for key := range raw {
		if key != "sources" && key != "ignored" {
			return false
		}
	}
	var state State
	return json.Unmarshal(data, &state) == nil
}

func writeBytesAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
