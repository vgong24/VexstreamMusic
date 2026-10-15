package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
)

func testHandler(t *testing.T) (*Handler, *playlists.Service) {
	t.Helper()
	service, err := playlists.Open(filepath.Join(t.TempDir(), "playlists.json"))
	if err != nil {
		t.Fatal(err)
	}
	return New(service), service
}

func request(t *testing.T, handler *Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(method, path, strings.NewReader(body)),
	)
	return response
}

func playlistFromResponse(t *testing.T, response *httptest.ResponseRecorder) playlists.Playlist {
	t.Helper()
	var payload struct {
		Playlist playlists.Playlist `json:"playlist"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Playlist
}

func TestGetProjectsDurablePlaylistOrderIncludingMissingRefs(t *testing.T) {
	handler, service := testHandler(t)
	row, err := service.Create("Road Trip", []string{"track-a", "missing", "track-b"})
	if err != nil {
		t.Fatal(err)
	}
	response := request(t, handler, http.MethodGet, "/api/playlists", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	var payload struct {
		Playlists []playlists.Playlist `json:"playlists"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Playlists) != 1 ||
		payload.Playlists[0].ID != row.ID ||
		strings.Join(payload.Playlists[0].TrackRefs, ",") != "track-a,missing,track-b" {
		t.Fatalf("playlist projection drift: %+v", payload)
	}
}

func TestGetReturnsEmptyArrayInsteadOfNull(t *testing.T) {
	handler, _ := testHandler(t)
	response := request(t, handler, http.MethodGet, "/api/playlists", "")
	if body := response.Body.String(); !strings.Contains(body, `"playlists":[]`) {
		t.Fatalf("empty projection drift: %s", body)
	}
}

func TestCreateAddOrderRemoveRenameLifecycle(t *testing.T) {
	handler, _ := testHandler(t)
	created := request(
		t, handler, http.MethodPost, "/api/playlists/create",
		`{"name":" Road Trip ","trackIds":["a","a","missing","b"]}`,
	)
	if created.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	row := playlistFromResponse(t, created)
	if row.Name != "Road Trip" || strings.Join(row.TrackRefs, ",") != "a,missing,b" {
		t.Fatalf("create drift: %+v", row)
	}

	added := request(
		t, handler, http.MethodPost, "/api/playlists/add",
		`{"id":"`+row.ID+`","trackIds":["c","a"]}`,
	)
	if added.Code != http.StatusOK {
		t.Fatalf("add status=%d body=%s", added.Code, added.Body.String())
	}
	var addPayload struct {
		Added    int                `json:"added"`
		Playlist playlists.Playlist `json:"playlist"`
	}
	if err := json.Unmarshal(added.Body.Bytes(), &addPayload); err != nil {
		t.Fatal(err)
	}
	if addPayload.Added != 1 ||
		strings.Join(addPayload.Playlist.TrackRefs, ",") != "a,missing,b,c" {
		t.Fatalf("add drift: %+v", addPayload)
	}

	conflict := request(
		t, handler, http.MethodPost, "/api/playlists/order",
		`{"id":"`+row.ID+`","trackIds":["a","b"]}`,
	)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("order mismatch status=%d body=%s", conflict.Code, conflict.Body.String())
	}

	ordered := request(
		t, handler, http.MethodPost, "/api/playlists/order",
		`{"id":"`+row.ID+`","trackIds":["missing","c","b","a"]}`,
	)
	if ordered.Code != http.StatusOK {
		t.Fatalf("order status=%d body=%s", ordered.Code, ordered.Body.String())
	}

	removed := request(
		t, handler, http.MethodPost, "/api/playlists/remove",
		`{"id":"`+row.ID+`","trackIds":["missing","c"]}`,
	)
	if removed.Code != http.StatusOK {
		t.Fatalf("remove status=%d body=%s", removed.Code, removed.Body.String())
	}
	if got := strings.Join(playlistFromResponse(t, removed).TrackRefs, ","); got != "b,a" {
		t.Fatalf("remove drift: %s", got)
	}

	renamed := request(
		t, handler, http.MethodPost, "/api/playlists/rename",
		`{"id":"`+row.ID+`","name":" Final Mix "}`,
	)
	if renamed.Code != http.StatusOK ||
		playlistFromResponse(t, renamed).Name != "Final Mix" {
		t.Fatalf("rename drift: %d %s", renamed.Code, renamed.Body.String())
	}
}

func TestDeleteRequiresExplicitPlaylistOnlyConfirmation(t *testing.T) {
	handler, service := testHandler(t)
	row, err := service.Create("Mix", []string{"a", "missing"})
	if err != nil {
		t.Fatal(err)
	}

	held := request(
		t, handler, http.MethodPost, "/api/playlists/delete",
		`{"id":"`+row.ID+`"}`,
	)
	if held.Code != http.StatusBadRequest || len(service.Snapshot()) != 1 {
		t.Fatalf("delete confirmation hold drift: %d %+v", held.Code, service.Snapshot())
	}

	deleted := request(
		t, handler, http.MethodPost, "/api/playlists/delete",
		`{"id":"`+row.ID+`","confirm":"DELETE_PLAYLIST_ONLY"}`,
	)
	if deleted.Code != http.StatusOK || len(service.Snapshot()) != 0 {
		t.Fatalf("delete drift: %d %+v", deleted.Code, service.Snapshot())
	}
}

func TestUnknownPlaylistAndInvalidRequestsFailClosed(t *testing.T) {
	handler, _ := testHandler(t)

	missing := request(
		t, handler, http.MethodPost, "/api/playlists/add",
		`{"id":"missing","trackIds":["a"]}`,
	)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing playlist status=%d", missing.Code)
	}

	badName := request(
		t, handler, http.MethodPost, "/api/playlists/create",
		`{"name":" ","trackIds":[]}`,
	)
	if badName.Code != http.StatusBadRequest {
		t.Fatalf("blank name status=%d", badName.Code)
	}

	badJSON := request(
		t, handler, http.MethodPost, "/api/playlists/create", "{bad",
	)
	if badJSON.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON status=%d", badJSON.Code)
	}
}

func TestRouteMethodsAreExplicit(t *testing.T) {
	handler, _ := testHandler(t)
	for _, item := range []struct {
		path    string
		method  string
		allowed string
	}{
		{"/api/playlists", http.MethodPost, http.MethodGet},
		{"/api/playlists/create", http.MethodGet, http.MethodPost},
		{"/api/playlists/add", http.MethodGet, http.MethodPost},
		{"/api/playlists/order", http.MethodGet, http.MethodPost},
		{"/api/playlists/remove", http.MethodGet, http.MethodPost},
		{"/api/playlists/rename", http.MethodGet, http.MethodPost},
		{"/api/playlists/delete", http.MethodGet, http.MethodPost},
	} {
		response := request(t, handler, item.method, item.path, "")
		if response.Code != http.StatusMethodNotAllowed ||
			response.Header().Get("Allow") != item.allowed {
			t.Fatalf(
				"%s method drift: status=%d allow=%q",
				item.path, response.Code, response.Header().Get("Allow"),
			)
		}
	}
}
