// Package httpapi exposes durable desktop Playlist operations to the local V4 browser.
//
// Playlist availability remains a Library/browser projection concern; unresolved track refs stay
// durable. This is a local desktop administration surface, not the future mobile music surface.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
)

const maxRequestBytes = 64 << 10

type Handler struct {
	service *playlists.Service
}

func New(service *playlists.Service) *Handler {
	if service == nil {
		panic("playlist http adapter requires service")
	}
	return &Handler{service: service}
}

type response struct {
	Playlists []playlists.Playlist `json:"playlists"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/playlists":
		h.list(w, r)
	case "/api/playlists/create":
		h.create(w, r)
	case "/api/playlists/add":
		h.add(w, r)
	case "/api/playlists/order":
		h.order(w, r)
	case "/api/playlists/remove":
		h.remove(w, r)
	case "/api/playlists/rename":
		h.rename(w, r)
	case "/api/playlists/delete":
		h.delete(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodGet) {
		return
	}
	rows := h.service.Snapshot()
	if rows == nil {
		rows = []playlists.Playlist{}
	}
	jsonOut(w, http.StatusOK, response{Playlists: rows})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	var request struct {
		Name      string   `json:"name"`
		TrackRefs []string `json:"trackIds"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	row, err := h.service.Create(request.Name, request.TrackRefs)
	if writeServiceError(w, err) {
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{"ok": true, "playlist": row})
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	var request struct {
		ID        string   `json:"id"`
		TrackRefs []string `json:"trackIds"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	request.ID = strings.TrimSpace(request.ID)
	if request.ID == "" {
		jsonError(w, http.StatusBadRequest, "playlist id required")
		return
	}
	row, added, err := h.service.Add(request.ID, request.TrackRefs)
	if writeServiceError(w, err) {
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{
		"ok": true, "added": added, "playlist": row,
	})
}

func (h *Handler) order(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	var request struct {
		ID        string   `json:"id"`
		TrackRefs []string `json:"trackIds"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	request.ID = strings.TrimSpace(request.ID)
	if request.ID == "" {
		jsonError(w, http.StatusBadRequest, "playlist id required")
		return
	}
	if err := h.service.Reorder(request.ID, request.TrackRefs); writeServiceError(w, err) {
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	var request struct {
		ID        string   `json:"id"`
		TrackRefs []string `json:"trackIds"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	request.ID = strings.TrimSpace(request.ID)
	if request.ID == "" {
		jsonError(w, http.StatusBadRequest, "playlist id required")
		return
	}
	row, err := h.service.Remove(request.ID, request.TrackRefs)
	if writeServiceError(w, err) {
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{"ok": true, "playlist": row})
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	var request struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	request.ID = strings.TrimSpace(request.ID)
	if request.ID == "" {
		jsonError(w, http.StatusBadRequest, "playlist id required")
		return
	}
	row, err := h.service.Rename(request.ID, request.Name)
	if writeServiceError(w, err) {
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{"ok": true, "playlist": row})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	var request struct {
		ID      string `json:"id"`
		Confirm string `json:"confirm"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	request.ID = strings.TrimSpace(request.ID)
	if request.ID == "" {
		jsonError(w, http.StatusBadRequest, "playlist id required")
		return
	}
	if request.Confirm != "DELETE_PLAYLIST_ONLY" {
		jsonError(w, http.StatusBadRequest, "explicit playlist-delete confirmation missing")
		return
	}
	if err := h.service.Delete(request.ID); writeServiceError(w, err) {
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{"ok": true})
}

func method(w http.ResponseWriter, r *http.Request, allowed string) bool {
	if r.Method == allowed {
		return true
	}
	w.Header().Set("Allow", allowed)
	jsonError(w, http.StatusMethodNotAllowed, allowed+" required")
	return false
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid playlist request")
		return false
	}
	return true
}

func writeServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, playlists.ErrNotFound):
		jsonError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, playlists.ErrNameRequired):
		jsonError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, playlists.ErrOrderMismatch):
		jsonError(w, http.StatusConflict, err.Error())
	default:
		jsonError(w, http.StatusInternalServerError, err.Error())
	}
	return true
}

func jsonOut(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func jsonError(w http.ResponseWriter, status int, message string) {
	jsonOut(w, status, map[string]string{"error": message})
}
