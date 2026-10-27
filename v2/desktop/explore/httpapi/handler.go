package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/explore"
)

type Explorer interface {
	Explore(context.Context, explore.Seed) (explore.Frame, error)
}

type SeedResolver interface {
	ResolveTrackSeed(trackRef string) (explore.Seed, bool)
}

type Handler struct {
	explorer Explorer
	resolver SeedResolver
}

func New(explorer Explorer, resolver SeedResolver) *Handler {
	if explorer == nil {
		panic("explore http adapter requires explorer")
	}
	return &Handler{explorer: explorer, resolver: resolver}
}

type request struct {
	TrackRef string        `json:"trackId"`
	Seed     *explore.Seed `json:"seed"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/discovery/radio" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var input request
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid Explore request", http.StatusBadRequest)
		return
	}

	seed, status, err := h.resolveSeed(input)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	frame, err := h.explorer.Explore(r.Context(), seed)
	if err != nil {
		status := http.StatusBadGateway
		if strings.Contains(err.Error(), "seed title") ||
			strings.Contains(err.Error(), "query formation") {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(frame)
}

func (h *Handler) resolveSeed(input request) (explore.Seed, int, error) {
	ref := strings.TrimSpace(input.TrackRef)
	if ref != "" {
		if h.resolver == nil {
			return explore.Seed{}, http.StatusNotFound, errors.New("Explore seed track is unavailable")
		}
		seed, ok := h.resolver.ResolveTrackSeed(ref)
		if !ok {
			return explore.Seed{}, http.StatusNotFound, errors.New("Explore seed track is no longer in this library")
		}
		seed.Kind = "library"
		seed.TrackRef = ref
		return seed, 0, nil
	}
	if input.Seed == nil {
		return explore.Seed{}, http.StatusBadRequest, errors.New("Explore requires a local track or provider candidate seed")
	}
	seed := *input.Seed
	seed.Kind = "provider"
	seed.TrackRef = ""
	return seed, 0, nil
}
