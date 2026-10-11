package main

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
)

func TestMobileCatalogSanitizesDesktopPathsAndBindsMediaBytes(t *testing.T) {
	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "private-library", "song.mp3")
	if err := os.MkdirAll(filepath.Dir(mediaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	mediaBytes := []byte("vexstream-test-media")
	if err := os.WriteFile(mediaPath, mediaBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	end := 18.0
	app := &App{
		tracks: []Track{{
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
		paths: map[string]string{
			"path-derived-track-ref": mediaPath,
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/catalog", nil)
	response := httptest.NewRecorder()
	app.mobileCatalog(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", response.Code, response.Body.String())
	}

	var catalog mobileCatalogProjection
	if err := json.Unmarshal(response.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tracks) != 1 {
		t.Fatalf("tracks=%d", len(catalog.Tracks))
	}

	digest := sha256.Sum256(mediaBytes)
	expectedDigest := hex.EncodeToString(digest[:])
	got := catalog.Tracks[0]
	if got.TrackRef != "path-derived-track-ref" {
		t.Fatalf("track ref=%q", got.TrackRef)
	}
	if got.MediaSHA256 != expectedDigest {
		t.Fatalf("media digest=%q want=%q", got.MediaSHA256, expectedDigest)
	}
	if got.MediaByteLength != int64(len(mediaBytes)) {
		t.Fatalf("media length=%d", got.MediaByteLength)
	}
	if got.PlaybackStartSeconds != 2 || got.PlaybackEndSeconds == nil || *got.PlaybackEndSeconds != 18 {
		t.Fatalf("playback window=%v..%v", got.PlaybackStartSeconds, got.PlaybackEndSeconds)
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

func TestMobileMediaSupportsExactRangeReads(t *testing.T) {
	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "song.mp3")
	mediaBytes := []byte("0123456789")
	if err := os.WriteFile(mediaPath, mediaBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	app := &App{
		paths: map[string]string{
			"track-ref": mediaPath,
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/media/track-ref", nil)
	request.Header.Set("Range", "bytes=2-5")
	response := httptest.NewRecorder()
	app.mobileMedia(response, request)

	if response.Code != http.StatusPartialContent {
		t.Fatalf("range status=%d body=%q", response.Code, response.Body.String())
	}
	if !bytes.Equal(response.Body.Bytes(), []byte("2345")) {
		t.Fatalf("range body=%q", response.Body.Bytes())
	}
	if response.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("Accept-Ranges=%q", response.Header().Get("Accept-Ranges"))
	}
}

func TestMobileBridgeRejectsMutationMethods(t *testing.T) {
	app := &App{paths: map[string]string{}}

	for _, target := range []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/v1/health", app.mobileHealth},
		{"/v1/catalog", app.mobileCatalog},
		{"/v1/media/track-ref", app.mobileMedia},
	} {
		request := httptest.NewRequest(http.MethodPost, target.path, nil)
		response := httptest.NewRecorder()
		target.handler(response, request)
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s POST status=%d", target.path, response.Code)
		}
	}
}
