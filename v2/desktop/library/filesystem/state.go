package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

func LoadFirstSeen(path string) (map[string]string, error) {
	return loadMap[string](path)
}

func SaveFirstSeen(path string, state map[string]string) error {
	return saveJSONAtomic(path, state)
}

func LoadOverlays(path string) (map[string]library.Overlay, error) {
	return loadMap[library.Overlay](path)
}

func SaveOverlays(path string, state map[string]library.Overlay) error {
	return saveJSONAtomic(path, state)
}

func LoadPlaybackWindows(path string) (map[string]library.PlaybackWindow, error) {
	return loadMap[library.PlaybackWindow](path)
}

func SavePlaybackWindows(path string, state map[string]library.PlaybackWindow) error {
	return saveJSONAtomic(path, state)
}

func loadMap[T any](path string) (map[string]T, error) {
	state := map[string]T{}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return map[string]T{}, err
	}
	if state == nil {
		state = map[string]T{}
	}
	return state, nil
}

func saveJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
