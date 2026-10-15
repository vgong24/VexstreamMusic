// Package httpapi exposes local-only reversible Library edit operations.
//
// Raw media metadata remains provider/filesystem truth. These routes mutate only VexStream-owned
// display overlays and playback-window projections.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	libraryservice "github.com/vgong24/VexstreamMusic/v2/desktop/library/service"
)

type SnapshotFunc func() library.Projection

type Mutator interface {
	UpdateOverlays(
		[]string,
		libraryservice.OverlayPatch,
		bool,
	) (int, error)
	UpdatePlaybackWindow(
		string,
		float64,
		*float64,
		bool,
	) (library.PlaybackWindow, bool, error)
}

type Handler struct {
	snapshot SnapshotFunc
	mutator  Mutator
}

func New(snapshot SnapshotFunc, mutator Mutator) *Handler {
	if snapshot == nil || mutator == nil {
		panic("library admin http adapter requires snapshot and mutator")
	}
	return &Handler{snapshot: snapshot, mutator: mutator}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/library/overlay":
		h.overlay(w, r)
	case "/api/library/playback-window":
		h.playbackWindow(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) overlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ID      string   `json:"id"`
		IDs     []string `json:"ids"`
		Restore bool     `json:"restore"`
		Title   *string  `json:"title"`
		Artist  *string  `json:"artist"`
		Album   *string  `json:"album"`
		Genre   *string  `json:"genre"`
	}
	if !decode(w, r, &request) {
		return
	}

	targets := normalizeRefs(append(request.IDs, request.ID))
	if len(targets) == 0 {
		jsonError(w, http.StatusBadRequest, "track id required")
		return
	}
	if !request.Restore &&
		request.Title == nil &&
		request.Artist == nil &&
		request.Album == nil &&
		request.Genre == nil {
		jsonError(w, http.StatusBadRequest, "no display fields supplied")
		return
	}

	current := h.snapshot()
	for _, id := range targets {
		if current.Paths[id] == "" {
			jsonError(
				w,
				http.StatusNotFound,
				"one or more tracks are no longer available",
			)
			return
		}
	}

	updated, err := h.mutator.UpdateOverlays(
		targets,
		libraryservice.OverlayPatch{
			Title:  request.Title,
			Artist: request.Artist,
			Album:  request.Album,
			Genre:  request.Genre,
		},
		request.Restore,
	)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{
		"ok":      true,
		"updated": updated,
	})
}

func (h *Handler) playbackWindow(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ID      string   `json:"id"`
		Start   float64  `json:"start"`
		End     *float64 `json:"end"`
		Restore bool     `json:"restore"`
	}
	if !decode(w, r, &request) {
		return
	}
	request.ID = strings.TrimSpace(request.ID)
	if request.ID == "" {
		jsonError(w, http.StatusBadRequest, "track id required")
		return
	}

	current := h.snapshot()
	var track *library.Track
	for index := range current.Tracks {
		if current.Tracks[index].ID == request.ID {
			copy := current.Tracks[index]
			track = &copy
			break
		}
	}
	if track == nil || current.Paths[request.ID] == "" {
		jsonError(
			w,
			http.StatusNotFound,
			"track is no longer available",
		)
		return
	}

	if request.Restore {
		_, _, err := h.mutator.UpdatePlaybackWindow(
			request.ID,
			0,
			nil,
			true,
		)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonOut(w, http.StatusOK, map[string]any{
			"ok":       true,
			"restored": true,
		})
		return
	}

	start := request.Start
	if start < 0.001 {
		start = 0
	}
	duration := track.Duration
	if duration > 0 && start >= duration-0.02 {
		jsonError(
			w,
			http.StatusBadRequest,
			"playback start must be before the end of the track",
		)
		return
	}

	var end *float64
	if request.End != nil {
		value := *request.End
		if duration > 0 && value > duration {
			value = duration
		}
		if value <= start+0.02 {
			jsonError(
				w,
				http.StatusBadRequest,
				"playback end must be after playback start",
			)
			return
		}
		if duration <= 0 || value < duration-0.02 {
			end = &value
		}
	}

	window, present, err := h.mutator.UpdatePlaybackWindow(
		request.ID,
		start,
		end,
		false,
	)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var projected any = map[string]any{}
	if present {
		projected = window
	}
	jsonOut(w, http.StatusOK, map[string]any{
		"ok":     true,
		"window": projected,
	})
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

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid library edit request")
		return false
	}
	return true
}

func methodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	jsonError(w, http.StatusMethodNotAllowed, method+" required")
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

var _ = errors.Is
