package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

func fixtureProjection(t *testing.T) (library.Projection, []byte) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "tone.mp3")
	data := bytes.Repeat([]byte("vexstream-range-fixture-"), 64)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	end := 42.5
	return library.Projection{
		Tracks: []library.Track{{
			ID: "track-a", Path: path, Filename: "tone.mp3", Source: root,
			Title: "Display Tone", Artist: "Vex", Album: "V4", Genre: "Test", Year: "2026",
			Duration: 90, SourceTitle: "Source Tone", SourceArtist: "Source Artist",
			PlaybackStart: 1.25, PlaybackEnd: &end,
		}},
		Paths: map[string]string{"track-a": path},
	}, data
}

func TestCatalogIsReadOnlyBrowserProjectionWithoutFilesystemCustody(t *testing.T) {
	projection, _ := fixtureProjection(t)
	handler := New(func() library.Projection { return projection })
	request := httptest.NewRequest(http.MethodGet, "/api/library", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status: %d", response.Code)
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("content type: %q", got)
	}
	raw := response.Body.String()
	if strings.Contains(raw, projection.Paths["track-a"]) || strings.Contains(raw, `"path"`) || strings.Contains(raw, `"source"`) || strings.Contains(raw, `"filename"`) {
		t.Fatalf("filesystem custody leaked into browser catalog: %s", raw)
	}
	var result struct {
		Tracks []map[string]any `json:"tracks"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Tracks) != 1 || result.Tracks[0]["id"] != "track-a" || result.Tracks[0]["title"] != "Display Tone" {
		t.Fatalf("catalog drift: %+v", result)
	}
	if result.Tracks[0]["sourceTitle"] != "Source Tone" || result.Tracks[0]["playbackStart"] != 1.25 {
		t.Fatalf("display/source projection missing: %+v", result.Tracks[0])
	}
}

func TestCatalogRejectsMutationMethods(t *testing.T) {
	handler := New(func() library.Projection { return library.Projection{} })
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/library", nil))
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET" {
		t.Fatalf("method contract drift: %d %q", response.Code, response.Header().Get("Allow"))
	}
}

func TestMediaServesExactBytesAndRanges(t *testing.T) {
	projection, data := fixtureProjection(t)
	handler := New(func() library.Projection { return projection })
	full := httptest.NewRecorder()
	handler.ServeHTTP(full, httptest.NewRequest(http.MethodGet, "/media/track-a", nil))
	if full.Code != http.StatusOK || !bytes.Equal(full.Body.Bytes(), data) {
		t.Fatalf("full media drift: %d bytes=%d", full.Code, full.Body.Len())
	}
	if full.Header().Get("Accept-Ranges") != "bytes" || full.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("media headers drift: %+v", full.Header())
	}
	req := httptest.NewRequest(http.MethodGet, "/media/track-a", nil)
	req.Header.Set("Range", "bytes=0-31")
	partial := httptest.NewRecorder()
	handler.ServeHTTP(partial, req)
	if partial.Code != http.StatusPartialContent || !bytes.Equal(partial.Body.Bytes(), data[:32]) {
		t.Fatalf("range drift: %d %q", partial.Code, partial.Body.Bytes())
	}
	if got := partial.Header().Get("Content-Range"); !strings.HasPrefix(got, "bytes 0-31/") {
		t.Fatalf("content range: %q", got)
	}
}

func TestMediaHeadHasNoBody(t *testing.T) {
	projection, _ := fixtureProjection(t)
	handler := New(func() library.Projection { return projection })
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodHead, "/media/track-a", nil))
	body, err := io.ReadAll(response.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(body) != 0 {
		t.Fatalf("HEAD drift: status=%d body=%d", response.Code, len(body))
	}
}

func TestMediaRejectsUnknownTraversalAndMutation(t *testing.T) {
	projection, _ := fixtureProjection(t)
	handler := New(func() library.Projection { return projection })
	for _, target := range []string{"/media/missing", "/media/..%2Fsecret", "/media/%5Csecret"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d", target, response.Code)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/media/track-a", nil))
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("media mutation contract drift: %d %q", response.Code, response.Header().Get("Allow"))
	}
}

func TestHandlerReadsFreshSnapshotPerRequest(t *testing.T) {
	first, _ := fixtureProjection(t)
	current := first
	handler := New(func() library.Projection { return current })
	before := httptest.NewRecorder()
	handler.ServeHTTP(before, httptest.NewRequest(http.MethodGet, "/api/library", nil))
	current = library.Projection{Tracks: []library.Track{{ID: "track-b", Title: "Later"}}, Paths: map[string]string{}}
	after := httptest.NewRecorder()
	handler.ServeHTTP(after, httptest.NewRequest(http.MethodGet, "/api/library", nil))
	if !strings.Contains(before.Body.String(), "track-a") || !strings.Contains(after.Body.String(), "track-b") || strings.Contains(after.Body.String(), "track-a") {
		t.Fatalf("snapshot currentness drift: before=%s after=%s", before.Body.String(), after.Body.String())
	}
}
