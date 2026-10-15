// Package httpapi exposes durable desktop Playlist truth through a read-only local browser surface.
// Playlist availability remains a Library/browser projection concern; missing track refs stay durable.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
)

type SnapshotFunc func() []playlists.Playlist

type Handler struct {
	snapshot SnapshotFunc
}

func New(snapshot SnapshotFunc) *Handler {
	if snapshot == nil {
		panic("playlist http adapter requires snapshot")
	}
	return &Handler{snapshot: snapshot}
}

type response struct {
	Playlists []playlists.Playlist `json:"playlists"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/playlists" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	rows := h.snapshot()
	if rows == nil {
		rows = []playlists.Playlist{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response{Playlists: rows})
}
