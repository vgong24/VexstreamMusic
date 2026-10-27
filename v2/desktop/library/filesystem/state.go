package filesystem

import (
	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/persistence"
)

func LoadFirstSeen(path string) (map[string]string, error) { return loadMap[string](path) }
func SaveFirstSeen(path string, state map[string]string) error {
	return persistence.SaveJSONAtomic(path, state)
}
func LoadOverlays(path string) (map[string]library.Overlay, error) {
	return loadMap[library.Overlay](path)
}
func SaveOverlays(path string, state map[string]library.Overlay) error {
	return persistence.SaveJSONAtomic(path, state)
}
func LoadPlaybackWindows(path string) (map[string]library.PlaybackWindow, error) {
	return loadMap[library.PlaybackWindow](path)
}
func SavePlaybackWindows(path string, state map[string]library.PlaybackWindow) error {
	return persistence.SaveJSONAtomic(path, state)
}

func loadMap[T any](path string) (map[string]T, error) {
	state := map[string]T{}
	found, err := persistence.LoadJSON(path, &state)
	if err != nil {
		return map[string]T{}, err
	}
	if !found || state == nil {
		return map[string]T{}, nil
	}
	return state, nil
}
