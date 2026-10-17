// Package httpapi exposes temporary music-folder discovery.
//
// Discovery never mutates config. Include/ignore remains a separate explicit config action.
package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	desktopconfig "github.com/vgong24/VexstreamMusic/v2/desktop/config"
	"github.com/vgong24/VexstreamMusic/v2/desktop/config/discovery"
)

type ConfigSnapshot func() desktopconfig.State
type HomeFunc func() string

type Handler struct {
	config ConfigSnapshot
	home   HomeFunc
}

func New(config ConfigSnapshot, home HomeFunc) *Handler {
	if config == nil || home == nil {
		panic("source discovery http adapter requires config snapshot and home")
	}
	return &Handler{config: config, home: home}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/discover" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(
			w,
			http.StatusMethodNotAllowed,
			map[string]string{"error": "POST required"},
		)
		return
	}
	var request struct {
		Deep bool `json:"deep"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{"error": "invalid discovery request"},
		)
		return
	}

	state := h.config()
	home := strings.TrimSpace(h.home())
	result := discovery.Scan(
		discovery.DefaultRoots(home, request.Deep),
		state.Sources,
		state.Ignored,
		request.Deep,
	)
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
