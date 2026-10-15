// Package httpapi exposes local desktop Trash custody.
//
// File custody is owned by Trash. Library reconciliation is a downstream projection effect:
// TRASH_CUSTODY_CHANGED != LIBRARY_RECONCILED.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/trash"
)

type Trash interface {
	Snapshot() trash.Snapshot
	Move([]trash.MoveInput) ([]trash.Item, error)
	Restore([]string) ([]trash.Item, error)
	Purge([]string, bool) (int, error)
	RecoverPurge() (int, error)
}

type SnapshotFunc func() library.Projection
type Refresh func() error

type Handler struct {
	trash    Trash
	library  SnapshotFunc
	refresh  Refresh
}

func New(
	trashService Trash,
	librarySnapshot SnapshotFunc,
	refresh Refresh,
) *Handler {
	if trashService == nil || librarySnapshot == nil {
		panic("trash http adapter requires Trash and Library snapshot")
	}
	return &Handler{
		trash: trashService,
		library: librarySnapshot,
		refresh: refresh,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/library/trash":
		h.trashRoute(w, r)
	case "/api/library/trash/restore":
		h.restore(w, r)
	case "/api/library/trash/purge":
		h.purge(w, r)
	case "/api/library/trash/purge/recover":
		h.recoverPurge(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) trashRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, h.trash.Snapshot())
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "GET or POST required")
		return
	}
	var request struct {
		IDs     []string `json:"ids"`
		Confirm string   `json:"confirm"`
	}
	if !decode(w, r, &request) {
		return
	}
	if request.Confirm != "MOVE_TO_VEXSTREAM_TRASH" {
		writeError(w, http.StatusBadRequest, "explicit trash confirmation missing")
		return
	}
	ids := normalizeIDs(request.IDs)
	if len(ids) == 0 {
		writeError(w, http.StatusBadRequest, "no tracks selected")
		return
	}

	projection := h.library()
	byID := map[string]library.Track{}
	for _, track := range projection.Tracks {
		byID[track.ID] = track
	}
	inputs := make([]trash.MoveInput, 0, len(ids))
	for _, id := range ids {
		track, ok := byID[id]
		path := projection.Paths[id]
		if !ok || strings.TrimSpace(path) == "" {
			writeError(
				w,
				http.StatusNotFound,
				"one or more selected tracks are no longer available",
			)
			return
		}
		inputs = append(inputs, trash.MoveInput{Track: track, Path: path})
	}

	moved, err := h.trash.Move(inputs)
	if err != nil {
		writeTrashError(w, err)
		return
	}
	h.writeCustodyResult(w, map[string]any{
		"ok": true,
		"moved": len(moved),
	})
}

func (h *Handler) restore(w http.ResponseWriter, r *http.Request) {
	ids, ok := decodeIDs(w, r)
	if !ok {
		return
	}
	restored, err := h.trash.Restore(ids)
	if err != nil {
		writeTrashError(w, err)
		return
	}
	h.writeCustodyResult(w, map[string]any{
		"ok": true,
		"restored": len(restored),
	})
}

func (h *Handler) purge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		IDs              []string `json:"ids"`
		Confirm          string   `json:"confirm"`
		PreserveMetadata bool     `json:"preserveMetadata"`
	}
	if !decode(w, r, &request) {
		return
	}
	if request.Confirm != "PERMANENTLY_DELETE_TRASH" {
		writeError(
			w,
			http.StatusBadRequest,
			"explicit permanent-delete confirmation missing",
		)
		return
	}
	ids := normalizeIDs(request.IDs)
	if len(ids) == 0 {
		writeError(w, http.StatusBadRequest, "no trash items selected")
		return
	}
	count, err := h.trash.Purge(ids, request.PreserveMetadata)
	if err != nil {
		writeTrashError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"deleted": count,
		"preservedMetadata": request.PreserveMetadata,
	})
}

func (h *Handler) recoverPurge(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		Confirm string `json:"confirm"`
	}
	if !decode(w, r, &request) {
		return
	}
	if request.Confirm != "RECOVER_ADMITTED_TRASH_PURGE" {
		writeError(
			w,
			http.StatusBadRequest,
			"explicit purge-recovery confirmation missing",
		)
		return
	}
	count, err := h.trash.RecoverPurge()
	if err != nil {
		writeTrashError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"recovered": count,
	})
}

func (h *Handler) writeCustodyResult(
	w http.ResponseWriter,
	response map[string]any,
) {
	if h.refresh != nil {
		if err := h.refresh(); err != nil {
			response["libraryRefresh"] = "FAILED"
			response["warning"] = err.Error()
			writeJSON(w, http.StatusOK, response)
			return
		}
		response["libraryRefresh"] = "CURRENT"
	}
	writeJSON(w, http.StatusOK, response)
}

func decodeIDs(
	w http.ResponseWriter,
	r *http.Request,
) ([]string, bool) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return nil, false
	}
	var request struct {
		IDs []string `json:"ids"`
	}
	if !decode(w, r, &request) {
		return nil, false
	}
	ids := normalizeIDs(request.IDs)
	if len(ids) == 0 {
		writeError(w, http.StatusBadRequest, "no trash items selected")
		return nil, false
	}
	return ids, true
}

func normalizeIDs(values []string) []string {
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

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid Trash request")
		return false
	}
	return true
}

func writeTrashError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, trash.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, trash.ErrRestoreCollision),
		errors.Is(err, trash.ErrPurgeRecoveryRequired):
		status = http.StatusConflict
	}
	writeError(w, status, err.Error())
}

func methodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, method+" required")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
