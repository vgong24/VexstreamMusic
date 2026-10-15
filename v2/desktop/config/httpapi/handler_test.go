package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	desktopconfig "github.com/vgong24/VexstreamMusic/v2/desktop/config"
)

type configFixture struct {
	state desktopconfig.State
	addErr error
	removeErr error
	restored bool
}

func (f *configFixture) Snapshot() desktopconfig.State {
	return f.state
}
func (f *configFixture) AddSource(path string) (desktopconfig.State, error) {
	if f.addErr != nil {
		return desktopconfig.State{}, f.addErr
	}
	f.state.Sources = []string{strings.TrimSpace(path)}
	return f.state, nil
}
func (f *configFixture) RemoveSource(path string) (desktopconfig.State, error) {
	if f.removeErr != nil {
		return desktopconfig.State{}, f.removeErr
	}
	f.state.Sources = []string{}
	return f.state, nil
}
func (f *configFixture) Ignore(path string) (desktopconfig.State, error) {
	f.state.Ignored = []string{strings.TrimSpace(path)}
	return f.state, nil
}
func (f *configFixture) Unignore(path string) (desktopconfig.State, error) {
	f.state.Ignored = []string{}
	return f.state, nil
}
func (f *configFixture) ClearIgnored() (desktopconfig.State, error) {
	f.state.Ignored = []string{}
	return f.state, nil
}
func (f *configFixture) RestoreBackup() (desktopconfig.State, error) {
	f.restored = true
	f.state = desktopconfig.State{Sources: []string{"/backup"}}
	return f.state, nil
}

func TestAddPersistsConfigThenReportsRefreshFailureSeparately(t *testing.T) {
	config := &configFixture{}
	var refreshed []string
	handler := New(config, func(sources []string) error {
		refreshed = append([]string(nil), sources...)
		return errors.New("scan unavailable")
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/source/add",
			strings.NewReader(`{"path":" /music "}`),
		),
	)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"ok":true`) ||
		!strings.Contains(response.Body.String(), `"libraryRefresh":"FAILED"`) ||
		!strings.Contains(response.Body.String(), "scan unavailable") {
		t.Fatalf("config/refresh truth collapsed: %d %s", response.Code, response.Body.String())
	}
	if !reflect.DeepEqual(refreshed, []string{"/music"}) ||
		!reflect.DeepEqual(config.state.Sources, []string{"/music"}) {
		t.Fatalf("config effect lost after refresh failure: config=%+v refresh=%v", config.state, refreshed)
	}
}

func TestRemoveTriggersRefreshWithPersistedRemainingSources(t *testing.T) {
	config := &configFixture{state: desktopconfig.State{Sources: []string{"/music"}}}
	var refreshed []string
	handler := New(config, func(sources []string) error {
		refreshed = append([]string(nil), sources...)
		return nil
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/source/remove",
			strings.NewReader(`{"path":"/music"}`),
		),
	)
	if response.Code != http.StatusOK ||
		!reflect.DeepEqual(refreshed, []string{}) {
		t.Fatalf("remove refresh drift: %d %v %s", response.Code, refreshed, response.Body.String())
	}
}

func TestIgnoreDoesNotInventLibraryReconcile(t *testing.T) {
	config := &configFixture{}
	refreshCalls := 0
	handler := New(config, func([]string) error {
		refreshCalls++
		return nil
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/candidate/ignore",
			strings.NewReader(`{"path":"/candidate"}`),
		),
	)
	if response.Code != http.StatusOK ||
		refreshCalls != 0 ||
		!reflect.DeepEqual(config.state.Ignored, []string{"/candidate"}) {
		t.Fatalf("ignore ownership drift: %d calls=%d state=%+v", response.Code, refreshCalls, config.state)
	}
}

func TestDestructiveConfigurationActionsRequireExplicitConfirmation(t *testing.T) {
	config := &configFixture{state: desktopconfig.State{Ignored: []string{"/skip"}}}
	handler := New(config, nil)

	clear := httptest.NewRecorder()
	handler.ServeHTTP(
		clear,
		httptest.NewRequest(
			http.MethodPost,
			"/api/ignored/clear",
			strings.NewReader(`{"confirm":"no"}`),
		),
	)
	if clear.Code != http.StatusBadRequest ||
		len(config.state.Ignored) != 1 {
		t.Fatalf("ignored clear admitted without confirmation: %d %+v", clear.Code, config.state)
	}

	restore := httptest.NewRecorder()
	handler.ServeHTTP(
		restore,
		httptest.NewRequest(
			http.MethodPost,
			"/api/source/config/restore-backup",
			strings.NewReader(`{"confirm":"no"}`),
		),
	)
	if restore.Code != http.StatusBadRequest || config.restored {
		t.Fatalf("backup restore admitted without confirmation: %d restored=%v", restore.Code, config.restored)
	}
}

func TestSnapshotAndWrongMethodsStayBounded(t *testing.T) {
	config := &configFixture{
		state: desktopconfig.State{
			Sources: []string{"/music"},
			Ignored: []string{"/skip"},
		},
	}
	handler := New(config, nil)

	get := httptest.NewRecorder()
	handler.ServeHTTP(
		get,
		httptest.NewRequest(http.MethodGet, "/api/source/config", nil),
	)
	if get.Code != http.StatusOK ||
		!strings.Contains(get.Body.String(), "/music") ||
		!strings.Contains(get.Body.String(), "/skip") {
		t.Fatalf("snapshot drift: %d %s", get.Code, get.Body.String())
	}

	wrong := httptest.NewRecorder()
	handler.ServeHTTP(
		wrong,
		httptest.NewRequest(http.MethodGet, "/api/source/add", nil),
	)
	if wrong.Code != http.StatusMethodNotAllowed ||
		wrong.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("source add method drift: %d %q", wrong.Code, wrong.Header().Get("Allow"))
	}
}
