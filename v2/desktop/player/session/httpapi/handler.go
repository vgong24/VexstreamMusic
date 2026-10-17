package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	playersession "github.com/vgong24/VexstreamMusic/v2/desktop/player/session"
)

type Store interface {
	Load() (playersession.Snapshot, bool, error)
	Save(playersession.Snapshot) (playersession.Snapshot, error)
}

type Handler struct {
	store Store
}

func New(store Store) *Handler {
	if store == nil {
		panic("player session HTTP adapter requires store")
	}
	return &Handler{store: store}
}

type saveRequest struct {
	Items        []string `json:"items"`
	CurrentIndex int      `json:"currentIndex"`
	CurrentRef   string   `json:"currentRef"`
	Shuffle      bool     `json:"shuffle"`
	Repeat       bool     `json:"repeat"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/player/session" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.load(w)
	case http.MethodPut:
		h.save(w, r)
	default:
		w.Header().Set("Allow", "GET, PUT")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "GET or PUT required",
		})
	}
}

func (h *Handler) load(w http.ResponseWriter) {
	snapshot, found, err := h.store.Load()
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, playersession.ErrUnsupportedSchema) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	if !found {
		snapshot = playersession.Empty()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"found":   found,
		"session": snapshot,
	})
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	var input saveRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid player session request",
		})
		return
	}

	snapshot, err := h.store.Save(playersession.Snapshot{
		Items:        append([]string(nil), input.Items...),
		CurrentIndex: input.CurrentIndex,
		CurrentRef:   input.CurrentRef,
		Shuffle:      input.Shuffle,
		Repeat:       input.Repeat,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, playersession.ErrUnsupportedSchema) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"saved":   true,
		"session": snapshot,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
