package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/appdata"
)

func TestOpenFormsLocalDesktopCompositionWithoutMobileIngress(t *testing.T) {
	root := t.TempDir()
	application, err := Open(Options{
		Version: "4.0.0",
		BuildRef: "test-build",
		Layout: appdata.At(filepath.Join(root, "VexStreamMusic")),
		Home: root,
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

	for _, path := range []string{
		"/v1/pairing/request",
		"/api/mobile/pairing",
		"/mobile/catalog",
	} {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusNotFound {
			response.Body.Close()
			t.Fatalf("mobile ingress leaked into Desktop-first composition: %s -> %d", path, response.StatusCode)
		}
		response.Body.Close()
	}
}

func TestEmbeddedBrowserLoadsFromSameLoopbackHandler(t *testing.T) {
	root := t.TempDir()
	application, err := Open(Options{
		Version: "4.0.0",
		BuildRef: "test-build",
		Layout: appdata.At(filepath.Join(root, "VexStreamMusic")),
		Home: root,
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
		Version: "4.0.0",
		BuildRef: "test-build",
		Layout: appdata.At(filepath.Join(root, "VexStreamMusic")),
		Home: root,
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
