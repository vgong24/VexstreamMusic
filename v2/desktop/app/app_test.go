package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/appdata"
)

func TestOpenFormsLocalDesktopCompositionWithRecoveredPhoneAdministration(t *testing.T) {
	root := t.TempDir()
	application, err := Open(Options{
		Version:  "4.0.0",
		BuildRef: "test-build",
		Layout:   appdata.At(filepath.Join(root, "VexStreamMusic")),
		Home:     root,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	server := httptest.NewServer(application.Handler())
	defer server.Close()

	response, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var health map[string]any
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK ||
		health["version"] != "4.0.0" ||
		health["buildRef"] != "test-build" {
		t.Fatalf("health identity drift: code=%d body=%+v", response.StatusCode, health)
	}

	for _, path := range []string{
		"/api/library",
		"/api/playlists",
		"/api/source/config",
		"/api/library/trash",
		"/api/import/jobs",
		"/api/import/destinations",
		"/api/import/status",
		"/api/player/session",
	} {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			t.Fatalf("local product route %s status=%d", path, response.StatusCode)
		}
		response.Body.Close()
	}

	response, err = http.Get(server.URL + "/api/mobile/pairing")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		t.Fatalf("recovered phone administration status=%d", response.StatusCode)
	}
	response.Body.Close()

	for _, path := range []string{
		"/v1/pairing/request",
		"/v1/catalog",
		"/v1/media/track-ref",
	} {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusNotFound {
			response.Body.Close()
			t.Fatalf("sanitized LAN contract leaked into localhost admin mux: %s -> %d", path, response.StatusCode)
		}
		response.Body.Close()
	}
}

func TestEmbeddedBrowserLoadsFromSameLoopbackHandler(t *testing.T) {
	root := t.TempDir()
	application, err := Open(Options{
		Version:  "4.0.0",
		BuildRef: "test-build",
		Layout:   appdata.At(filepath.Join(root, "VexStreamMusic")),
		Home:     root,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), "Desktop V4") {
		t.Fatalf("embedded browser entry drift: %d %s", response.Code, response.Body.String())
	}
}

func TestExitRequiresExplicitConfirmationThenSignalsShutdown(t *testing.T) {
	root := t.TempDir()
	application, err := Open(Options{
		Version:  "4.0.0",
		BuildRef: "test-build",
		Layout:   appdata.At(filepath.Join(root, "VexStreamMusic")),
		Home:     root,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	bad := httptest.NewRecorder()
	application.Handler().ServeHTTP(
		bad,
		httptest.NewRequest(
			http.MethodPost,
			"/api/system/exit",
			strings.NewReader(`{"confirm":"no"}`),
		),
	)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("exit admitted without confirmation: %d", bad.Code)
	}

	ok := httptest.NewRecorder()
	application.Handler().ServeHTTP(
		ok,
		httptest.NewRequest(
			http.MethodPost,
			"/api/system/exit",
			strings.NewReader(`{"confirm":"EXIT_VEXSTREAM"}`),
		),
	)
	if ok.Code != http.StatusOK {
		t.Fatalf("exit response drift: %d %s", ok.Code, ok.Body.String())
	}
	select {
	case <-application.ExitRequested():
	default:
		t.Fatal("confirmed Exit did not signal shutdown")
	}
}

func TestPlayerSessionUsesV4OnlyLayoutThroughComposition(t *testing.T) {
	root := t.TempDir()
	layout := appdata.At(filepath.Join(root, "VexStreamMusic"))
	application, err := Open(Options{
		Version:  "4.0.0",
		BuildRef: "test-build",
		Layout:   layout,
		Home:     root,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	save := httptest.NewRecorder()
	application.Handler().ServeHTTP(
		save,
		httptest.NewRequest(
			http.MethodPut,
			"/api/player/session",
			strings.NewReader(`{
			  "items":["track-a"],
			  "currentIndex":0,
			  "currentRef":"track-a",
			  "shuffle":true,
			  "repeat":false
			}`),
		),
	)
	if save.Code != http.StatusOK {
		t.Fatalf("session save failed: %d %s", save.Code, save.Body.String())
	}
	if _, err := os.Stat(layout.PlayerSession); err != nil {
		t.Fatalf("V4 session was not stored at V4 path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(layout.Base, "player-session.json")); !os.IsNotExist(err) {
		t.Fatalf("player session leaked into protected V3 root: %v", err)
	}
}
