package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	desktopconfig "github.com/vgong24/VexstreamMusic/v2/desktop/config"
)

func TestDiscoverUsesCurrentConfigOnlyAsExclusionProjection(t *testing.T) {
	home := t.TempDir()
	included := filepath.Join(home, "Music", "Included")
	candidate := filepath.Join(home, "Music", "Candidate")
	for _, path := range []string{
		filepath.Join(included, "one.mp3"),
		filepath.Join(candidate, "one.mp3"),
		filepath.Join(candidate, "two.mp3"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("mp3"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	handler := New(
		func() desktopconfig.State {
			return desktopconfig.State{Sources: []string{included}}
		},
		func() string { return home },
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/discover",
			strings.NewReader(`{"deep":false}`),
		),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("discovery status drift: %d %s", response.Code, response.Body.String())
	}
	var result struct {
		Candidates []struct {
			Path     string `json:"path"`
			MP3Count int    `json:"mp3Count"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("discovery response JSON drift: %v: %s", err, response.Body.String())
	}
	if len(result.Candidates) != 1 ||
		result.Candidates[0].Path != candidate ||
		result.Candidates[0].MP3Count != 2 {
		t.Fatalf("discovery projection drift: %s", response.Body.String())
	}
}

func TestDiscoverRejectsUnknownFieldsAndWrongMethods(t *testing.T) {
	handler := New(
		func() desktopconfig.State { return desktopconfig.State{} },
		func() string { return t.TempDir() },
	)

	unknown := httptest.NewRecorder()
	handler.ServeHTTP(
		unknown,
		httptest.NewRequest(
			http.MethodPost,
			"/api/discover",
			strings.NewReader(`{"deep":false,"includeEverything":true}`),
		),
	)
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown discovery field accepted: %d", unknown.Code)
	}

	wrong := httptest.NewRecorder()
	handler.ServeHTTP(
		wrong,
		httptest.NewRequest(http.MethodGet, "/api/discover", nil),
	)
	if wrong.Code != http.StatusMethodNotAllowed ||
		wrong.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("discovery method drift: %d %q", wrong.Code, wrong.Header().Get("Allow"))
	}
}
