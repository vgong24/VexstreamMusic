// Package httpapi exposes local desktop source-folder configuration.
//
// Configuration owns persisted source/ignore custody. Library reconciliation is a downstream effect
// supplied by composition; CONFIG_CHANGED != LIBRARY_RECONCILED.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	desktopconfig "github.com/vgong24/VexstreamMusic/v2/desktop/config"
)

type Config interface {
	Snapshot() desktopconfig.State
	AddSource(string) (desktopconfig.State, error)
	RemoveSource(string) (desktopconfig.State, error)
	Ignore(string) (desktopconfig.State, error)
	Unignore(string) (desktopconfig.State, error)
	ClearIgnored() (desktopconfig.State, error)
	RestoreBackup() (desktopconfig.State, error)
}

type Refresh func([]string) error
type Picker func() (string, error)

type Handler struct {
	config  Config
	refresh Refresh
	picker  Picker
}

func New(config Config, refresh Refresh) *Handler {
	return NewWithPicker(config, refresh, nil)
}

func NewWithPicker(
	config Config,
	refresh Refresh,
	picker Picker,
) *Handler {
	if config == nil {
		panic("source config http adapter requires config")
	}
	return &Handler{
		config:  config,
		refresh: refresh,
		picker:  picker,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/source/config":
		h.snapshot(w, r)
	case "/api/source/choose":
		h.choose(w, r)
	case "/api/source/add":
		h.add(w, r)
	case "/api/source/remove":
		h.remove(w, r)
	case "/api/candidate/ignore":
		h.ignore(w, r)
	case "/api/candidate/unignore":
		h.unignore(w, r)
	case "/api/ignored/clear":
		h.clearIgnored(w, r)
	case "/api/source/config/restore-backup":
		h.restoreBackup(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) choose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if h.picker == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "folder picker unavailable",
		})
		return
	}
	path, err := h.picker()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	path = strings.TrimSpace(path)
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "folder selection cancelled",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": path})
}

func (h *Handler) snapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, h.config.Snapshot())
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	path, ok := decodePath(w, r)
	if !ok {
		return
	}
	state, err := h.config.AddSource(path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.writeMutation(w, state, true)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	path, ok := decodePath(w, r)
	if !ok {
		return
	}
	state, err := h.config.RemoveSource(path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.writeMutation(w, state, true)
}

func (h *Handler) ignore(w http.ResponseWriter, r *http.Request) {
	path, ok := decodePath(w, r)
	if !ok {
		return
	}
	state, err := h.config.Ignore(path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.writeMutation(w, state, false)
}

func (h *Handler) unignore(w http.ResponseWriter, r *http.Request) {
	path, ok := decodePath(w, r)
	if !ok {
		return
	}
	state, err := h.config.Unignore(path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.writeMutation(w, state, false)
}

func (h *Handler) clearIgnored(w http.ResponseWriter, r *http.Request) {
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
	if request.Confirm != "CLEAR_IGNORED_FOLDERS" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "explicit ignored-folder clear confirmation missing",
		})
		return
	}
	state, err := h.config.ClearIgnored()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.writeMutation(w, state, false)
}

func (h *Handler) restoreBackup(w http.ResponseWriter, r *http.Request) {
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
	if request.Confirm != "RESTORE_SOURCE_CONFIG_BACKUP" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "explicit config-backup restore confirmation missing",
		})
		return
	}
	state, err := h.config.RestoreBackup()
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	h.writeMutation(w, state, true)
}

func (h *Handler) writeMutation(
	w http.ResponseWriter,
	state desktopconfig.State,
	reconcile bool,
) {
	response := map[string]any{
		"ok":     true,
		"config": state,
	}
	if reconcile && h.refresh != nil {
		if err := h.refresh(append([]string(nil), state.Sources...)); err != nil {
			response["libraryRefresh"] = "FAILED"
			response["warning"] = err.Error()
			writeJSON(w, http.StatusOK, response)
			return
		}
		response["libraryRefresh"] = "CURRENT"
	}
	writeJSON(w, http.StatusOK, response)
}

func decodePath(w http.ResponseWriter, r *http.Request) (string, bool) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return "", false
	}
	var request struct {
		Path string `json:"path"`
	}
	if !decode(w, r, &request) {
		return "", false
	}
	request.Path = strings.TrimSpace(request.Path)
	if request.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path required"})
		return "", false
	}
	return request.Path, true
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid source configuration request"})
		return false
	}
	return true
}

func methodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": method + " required"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

var _ = errors.Is
