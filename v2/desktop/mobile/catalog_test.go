package mobile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
)

func testService(
	t *testing.T,
	projection library.Projection,
	rows []playlists.Playlist,
) *Service {
	t.Helper()

	service, err := New(Options{
		Root:    t.TempDir(),
		Version: "4.0.0",
		LibrarySnapshot: func() library.Projection {
			return projection
		},
		PlaylistSnapshot: func() []playlists.Playlist {
			return rows
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestCatalogSanitizesDesktopPathsAndBindsMediaBytes(t *testing.T) {
	root := t.TempDir()
	mediaPath := filepath.Join(
		root,
		"private-library",
		"song.mp3",
	)
	if err := os.MkdirAll(filepath.Dir(mediaPath), 0o755); err != nil {
		t.Fatal(err)
	}

	mediaBytes := []byte("vexstream-test-media")
	if err := os.WriteFile(mediaPath, mediaBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	end := 18.0
	service := testService(
		t,
		library.Projection{
			Tracks: []library.Track{{
				ID:            "path-derived-track-ref",
				Path:          mediaPath,
				Filename:      "song.mp3",
				Title:         "On The Go",
				Artist:        "Vex",
				Album:         "Mobile",
				Genre:         "Test",
				Year:          "2026",
				Duration:      20,
				Source:        filepath.Dir(mediaPath),
				PlaybackStart: 2,
				PlaybackEnd:   &end,
			}},
			Paths: map[string]string{
				"path-derived-track-ref": mediaPath,
			},
		},
		nil,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/v1/catalog",
		nil,
	)
	response := httptest.NewRecorder()
	service.mobileCatalog(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"catalog status=%d body=%s",
			response.Code,
			response.Body.String(),
		)
	}

	var catalog catalogProjection
	if err := json.Unmarshal(response.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tracks) != 1 {
		t.Fatalf("tracks=%d", len(catalog.Tracks))
	}

	digest := sha256.Sum256(mediaBytes)
	got := catalog.Tracks[0]
	if got.MediaSHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("media digest=%q", got.MediaSHA256)
	}
	if got.MediaByteLength != int64(len(mediaBytes)) {
		t.Fatalf("media length=%d", got.MediaByteLength)
	}
	if got.PlaybackStartSeconds != 2 ||
		got.PlaybackEndSeconds == nil ||
		*got.PlaybackEndSeconds != 18 {
		t.Fatalf(
			"playback window=%v..%v",
			got.PlaybackStartSeconds,
			got.PlaybackEndSeconds,
		)
	}

	body := response.Body.String()
	for _, forbidden := range []string{
		mediaPath,
		filepath.Dir(mediaPath),
		`"path"`,
		`"filename"`,
		`"source"`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("catalog leaked %q: %s", forbidden, body)
		}
	}
}

func TestCatalogProjectsDesktopPlaylistsWithoutMissingTracks(t *testing.T) {
	projected := projectPlaylists(
		[]playlists.Playlist{{
			ID:        "playlist.one",
			Name:      "Road Trip",
			TrackRefs: []string{"track-b", "missing-track", "track-a"},
		}},
		map[string]bool{
			"track-a": true,
			"track-b": true,
		},
	)

	if len(projected) != 1 {
		t.Fatalf("playlists=%d", len(projected))
	}
	if projected[0].PlaylistRef != "playlist.one" ||
		projected[0].Name != "Road Trip" {
		t.Fatalf("playlist=%+v", projected[0])
	}
	if got := strings.Join(projected[0].TrackRefs, ","); got != "track-b,track-a" {
		t.Fatalf("trackRefs=%q", got)
	}
}

func TestMediaSupportsExactRangeReads(t *testing.T) {
	root := t.TempDir()
	mediaPath := filepath.Join(root, "song.mp3")
	if err := os.WriteFile(
		mediaPath,
		[]byte("0123456789"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	service := testService(
		t,
		library.Projection{
			Paths: map[string]string{
				"track-ref": mediaPath,
			},
		},
		nil,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/v1/media/track-ref",
		nil,
	)
	request.Header.Set("Range", "bytes=2-5")
	response := httptest.NewRecorder()
	service.mobileMedia(response, request)

	if response.Code != http.StatusPartialContent {
		t.Fatalf(
			"range status=%d body=%q",
			response.Code,
			response.Body.String(),
		)
	}
	if !bytes.Equal(response.Body.Bytes(), []byte("2345")) {
		t.Fatalf("range body=%q", response.Body.Bytes())
	}
	if response.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf(
			"Accept-Ranges=%q",
			response.Header().Get("Accept-Ranges"),
		)
	}
}

func TestMobileProjectionRejectsMutationMethods(t *testing.T) {
	service := testService(t, library.Projection{}, nil)

	for _, target := range []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/v1/health", service.lanHealth},
		{"/v1/catalog", service.mobileCatalog},
		{"/v1/media/track-ref", service.mobileMedia},
	} {
		request := httptest.NewRequest(
			http.MethodPost,
			target.path,
			nil,
		)
		response := httptest.NewRecorder()
		target.handler(response, request)

		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf(
				"%s POST status=%d",
				target.path,
				response.Code,
			)
		}
	}
}
