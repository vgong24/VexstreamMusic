package library

import (
	"path/filepath"
	"sort"
	"strings"
)

// ProjectTrack derives current human-facing catalog metadata from one raw discovered track.
// The input Track remains unchanged; cache/source truth must not absorb display edits.
func ProjectTrack(raw Track, overlay *Overlay, window *PlaybackWindow) Track {
	out := raw
	out.SourceTitle = raw.Title
	out.SourceArtist = raw.Artist
	out.SourceAlbum = raw.Album
	out.SourceGenre = raw.Genre
	out.OverlayUpdatedAt = ""
	out.PlaybackStart = 0
	out.PlaybackEnd = nil
	out.PlaybackUpdatedAt = ""

	// A YouTube uploader/channel is provenance, not automatically a musical artist.
	if raw.ArtistBasis == "youtube.channel" || raw.ArtistBasis == "legacy.youtube.channel.unverified" {
		out.Artist = ""
	}

	if overlay != nil {
		if overlay.Title != nil {
			out.Title = *overlay.Title
		}
		if overlay.Artist != nil {
			out.Artist = *overlay.Artist
		}
		if overlay.Album != nil {
			out.Album = *overlay.Album
		}
		if overlay.Genre != nil {
			out.Genre = *overlay.Genre
		}
		out.OverlayUpdatedAt = overlay.UpdatedAt
	}

	if window != nil {
		if window.Start > 0 {
			out.PlaybackStart = window.Start
		}
		if window.End != nil {
			end := *window.End
			out.PlaybackEnd = &end
		}
		out.PlaybackUpdatedAt = window.UpdatedAt
	}

	return out
}

type Projection struct {
	Tracks []Track
	Paths  map[string]string
}

// ProjectCatalog forms the current desktop catalog from raw cache entries.
// Source filtering belongs here because the cache may retain entries no longer in the active source set.
func ProjectCatalog(
	entries map[string]CacheEntry,
	activeSources []string,
	overlays map[string]Overlay,
	windows map[string]PlaybackWindow,
	useAll bool,
) Projection {
	tracks := make([]Track, 0, len(entries))
	paths := make(map[string]string, len(entries))
	for _, entry := range entries {
		raw := entry.Track
		if !useAll && !sourceAllowed(activeSources, raw.Source) {
			continue
		}

		var overlay *Overlay
		if value, ok := overlays[raw.ID]; ok {
			copy := value
			overlay = &copy
		}
		var window *PlaybackWindow
		if value, ok := windows[raw.ID]; ok {
			copy := value
			if copy.End != nil {
				end := *copy.End
				copy.End = &end
			}
			window = &copy
		}

		tracks = append(tracks, ProjectTrack(raw, overlay, window))
		paths[raw.ID] = entry.Path
	}

	sort.Slice(tracks, func(i, j int) bool {
		left := strings.ToLower(tracks[i].Artist + " " + tracks[i].Album + " " + tracks[i].Title)
		right := strings.ToLower(tracks[j].Artist + " " + tracks[j].Album + " " + tracks[j].Title)
		return left < right
	})

	return Projection{Tracks: tracks, Paths: paths}
}

func sourceAllowed(sources []string, source string) bool {
	for _, active := range sources {
		if strings.EqualFold(filepath.Clean(active), filepath.Clean(source)) {
			return true
		}
	}
	return false
}
