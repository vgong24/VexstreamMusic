// Package service composes VexStream's desktop Library truth from source discovery, raw cache,
// human projection state, and the current active source set.
//
// It is intentionally not the source-folder/configuration owner, browser/API owner, playlist owner,
// import owner, or playback owner. Callers supply the active sources and consume a projected catalog.
package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/filesystem"
)

type Paths struct {
	Cache           string
	FirstSeen       string
	Overlays        string
	PlaybackWindows string
}

type OpenResult struct {
	Library              *Library
	DiscardedLegacyCache bool
}

type Library struct {
	mu          sync.RWMutex
	reconcileMu sync.Mutex
	paths       Paths
	entries   map[string]library.CacheEntry
	firstSeen map[string]string
	overlays  map[string]library.Overlay
	windows   map[string]library.PlaybackWindow
}

func Open(paths Paths) (OpenResult, error) {
	cache, err := filesystem.LoadCache(paths.Cache)
	if err != nil {
		return OpenResult{}, fmt.Errorf("load library cache: %w", err)
	}
	firstSeen, err := filesystem.LoadFirstSeen(paths.FirstSeen)
	if err != nil {
		return OpenResult{}, fmt.Errorf("load first-seen state: %w", err)
	}
	overlays, err := filesystem.LoadOverlays(paths.Overlays)
	if err != nil {
		return OpenResult{}, fmt.Errorf("load track overlays: %w", err)
	}
	windows, err := filesystem.LoadPlaybackWindows(paths.PlaybackWindows)
	if err != nil {
		return OpenResult{}, fmt.Errorf("load playback windows: %w", err)
	}

	return OpenResult{
		Library: &Library{
			paths:     paths,
			entries:   cloneEntries(cache.Entries),
			firstSeen: cloneStrings(firstSeen),
			overlays:  cloneOverlays(overlays),
			windows:   cloneWindows(windows),
		},
		DiscardedLegacyCache: cache.DiscardedLegacy,
	}, nil
}

// Snapshot derives the current human-facing desktop catalog from already-loaded raw truth.
// useAll is reserved for recovery projection when configuration/source truth is unavailable.
func (l *Library) Snapshot(activeSources []string, useAll bool) library.Projection {
	l.mu.RLock()
	entries := cloneEntries(l.entries)
	overlays := cloneOverlays(l.overlays)
	windows := cloneWindows(l.windows)
	l.mu.RUnlock()
	return library.ProjectCatalog(
		entries,
		append([]string(nil), activeSources...),
		overlays,
		windows,
		useAll,
	)
}

// Reconcile scans the supplied active sources, preserves first-seen identity, persists rebuildable
// raw cache plus durable first-seen state, then publishes the derived human-facing projection.
//
// FirstSeen is persisted before Cache deliberately: if the second write fails, a later retry can
// rebuild raw cache without silently minting a new AddedAt identity for already-observed media.
func (l *Library) Reconcile(activeSources []string) (library.Projection, error) {
	l.reconcileMu.Lock()
	defer l.reconcileMu.Unlock()

	l.mu.RLock()
	previous := cloneEntries(l.entries)
	firstSeen := cloneStrings(l.firstSeen)
	l.mu.RUnlock()

	scan, err := filesystem.Scan(filesystem.ScanInput{
		Sources:   append([]string(nil), activeSources...),
		Previous:  previous,
		FirstSeen: firstSeen,
	})
	if err != nil {
		return library.Projection{}, fmt.Errorf("scan library sources: %w", err)
	}

	if err := filesystem.SaveFirstSeen(l.paths.FirstSeen, scan.FirstSeen); err != nil {
		return library.Projection{}, fmt.Errorf("persist first-seen state: %w", err)
	}
	// FirstSeen is durable identity, not rebuildable cache. Adopt the successfully persisted
	// identity in memory immediately so a same-process retry cannot mint a different AddedAt value
	// if the following cache write fails.
	l.mu.Lock()
	l.firstSeen = cloneStrings(scan.FirstSeen)
	l.mu.Unlock()
	if err := filesystem.SaveCache(l.paths.Cache, scan.Entries); err != nil {
		return library.Projection{}, fmt.Errorf("persist library cache: %w", err)
	}

	l.mu.Lock()
	l.entries = cloneEntries(scan.Entries)
	l.firstSeen = cloneStrings(scan.FirstSeen)
	l.mu.Unlock()

	return l.Snapshot(activeSources, false), nil
}

type OverlayPatch struct {
	Title  *string
	Artist *string
	Album  *string
	Genre  *string
}

func (l *Library) UpdateOverlays(
	trackRefs []string,
	patch OverlayPatch,
	restore bool,
) (int, error) {
	refs := normalizeRefs(trackRefs)
	if len(refs) == 0 {
		return 0, errors.New("track id required")
	}
	if !restore &&
		patch.Title == nil &&
		patch.Artist == nil &&
		patch.Album == nil &&
		patch.Genre == nil {
		return 0, errors.New("no display fields supplied")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	next := cloneOverlays(l.overlays)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, ref := range refs {
		if restore {
			delete(next, ref)
			continue
		}
		value := next[ref]
		if patch.Title != nil {
			value.Title = cloneStringPointer(patch.Title)
		}
		if patch.Artist != nil {
			value.Artist = cloneStringPointer(patch.Artist)
		}
		if patch.Album != nil {
			value.Album = cloneStringPointer(patch.Album)
		}
		if patch.Genre != nil {
			value.Genre = cloneStringPointer(patch.Genre)
		}
		value.UpdatedAt = now
		next[ref] = value
	}
	if err := filesystem.SaveOverlays(l.paths.Overlays, next); err != nil {
		return 0, fmt.Errorf("persist track overlays: %w", err)
	}
	l.overlays = next
	return len(refs), nil
}

func (l *Library) UpdatePlaybackWindow(
	trackRef string,
	start float64,
	end *float64,
	restore bool,
) (library.PlaybackWindow, bool, error) {
	trackRef = strings.TrimSpace(trackRef)
	if trackRef == "" {
		return library.PlaybackWindow{}, false, errors.New("track id required")
	}
	if start < 0 {
		start = 0
	}
	if start < 0.001 {
		start = 0
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	next := cloneWindows(l.windows)
	if restore || (start == 0 && end == nil) {
		delete(next, trackRef)
		if err := filesystem.SavePlaybackWindows(
			l.paths.PlaybackWindows,
			next,
		); err != nil {
			return library.PlaybackWindow{}, false,
				fmt.Errorf("persist playback windows: %w", err)
		}
		l.windows = next
		return library.PlaybackWindow{}, false, nil
	}

	var clonedEnd *float64
	if end != nil {
		value := *end
		clonedEnd = &value
	}
	window := library.PlaybackWindow{
		Start: start,
		End: clonedEnd,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	next[trackRef] = window
	if err := filesystem.SavePlaybackWindows(
		l.paths.PlaybackWindows,
		next,
	); err != nil {
		return library.PlaybackWindow{}, false,
			fmt.Errorf("persist playback windows: %w", err)
	}
	l.windows = next
	return cloneWindow(window), true, nil
}

func (l *Library) ForgetTrackState(trackRefs []string) error {
	refs := normalizeRefs(trackRefs)
	if len(refs) == 0 {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	firstSeen := cloneStrings(l.firstSeen)
	overlays := cloneOverlays(l.overlays)
	windows := cloneWindows(l.windows)
	for _, ref := range refs {
		delete(firstSeen, ref)
		delete(overlays, ref)
		delete(windows, ref)
	}

	// These stores are independently V3-readable and this cleanup is idempotent. Adopt each
	// successfully persisted state immediately; an interrupted purge journal can safely retry the
	// remaining stores without manufacturing completion.
	if err := filesystem.SaveOverlays(l.paths.Overlays, overlays); err != nil {
		return fmt.Errorf("persist overlay cleanup: %w", err)
	}
	l.overlays = overlays
	if err := filesystem.SavePlaybackWindows(
		l.paths.PlaybackWindows,
		windows,
	); err != nil {
		return fmt.Errorf("persist playback-window cleanup: %w", err)
	}
	l.windows = windows
	if err := filesystem.SaveFirstSeen(l.paths.FirstSeen, firstSeen); err != nil {
		return fmt.Errorf("persist first-seen cleanup: %w", err)
	}
	l.firstSeen = firstSeen
	return nil
}

func normalizeRefs(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneWindow(value library.PlaybackWindow) library.PlaybackWindow {
	if value.End != nil {
		end := *value.End
		value.End = &end
	}
	return value
}

func cloneEntries(input map[string]library.CacheEntry) map[string]library.CacheEntry {
	out := make(map[string]library.CacheEntry, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneStrings(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneOverlays(input map[string]library.Overlay) map[string]library.Overlay {
	out := make(map[string]library.Overlay, len(input))
	for key, value := range input {
		copy := value
		if value.Title != nil {
			value := *value.Title
			copy.Title = &value
		}
		if value.Artist != nil {
			value := *value.Artist
			copy.Artist = &value
		}
		if value.Album != nil {
			value := *value.Album
			copy.Album = &value
		}
		if value.Genre != nil {
			value := *value.Genre
			copy.Genre = &value
		}
		out[key] = copy
	}
	return out
}

func cloneWindows(input map[string]library.PlaybackWindow) map[string]library.PlaybackWindow {
	out := make(map[string]library.PlaybackWindow, len(input))
	for key, value := range input {
		copy := value
		if value.End != nil {
			end := *value.End
			copy.End = &end
		}
		out[key] = copy
	}
	return out
}
