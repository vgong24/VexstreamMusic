package mobile

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
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

func writeEmbeddedArtworkMP3(
	t *testing.T,
	path string,
	artwork []byte,
) {
	t.Helper()

	payload :=
		append(
			[]byte{3},
			[]byte("image/jpeg")...,
		)
	payload =
		append(
			payload,
			0,
			3,
			0,
		)
	payload =
		append(
			payload,
			artwork...,
		)

	frame :=
		make(
			[]byte,
			10+
				len(payload),
		)
	copy(
		frame[:4],
		[]byte("APIC"),
	)
	binary.BigEndian.PutUint32(
		frame[4:8],
		uint32(
			len(payload),
		),
	)
	copy(
		frame[10:],
		payload,
	)

	header :=
		make(
			[]byte,
			10,
		)
	copy(
		header[:3],
		[]byte("ID3"),
	)
	header[3] = 3
	size := len(frame)
	header[6] =
		byte(
			(size >> 21) &
				0x7f,
		)
	header[7] =
		byte(
			(size >> 14) &
				0x7f,
		)
	header[8] =
		byte(
			(size >> 7) &
				0x7f,
		)
	header[9] =
		byte(
			size &
				0x7f,
		)

	data :=
		append(
			header,
			frame...,
		)
	data =
		append(
			data,
			[]byte("audio")...,
		)
	if err :=
		os.WriteFile(
			path,
			data,
			0o644,
		); err != nil {
		t.Fatal(err)
	}
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
				ThumbnailURL:  "https://images.example.test/artwork/on-the-go.jpg",
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
	if got.ArtworkURL != "https://images.example.test/artwork/on-the-go.jpg" {
		t.Fatalf("artwork url=%q", got.ArtworkURL)
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

func TestCatalogProjectsEmbeddedArtworkRefAndServesBytes(
	t *testing.T,
) {
	root :=
		t.TempDir()
	mediaPath :=
		filepath.Join(
			root,
			"embedded.mp3",
		)
	artwork :=
		[]byte{
			0xff,
			0xd8,
			0xff,
			0xe0,
			0x4a,
			0x46,
			0x49,
			0x46,
		}
	writeEmbeddedArtworkMP3(
		t,
		mediaPath,
		artwork,
	)

	service :=
		testService(
			t,
			library.Projection{
				Tracks: []library.Track{{
					ID:       "track-with-cover",
					Path:     mediaPath,
					Filename: "embedded.mp3",
					Title:    "Embedded",
					Artist:   "Vex",
				}},
				Paths: map[string]string{
					"track-with-cover": mediaPath,
				},
			},
			nil,
		)

	catalogRequest :=
		httptest.NewRequest(
			http.MethodGet,
			"/v1/catalog",
			nil,
		)
	catalogResponse :=
		httptest.NewRecorder()
	service.mobileCatalog(
		catalogResponse,
		catalogRequest,
	)
	if catalogResponse.Code !=
		http.StatusOK {
		t.Fatalf(
			"catalog status=%d body=%s",
			catalogResponse.Code,
			catalogResponse.Body.String(),
		)
	}

	var catalog catalogProjection
	if err :=
		json.Unmarshal(
			catalogResponse.Body.Bytes(),
			&catalog,
		); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tracks) != 1 ||
		catalog.Tracks[0].ArtworkRef !=
			"track-with-cover" {
		t.Fatalf(
			"artwork projection=%+v",
			catalog.Tracks,
		)
	}

	artworkRequest :=
		httptest.NewRequest(
			http.MethodGet,
			"/v1/artwork/track-with-cover",
			nil,
		)
	artworkResponse :=
		httptest.NewRecorder()
	service.mobileArtwork(
		artworkResponse,
		artworkRequest,
	)
	if artworkResponse.Code !=
		http.StatusOK {
		t.Fatalf(
			"artwork status=%d body=%s",
			artworkResponse.Code,
			artworkResponse.Body.String(),
		)
	}
	if artworkResponse.Header().Get(
		"Content-Type",
	) != "image/jpeg" {
		t.Fatalf(
			"content type=%q",
			artworkResponse.Header().Get(
				"Content-Type",
			),
		)
	}
	if !bytes.Equal(
		artworkResponse.Body.Bytes(),
		artwork,
	) {
		t.Fatalf(
			"artwork=%x want=%x",
			artworkResponse.Body.Bytes(),
			artwork,
		)
	}
}

func TestEmbeddedArtworkAvailabilityCacheInvalidatesWithFileFingerprint(
	t *testing.T,
) {
	root :=
		t.TempDir()
	mediaPath :=
		filepath.Join(
			root,
			"changing.mp3",
		)
	writeEmbeddedArtworkMP3(
		t,
		mediaPath,
		[]byte{
			0xff,
			0xd8,
			0xff,
			0xe0,
		},
	)

	service :=
		testService(
			t,
			library.Projection{},
			nil,
		)

	available, err :=
		service.embeddedArtworkAvailable(
			mediaPath,
		)
	if err != nil {
		t.Fatal(err)
	}
	if !available {
		t.Fatal(
			"embedded artwork was not detected",
		)
	}

	if err :=
		os.WriteFile(
			mediaPath,
			[]byte(
				"plain audio without id3",
			),
			0o644,
		); err != nil {
		t.Fatal(err)
	}

	available, err =
		service.embeddedArtworkAvailable(
			mediaPath,
		)
	if err != nil {
		t.Fatal(err)
	}
	if available {
		t.Fatal(
			"stale embedded-artwork availability survived a changed file fingerprint",
		)
	}
}

func TestProjectArtworkURLAllowsOnlyHttps(t *testing.T) {
	cases := map[string]string{
		"":                                      "",
		"   ":                                   "",
		"file:///private/cover.jpg":             "",
		"http://images.example.test/cover.jpg":  "",
		"javascript:alert(1)":                   "",
		"https://images.example.test/cover.jpg": "https://images.example.test/cover.jpg",
		" https://images.example.test/cover.jpg ": "https://images.example.test/cover.jpg",
	}

	for input, expected := range cases {
		if got := projectArtworkURL(input); got != expected {
			t.Fatalf("projectArtworkURL(%q)=%q want=%q", input, got, expected)
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
		{"/v1/artwork/track-ref", service.mobileArtwork},
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
