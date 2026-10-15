package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtimeenv "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube/runtime"
)

func TestStatusDoesNotTreatSearchReadyAsImportReady(t *testing.T) {
	handler := NewWith(
		"/runtime",
		func(path string) runtimeenv.Status {
			if path != "/runtime" {
				t.Fatalf("runtime path drift: %q", path)
			}
			return runtimeenv.Status{
				SearchReady: true,
				FFmpegReady: false,
				ImportReady: false,
				Python: "/python",
			}
		},
		func(context.Context, string) (string, error) {
			return "", nil
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/import/status", nil),
	)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"searchReady":true`) ||
		!strings.Contains(response.Body.String(), `"importReady":false`) {
		t.Fatalf("runtime status drift: %d %s", response.Code, response.Body.String())
	}
}

func TestSetupIsExplicitThenReturnsFreshStatus(t *testing.T) {
	setupCalls := 0
	resolveCalls := 0
	handler := NewWith(
		"/runtime",
		func(string) runtimeenv.Status {
			resolveCalls++
			return runtimeenv.Status{
				SearchReady: true,
				FFmpegReady: true,
				ImportReady: true,
			}
		},
		func(_ context.Context, path string) (string, error) {
			setupCalls++
			if path != "/runtime" {
				t.Fatalf("setup runtime path drift: %q", path)
			}
			return "ready", nil
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPost, "/api/import/setup", strings.NewReader(`{}`)),
	)
	if response.Code != http.StatusOK ||
		setupCalls != 1 ||
		resolveCalls != 1 ||
		!strings.Contains(response.Body.String(), `"message":"ready"`) ||
		!strings.Contains(response.Body.String(), `"importReady":true`) {
		t.Fatalf(
			"setup/status drift: code=%d setup=%d resolve=%d body=%s",
			response.Code,
			setupCalls,
			resolveCalls,
			response.Body.String(),
		)
	}
}

func TestSetupFailureDoesNotManufactureReadyStatus(t *testing.T) {
	resolveCalls := 0
	handler := NewWith(
		"/runtime",
		func(string) runtimeenv.Status {
			resolveCalls++
			return runtimeenv.Status{ImportReady: true}
		},
		func(context.Context, string) (string, error) {
			return "", errors.New("setup failed")
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPost, "/api/import/setup", nil),
	)
	if response.Code != http.StatusInternalServerError ||
		resolveCalls != 0 ||
		!strings.Contains(response.Body.String(), "setup failed") {
		t.Fatalf("setup failure drift: %d calls=%d body=%s", response.Code, resolveCalls, response.Body.String())
	}
}

func TestRuntimeRoutesRejectWrongMethods(t *testing.T) {
	handler := NewWith(
		"/runtime",
		func(string) runtimeenv.Status { return runtimeenv.Status{} },
		func(context.Context, string) (string, error) { return "", nil },
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/import/setup", nil),
	)
	if response.Code != http.StatusMethodNotAllowed ||
		response.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("setup method drift: %d %q", response.Code, response.Header().Get("Allow"))
	}
}
