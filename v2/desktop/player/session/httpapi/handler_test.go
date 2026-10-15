package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	playersession "github.com/vgong24/VexstreamMusic/v2/desktop/player/session"
)

type storeFixture struct {
	load      playersession.Snapshot
	found     bool
	loadErr   error
	save      playersession.Snapshot
	saveErr   error
	saved     playersession.Snapshot
}

func (s *storeFixture) Load() (playersession.Snapshot, bool, error) {
	return s.load, s.found, s.loadErr
}

func (s *storeFixture) Save(
	input playersession.Snapshot,
) (playersession.Snapshot, error) {
	s.saved = input
	if s.saveErr != nil {
		return playersession.Snapshot{}, s.saveErr
	}
	if s.save.SchemaVersion == "" {
		s.save = input
		s.save.SchemaVersion = playersession.SchemaVersion
	}
	return s.save, nil
}

func TestGetReturnsAbsentEmptySessionWithoutInventingCurrentTrack(t *testing.T) {
	response := httptest.NewRecorder()
	New(&storeFixture{}).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/player/session", nil),
	)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"found":false`) ||
		!strings.Contains(response.Body.String(), `"currentIndex":-1`) {
		t.Fatalf("missing session projection drift: %d %s", response.Code, response.Body.String())
	}
}

func TestPutMapsExactQueueWitnessIntoStore(t *testing.T) {
	store := &storeFixture{}
	response := httptest.NewRecorder()
	New(store).ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPut,
			"/api/player/session",
			strings.NewReader(`{
			  "items":["a","b","a"],
			  "currentIndex":2,
			  "currentRef":"a",
			  "shuffle":true,
			  "repeat":false
			}`),
		),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if len(store.saved.Items) != 3 ||
		store.saved.CurrentIndex != 2 ||
		store.saved.CurrentRef != "a" ||
		!store.saved.Shuffle ||
		store.saved.Repeat {
		t.Fatalf("save mapping drift: %+v", store.saved)
	}
}

func TestPutRejectsUnknownFieldsBeforeStore(t *testing.T) {
	store := &storeFixture{}
	response := httptest.NewRecorder()
	New(store).ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPut,
			"/api/player/session",
			strings.NewReader(`{
			  "items":[],
			  "currentIndex":-1,
			  "currentRef":"",
			  "shuffle":false,
			  "repeat":false,
			  "autoplay":true
			}`),
		),
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown field accepted: %d", response.Code)
	}
	if store.saved.Items != nil {
		t.Fatalf("store saw rejected request: %+v", store.saved)
	}
}

func TestFutureSchemaHoldMapsToConflictAndDoesNotPretendMissing(t *testing.T) {
	store := &storeFixture{
		found: true,
		loadErr: playersession.ErrUnsupportedSchema,
	}
	response := httptest.NewRecorder()
	New(store).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/player/session", nil),
	)
	if response.Code != http.StatusConflict {
		t.Fatalf("future schema load status=%d body=%s", response.Code, response.Body.String())
	}

	store = &storeFixture{
		saveErr: playersession.ErrUnsupportedSchema,
	}
	response = httptest.NewRecorder()
	New(store).ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPut,
			"/api/player/session",
			strings.NewReader(`{
			  "items":[],
			  "currentIndex":-1,
			  "currentRef":"",
			  "shuffle":false,
			  "repeat":false
			}`),
		),
	)
	if response.Code != http.StatusConflict {
		t.Fatalf("future schema save status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestWrongMethodFailsClosed(t *testing.T) {
	response := httptest.NewRecorder()
	New(&storeFixture{}).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPost, "/api/player/session", nil),
	)
	if response.Code != http.StatusMethodNotAllowed ||
		response.Header().Get("Allow") != "GET, PUT" {
		t.Fatalf("method drift: %d %q", response.Code, response.Header().Get("Allow"))
	}
}

func TestUnexpectedLoadFailureStaysServerFailure(t *testing.T) {
	response := httptest.NewRecorder()
	New(&storeFixture{
		loadErr: errors.New("disk unavailable"),
	}).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/player/session", nil),
	)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected load failure status=%d", response.Code)
	}
}
