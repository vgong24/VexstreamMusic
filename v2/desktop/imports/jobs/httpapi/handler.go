// Package httpapi exposes the local V4 Import job manager.
//
// It preserves the familiar V3 route family while keeping uncertain crash recovery explicit.
// No endpoint may silently replay an interrupted commit.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	importmodel "github.com/vgong24/VexstreamMusic/v2/desktop/imports"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/jobs"
	importprovider "github.com/vgong24/VexstreamMusic/v2/desktop/imports/provider"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/workflow"
)

type Manager interface {
	Snapshot() jobs.Snapshot
	Start(workflow.Request) (importmodel.Job, error)
	Cancel(string) error
	RetryCommit(string) (importmodel.Job, error)
	RecoverCommit(string) (importmodel.Job, error)
	Discard(string) (importmodel.Job, error)
}

type DestinationSnapshot func() []string

type Handler struct {
	manager      Manager
	destinations DestinationSnapshot
}

func New(manager Manager) *Handler {
	return NewWithDestinations(manager, nil)
}

func NewWithDestinations(
	manager Manager,
	destinations DestinationSnapshot,
) *Handler {
	if manager == nil {
		panic("Import jobs HTTP adapter requires manager")
	}
	return &Handler{
		manager:      manager,
		destinations: destinations,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/import/jobs":
		h.jobs(w, r)
	case "/api/import/destinations":
		h.destinationsList(w, r)
	case "/api/import/start":
		h.start(w, r)
	case "/api/import/cancel":
		h.cancel(w, r)
	case "/api/import/commit":
		h.commit(w, r)
	case "/api/import/discard":
		h.discard(w, r)
	case "/api/import/recover-commit":
		h.recoverCommit(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) destinationsList(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	rows := []string{}
	if h.destinations != nil {
		seen := map[string]bool{}
		for _, value := range h.destinations() {
			value = strings.TrimSpace(value)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			rows = append(rows, value)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"destinations": rows,
	})
}

func (h *Handler) jobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, h.manager.Snapshot())
}

type startRequest struct {
	Title          string           `json:"sourceTitle"`
	Destination    string           `json:"destination"`
	URL            string           `json:"url"`
	Mode           string           `json:"mode,omitempty"`
	WholeTitle     string           `json:"wholeTitle,omitempty"`
	ArtistOverride string           `json:"artistOverride,omitempty"`
	Collection     string           `json:"collection,omitempty"`
	Genre          string           `json:"genre,omitempty"`
	Sections       []sectionRequest `json:"sections,omitempty"`
}

type sectionRequest struct {
	Index       int     `json:"index"`
	SourceTitle string  `json:"sourceTitle,omitempty"`
	Title       string  `json:"title,omitempty"`
	Artist      string  `json:"artist,omitempty"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Selected    *bool   `json:"selected,omitempty"`
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var input startRequest
	if err := decodeBounded(w, r, &input, 128*1024); err != nil {
		http.Error(w, "invalid import start request", http.StatusBadRequest)
		return
	}

	sections := make([]importprovider.PrepareSection, 0, len(input.Sections))
	for _, section := range input.Sections {
		selected := true
		if section.Selected != nil {
			selected = *section.Selected
		}
		sections = append(sections, importprovider.PrepareSection{
			Index:       section.Index,
			SourceTitle: section.SourceTitle,
			Title:       section.Title,
			Artist:      section.Artist,
			Start:       section.Start,
			End:         section.End,
			Selected:    selected,
		})
	}

	job, err := h.manager.Start(workflow.Request{
		Title:          strings.TrimSpace(input.Title),
		Destination:    strings.TrimSpace(input.Destination),
		URL:            strings.TrimSpace(input.URL),
		Mode:           strings.TrimSpace(input.Mode),
		WholeTitle:     strings.TrimSpace(input.WholeTitle),
		ArtistOverride: strings.TrimSpace(input.ArtistOverride),
		Collection:     strings.TrimSpace(input.Collection),
		Genre:          strings.TrimSpace(input.Genre),
		Sections:       sections,
	})
	if err != nil {
		status := http.StatusBadRequest
		if job.ID != "" {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"jobId":  job.ID,
		"status": job.Status,
	})
}

type jobRequest struct {
	ID string `json:"id"`
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	id, ok := h.jobID(w, r)
	if !ok {
		return
	}
	if err := h.manager.Cancel(id); err != nil {
		writeJobError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"status": "cancelling",
	})
}

func (h *Handler) commit(w http.ResponseWriter, r *http.Request) {
	id, ok := h.jobID(w, r)
	if !ok {
		return
	}
	job, err := h.manager.RetryCommit(id)
	if err != nil {
		writeJobError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) recoverCommit(w http.ResponseWriter, r *http.Request) {
	id, ok := h.jobID(w, r)
	if !ok {
		return
	}
	job, err := h.manager.RecoverCommit(id)
	if err != nil {
		writeJobError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) discard(w http.ResponseWriter, r *http.Request) {
	id, ok := h.jobID(w, r)
	if !ok {
		return
	}
	job, err := h.manager.Discard(id)
	if err != nil {
		writeJobError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) jobID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return "", false
	}
	var input jobRequest
	if err := decodeBounded(w, r, &input, 16*1024); err != nil {
		http.Error(w, "invalid import job request", http.StatusBadRequest)
		return "", false
	}
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		http.Error(w, "job id required", http.StatusBadRequest)
		return "", false
	}
	return input.ID, true
}

func writeJobError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, jobs.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, jobs.ErrNotCancellable),
		errors.Is(err, jobs.ErrNotDiscardable),
		errors.Is(err, jobs.ErrNotRetryable),
		errors.Is(err, jobs.ErrRecoveryNotReady):
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decodeBounded(
	w http.ResponseWriter,
	r *http.Request,
	target any,
	limit int64,
) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func methodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	http.Error(w, method+" required", http.StatusMethodNotAllowed)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
