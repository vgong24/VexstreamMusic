// Package httpapi exposes provider-runtime readiness and explicit one-time setup.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	runtimeenv "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube/runtime"
)

type ResolveFunc func(string) runtimeenv.Status
type SetupFunc func(context.Context, string) (string, error)

type Handler struct {
	sharedRuntime string
	resolve       ResolveFunc
	setup         SetupFunc
}

func New(sharedRuntime string) *Handler {
	return NewWith(
		sharedRuntime,
		runtimeenv.Resolve,
		runtimeenv.Setup,
	)
}

func NewWith(
	sharedRuntime string,
	resolve ResolveFunc,
	setup SetupFunc,
) *Handler {
	if resolve == nil || setup == nil {
		panic("provider runtime http adapter requires resolve and setup")
	}
	return &Handler{
		sharedRuntime: strings.TrimSpace(sharedRuntime),
		resolve:       resolve,
		setup:         setup,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/import/status":
		h.status(w, r)
	case "/api/import/setup":
		h.runSetup(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, h.resolve(h.sharedRuntime))
}

func (h *Handler) runSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	message, err := h.setup(r.Context(), h.sharedRuntime)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Import tool setup failed: " + err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": message,
		"status":  h.resolve(h.sharedRuntime),
	})
}

func methodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
		"error": method + " required",
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
