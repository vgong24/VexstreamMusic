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
	"github.com/vgong24/VexstreamMusic/v2/desktop/lifecycle"
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
		health["buildRef"] != "test-build" ||
		health["pid"] != float64(os.Getpid()) {
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

func TestOldSessionRoutesExposePlanAndHoldUnsafeRetirement(t *testing.T) {
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

	application.sessionPlan = func() lifecycle.SessionPlan {
		return lifecycle.SessionPlan{
			CurrentPID: os.Getpid(),
			Sessions: []lifecycle.LiveSession{
				{
					PID:             os.Getpid(),
					Version:         "4.0.0",
					Current:         true,
					TrackCount:      2,
					LibraryVerified: true,
					ImportsVerified: true,
				},
				{
					PID:             424242,
					Version:         "3.0.0",
					TrackCount:      1,
					ActiveImports:   1,
					LibraryVerified: true,
					ImportsVerified: true,
				},
			},
			Held: []lifecycle.RetireCandidate{
				{
					PID:           424242,
					Version:       "3.0.0",
					ActiveImports: 1,
					Reason:        "1 active import(s) are still owned by this session",
				},
			},
			Eligible: []lifecycle.RetireCandidate{},
			Warning:  "safe retirement only",
		}
	}

	plan := httptest.NewRecorder()
	application.Handler().ServeHTTP(
		plan,
		httptest.NewRequest(
			http.MethodGet,
			"/api/system/sessions/retire-plan",
			nil,
		),
	)
	if plan.Code != http.StatusOK {
		t.Fatalf("old-session plan status=%d body=%s", plan.Code, plan.Body.String())
	}
	var payload lifecycle.SessionPlan
	if err := json.NewDecoder(plan.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Held) != 1 ||
		payload.Held[0].PID != 424242 ||
		payload.Held[0].ActiveImports != 1 {
		t.Fatalf("old-session held projection drift: %+v", payload)
	}

	bad := httptest.NewRecorder()
	application.Handler().ServeHTTP(
		bad,
		httptest.NewRequest(
			http.MethodPost,
			"/api/system/sessions/retire",
			strings.NewReader(`{"confirm":"NO","pids":[424242]}`),
		),
	)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("old-session retirement admitted without confirmation: %d", bad.Code)
	}

	held := httptest.NewRecorder()
	application.Handler().ServeHTTP(
		held,
		httptest.NewRequest(
			http.MethodPost,
			"/api/system/sessions/retire",
			strings.NewReader(`{"confirm":"RETIRE_OLD_IDLE_SESSIONS","pids":[424242]}`),
		),
	)
	if held.Code != http.StatusOK {
		t.Fatalf("held retirement response=%d body=%s", held.Code, held.Body.String())
	}
	var result struct {
		Retired []int                     `json:"retired"`
		Failed  []lifecycle.RetireFailure `json:"failed"`
	}
	if err := json.NewDecoder(held.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Retired) != 0 ||
		len(result.Failed) != 1 ||
		result.Failed[0].PID != 424242 {
		t.Fatalf("held session was not kept safe: %+v", result)
	}
}
