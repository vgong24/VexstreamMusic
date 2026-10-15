package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	libraryservice "github.com/vgong24/VexstreamMusic/v2/desktop/library/service"
)

type mutatorFixture struct {
	overlayRefs    []string
	overlayPatch   libraryservice.OverlayPatch
	overlayRestore bool
	overlayErr     error

	windowRef     string
	windowStart   float64
	windowEnd     *float64
	windowRestore bool
	windowErr     error
}

func (m *mutatorFixture) UpdateOverlays(
	refs []string,
	patch libraryservice.OverlayPatch,
	restore bool,
) (int, error) {
	m.overlayRefs = append([]string(nil), refs...)
	m.overlayPatch = patch
	m.overlayRestore = restore
	return len(refs), m.overlayErr
}

func (m *mutatorFixture) UpdatePlaybackWindow(
	ref string,
	start float64,
	end *float64,
	restore bool,
) (library.PlaybackWindow, bool, error) {
	m.windowRef = ref
	m.windowStart = start
	m.windowRestore = restore
	if end != nil {
		value := *end
		m.windowEnd = &value
	} else {
		m.windowEnd = nil
	}
	if m.windowErr != nil {
		return library.PlaybackWindow{}, false, m.windowErr
	}
	if restore || (start == 0 && end == nil) {
		return library.PlaybackWindow{}, false, nil
	}
	return library.PlaybackWindow{
		Start: start,
		End: end,
		UpdatedAt: "now",
	}, true, nil
}

func snapshotTrack(duration float64) library.Projection {
	return library.Projection{
		Tracks: []library.Track{{
			ID: "track-a",
			Title: "Song",
			Duration: duration,
		}},
		Paths: map[string]string{
			"track-a": "/music/song.mp3",
		},
	}
}

func TestOverlayPreservesExplicitBlankAndDedupesTargets(t *testing.T) {
	mutator := &mutatorFixture{}
	handler := New(
		func() library.Projection { return snapshotTrack(120) },
		mutator,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/overlay",
			strings.NewReader(
				`{"id":"track-a","ids":["track-a"],"artist":""}`,
			),
		),
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if len(mutator.overlayRefs) != 1 ||
		mutator.overlayRefs[0] != "track-a" ||
		mutator.overlayPatch.Artist == nil ||
		*mutator.overlayPatch.Artist != "" {
		t.Fatalf("overlay mapping drift: refs=%v patch=%+v", mutator.overlayRefs, mutator.overlayPatch)
	}
}

func TestOverlayRejectsStaleTrackBeforeMutation(t *testing.T) {
	mutator := &mutatorFixture{}
	handler := New(
		func() library.Projection { return library.Projection{Paths: map[string]string{}} },
		mutator,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/overlay",
			strings.NewReader(`{"id":"gone","title":"Edit"}`),
		),
	)
	if response.Code != http.StatusNotFound ||
		len(mutator.overlayRefs) != 0 {
		t.Fatalf("stale overlay admitted: %d refs=%v", response.Code, mutator.overlayRefs)
	}
}

func TestPlaybackWindowClampsToDurationAndDropsFullLengthEnd(t *testing.T) {
	mutator := &mutatorFixture{}
	handler := New(
		func() library.Projection { return snapshotTrack(120) },
		mutator,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/playback-window",
			strings.NewReader(`{"id":"track-a","start":5,"end":500}`),
		),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if mutator.windowStart != 5 || mutator.windowEnd != nil {
		t.Fatalf("full-duration end should collapse to nil: start=%v end=%v", mutator.windowStart, mutator.windowEnd)
	}
}

func TestPlaybackWindowRejectsInvalidBoundariesBeforeMutation(t *testing.T) {
	for _, body := range []string{
		`{"id":"track-a","start":120}`,
		`{"id":"track-a","start":20,"end":20.01}`,
	} {
		t.Run(body, func(t *testing.T) {
			mutator := &mutatorFixture{}
			handler := New(
				func() library.Projection { return snapshotTrack(120) },
				mutator,
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(
				response,
				httptest.NewRequest(
					http.MethodPost,
					"/api/library/playback-window",
					strings.NewReader(body),
				),
			)
			if response.Code != http.StatusBadRequest ||
				mutator.windowRef != "" {
				t.Fatalf("invalid window admitted: %d ref=%q", response.Code, mutator.windowRef)
			}
		})
	}
}

func TestPlaybackRestoreIsExplicitAndProviderIndependent(t *testing.T) {
	mutator := &mutatorFixture{}
	handler := New(
		func() library.Projection { return snapshotTrack(120) },
		mutator,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/playback-window",
			strings.NewReader(`{"id":"track-a","restore":true}`),
		),
	)
	if response.Code != http.StatusOK ||
		!mutator.windowRestore ||
		mutator.windowRef != "track-a" {
		t.Fatalf("playback restore drift: %d %+v", response.Code, mutator)
	}
}

func TestMutationFailureStaysServerFailure(t *testing.T) {
	mutator := &mutatorFixture{overlayErr: errors.New("disk unavailable")}
	handler := New(
		func() library.Projection { return snapshotTrack(120) },
		mutator,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/overlay",
			strings.NewReader(`{"id":"track-a","title":"Edit"}`),
		),
	)
	if response.Code != http.StatusInternalServerError ||
		!strings.Contains(response.Body.String(), "disk unavailable") {
		t.Fatalf("mutation failure mapping drift: %d %s", response.Code, response.Body.String())
	}
}
