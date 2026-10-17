// Package httpapi exposes the read-only local desktop Library projection to the V4 browser.
// It does not own source-folder mutation, library reconciliation, trash/import/repair, playlists,
// mobile projection, or authorization.
package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

type SnapshotFunc func() library.Projection

type Handler struct {
	snapshot SnapshotFunc
}

func New(snapshot SnapshotFunc) *Handler {
	if snapshot == nil {
		panic("library http adapter requires snapshot")
	}
	return &Handler{snapshot: snapshot}
}

type catalogTrack struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Artist            string   `json:"artist"`
	Album             string   `json:"album"`
	Genre             string   `json:"genre"`
	Year              string   `json:"year"`
	Duration          float64  `json:"duration"`
	ProviderID        string   `json:"providerId,omitempty"`
	SourceURL         string   `json:"sourceUrl,omitempty"`
	Channel           string   `json:"channel,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Categories        []string `json:"categories,omitempty"`
	Description       string   `json:"description,omitempty"`
	ThumbnailURL      string   `json:"thumbnailUrl,omitempty"`
	AddedAt           string   `json:"addedAt,omitempty"`
	SourceTitle       string   `json:"sourceTitle,omitempty"`
	SourceArtist      string   `json:"sourceArtist,omitempty"`
	SourceAlbum       string   `json:"sourceAlbum,omitempty"`
	SourceGenre       string   `json:"sourceGenre,omitempty"`
	OverlayUpdatedAt  string   `json:"overlayUpdatedAt,omitempty"`
	PlaybackStart     float64  `json:"playbackStart,omitempty"`
	PlaybackEnd       *float64 `json:"playbackEnd,omitempty"`
	PlaybackUpdatedAt string   `json:"playbackUpdatedAt,omitempty"`
}

type catalogResponse struct {
	Tracks []catalogTrack `json:"tracks"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/library":
		h.catalog(w, r)
	case strings.HasPrefix(r.URL.Path, "/media/"):
		h.media(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) catalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	projection := h.snapshot()
	tracks := make([]catalogTrack, 0, len(projection.Tracks))
	for _, track := range projection.Tracks {
		tracks = append(tracks, projectTrack(track))
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(catalogResponse{Tracks: tracks})
}

func (h *Handler) media(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "GET or HEAD required", http.StatusMethodNotAllowed)
		return
	}

	escaped := strings.TrimPrefix(r.URL.EscapedPath(), "/media/")
	id, err := url.PathUnescape(escaped)
	if err != nil || id == "" || id == "." || id == ".." || strings.ContainsAny(id, `/\\`) {
		http.NotFound(w, r)
		return
	}

	projection := h.snapshot()
	path := projection.Paths[id]
	if path == "" {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil || !stat.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, filepath.Base(path), stat.ModTime(), file)
}

func projectTrack(track library.Track) catalogTrack {
	return catalogTrack{
		ID: track.ID, Title: track.Title, Artist: track.Artist, Album: track.Album,
		Genre: track.Genre, Year: track.Year, Duration: track.Duration,
		ProviderID: track.ProviderID, SourceURL: track.SourceURL, Channel: track.Channel,
		Tags: append([]string(nil), track.Tags...), Categories: append([]string(nil), track.Categories...),
		Description: track.Description, ThumbnailURL: track.ThumbnailURL, AddedAt: track.AddedAt,
		SourceTitle: track.SourceTitle, SourceArtist: track.SourceArtist, SourceAlbum: track.SourceAlbum,
		SourceGenre: track.SourceGenre, OverlayUpdatedAt: track.OverlayUpdatedAt,
		PlaybackStart: track.PlaybackStart, PlaybackEnd: cloneNumber(track.PlaybackEnd),
		PlaybackUpdatedAt: track.PlaybackUpdatedAt,
	}
}

func cloneNumber(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
