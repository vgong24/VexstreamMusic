// Package service composes VexStream's desktop Library truth from source discovery, raw cache,
// human projection state, and the current active source set.
//
// It is intentionally not the source-folder/configuration owner, browser/API owner, playlist owner,
// import owner, or playback owner. Callers supply the active sources and consume a projected catalog.
package service

import (
	"fmt"

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
	paths     Paths
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
	return library.ProjectCatalog(
		cloneEntries(l.entries),
		append([]string(nil), activeSources...),
		cloneOverlays(l.overlays),
		cloneWindows(l.windows),
		useAll,
	)
}

// Reconcile scans the supplied active sources, preserves first-seen identity, persists rebuildable
// raw cache plus durable first-seen state, then publishes the derived human-facing projection.
//
// FirstSeen is persisted before Cache deliberately: if the second write fails, a later retry can
// rebuild raw cache without silently minting a new AddedAt identity for already-observed media.
func (l *Library) Reconcile(activeSources []string) (library.Projection, error) {
	scan, err := filesystem.Scan(filesystem.ScanInput{
		Sources:   append([]string(nil), activeSources...),
		Previous:  cloneEntries(l.entries),
		FirstSeen: cloneStrings(l.firstSeen),
	})
	if err != nil {
		return library.Projection{}, fmt.Errorf("scan library sources: %w", err)
	}

	if err := filesystem.SaveFirstSeen(l.paths.FirstSeen, scan.FirstSeen); err != nil {
		return library.Projection{}, fmt.Errorf("persist first-seen state: %w", err)
	}
	if err := filesystem.SaveCache(l.paths.Cache, scan.Entries); err != nil {
		return library.Projection{}, fmt.Errorf("persist library cache: %w", err)
	}

	l.entries = cloneEntries(scan.Entries)
	l.firstSeen = cloneStrings(scan.FirstSeen)

	return l.Snapshot(activeSources, false), nil
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
