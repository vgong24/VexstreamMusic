package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
)

func TestGetProjectsDurablePlaylistOrderIncludingMissingRefs(t *testing.T) {
	handler := New(func() []playlists.Playlist {
		return []playlists.Playlist{{
			ID: "road-trip", Name: "Road Trip",
			TrackRefs: []string{"track-a", "missing", "track-b"},
			CreatedAt: "c", UpdatedAt: "u",
		}}
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/playlists", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	var payload struct {
		Playlists []playlists.Playlist `json:"playlists"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Playlists) != 1 || strings.Join(payload.Playlists[0].TrackRefs, ",") != "track-a,missing,track-b" {
		t.Fatalf("playlist projection drift: %+v", payload)
	}
}

func TestGetReturnsEmptyArrayInsteadOfNull(t *testing.T) {
	handler := New(func() []playlists.Playlist { return nil })
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/playlists", nil))
	if body := response.Body.String(); !strings.Contains(body, `"playlists":[]`) {
		t.Fatalf("empty projection drift: %s", body)
	}
}

func TestMutationMethodsFailClosed(t *testing.T) {
	handler := New(func() []playlists.Playlist { return []playlists.Playlist{} })
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/playlists", strings.NewReader(`{}`)))
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("method contract drift: %d %q", response.Code, response.Header().Get("Allow"))
	}
}

func TestHandlerReadsFreshSnapshotPerRequest(t *testing.T) {
	current := []playlists.Playlist{{ID: "a", Name: "A"}}
	handler := New(func() []playlists.Playlist { return current })

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/playlists", nil))
	current = []playlists.Playlist{{ID: "b", Name: "B"}}
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/playlists", nil))

	if !strings.Contains(first.Body.String(), `"id":"a"`) || !strings.Contains(second.Body.String(), `"id":"b"`) || strings.Contains(second.Body.String(), `"id":"a"`) {
		t.Fatalf("snapshot currentness drift: first=%s second=%s", first.Body.String(), second.Body.String())
	}
}
