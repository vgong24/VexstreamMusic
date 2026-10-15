package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/trash"
)

type trashFixture struct {
	snapshot trash.Snapshot
	moved    []trash.MoveInput
	moveErr  error
	restoreIDs []string
	restoreErr error
	purgeIDs []string
	preserve bool
	purgeErr error
	recoverErr error
}

func (f *trashFixture) Snapshot() trash.Snapshot { return f.snapshot }
func (f *trashFixture) Move(input []trash.MoveInput) ([]trash.Item, error) {
	f.moved = append([]trash.MoveInput(nil), input...)
	if f.moveErr != nil { return nil, f.moveErr }
	return []trash.Item{{ID: "trash-a"}}, nil
}
func (f *trashFixture) Restore(ids []string) ([]trash.Item, error) {
	f.restoreIDs = append([]string(nil), ids...)
	if f.restoreErr != nil { return nil, f.restoreErr }
	return []trash.Item{{ID: "trash-a"}}, nil
}
func (f *trashFixture) Purge(ids []string, preserve bool) (int, error) {
	f.purgeIDs = append([]string(nil), ids...)
	f.preserve = preserve
	if f.purgeErr != nil { return 0, f.purgeErr }
	return len(ids), nil
}
func (f *trashFixture) RecoverPurge() (int, error) {
	if f.recoverErr != nil { return 0, f.recoverErr }
	return 1, nil
}

func libraryFixture() library.Projection {
	return library.Projection{
		Tracks: []library.Track{{
			ID: "track-a",
			Title: "Song",
			Source: "/music",
			Filename: "song.mp3",
		}},
		Paths: map[string]string{"track-a": "/music/song.mp3"},
	}
}

func TestMoveRequiresExplicitConfirmationAndCurrentTrack(t *testing.T) {
	store := &trashFixture{}
	handler := New(store, libraryFixture, nil)

	noConfirm := httptest.NewRecorder()
	handler.ServeHTTP(
		noConfirm,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/trash",
			strings.NewReader(`{"ids":["track-a"],"confirm":"no"}`),
		),
	)
	if noConfirm.Code != http.StatusBadRequest || len(store.moved) != 0 {
		t.Fatalf("Trash move admitted without confirmation: %d %+v", noConfirm.Code, store.moved)
	}

	ok := httptest.NewRecorder()
	handler.ServeHTTP(
		ok,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/trash",
			strings.NewReader(`{"ids":["track-a","track-a"],"confirm":"MOVE_TO_VEXSTREAM_TRASH"}`),
		),
	)
	if ok.Code != http.StatusOK ||
		len(store.moved) != 1 ||
		store.moved[0].Track.ID != "track-a" ||
		store.moved[0].Path != "/music/song.mp3" {
		t.Fatalf("Trash move mapping drift: %d %+v %s", ok.Code, store.moved, ok.Body.String())
	}
}

func TestCustodySuccessAndRefreshFailureRemainDistinct(t *testing.T) {
	store := &trashFixture{}
	handler := New(
		store,
		libraryFixture,
		func() error { return errors.New("scan unavailable") },
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/trash",
			strings.NewReader(`{"ids":["track-a"],"confirm":"MOVE_TO_VEXSTREAM_TRASH"}`),
		),
	)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"moved":1`) ||
		!strings.Contains(response.Body.String(), `"libraryRefresh":"FAILED"`) {
		t.Fatalf("Trash custody was collapsed into refresh failure: %d %s", response.Code, response.Body.String())
	}
}

func TestRestoreAndPurgeUseTrashItemIdentityNotTrackIdentity(t *testing.T) {
	store := &trashFixture{}
	handler := New(store, libraryFixture, nil)

	restore := httptest.NewRecorder()
	handler.ServeHTTP(
		restore,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/trash/restore",
			strings.NewReader(`{"ids":["trash-a"]}`),
		),
	)
	if restore.Code != http.StatusOK ||
		len(store.restoreIDs) != 1 ||
		store.restoreIDs[0] != "trash-a" {
		t.Fatalf("restore identity drift: %d %v", restore.Code, store.restoreIDs)
	}

	purge := httptest.NewRecorder()
	handler.ServeHTTP(
		purge,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/trash/purge",
			strings.NewReader(`{"ids":["trash-a"],"confirm":"PERMANENTLY_DELETE_TRASH","preserveMetadata":true}`),
		),
	)
	if purge.Code != http.StatusOK ||
		len(store.purgeIDs) != 1 ||
		store.purgeIDs[0] != "trash-a" ||
		!store.preserve {
		t.Fatalf("purge identity drift: %d ids=%v preserve=%v", purge.Code, store.purgeIDs, store.preserve)
	}
}

func TestPendingPurgeRecoveryHasSeparateExplicitRoute(t *testing.T) {
	store := &trashFixture{}
	handler := New(store, libraryFixture, nil)

	bad := httptest.NewRecorder()
	handler.ServeHTTP(
		bad,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/trash/purge/recover",
			strings.NewReader(`{"confirm":"no"}`),
		),
	)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("purge recovery admitted without confirmation: %d", bad.Code)
	}

	ok := httptest.NewRecorder()
	handler.ServeHTTP(
		ok,
		httptest.NewRequest(
			http.MethodPost,
			"/api/library/trash/purge/recover",
			strings.NewReader(`{"confirm":"RECOVER_ADMITTED_TRASH_PURGE"}`),
		),
	)
	if ok.Code != http.StatusOK ||
		!strings.Contains(ok.Body.String(), `"recovered":1`) {
		t.Fatalf("purge recovery route drift: %d %s", ok.Code, ok.Body.String())
	}
}

func TestStableTrashErrorsMapToLocalHTTPMeaning(t *testing.T) {
	cases := []struct{
		name string
		err error
		want int
	}{
		{"not-found", trash.ErrNotFound, http.StatusNotFound},
		{"restore-collision", trash.ErrRestoreCollision, http.StatusConflict},
		{"pending-purge", trash.ErrPurgeRecoveryRequired, http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &trashFixture{restoreErr: tc.err}
			response := httptest.NewRecorder()
			New(store, libraryFixture, nil).ServeHTTP(
				response,
				httptest.NewRequest(
					http.MethodPost,
					"/api/library/trash/restore",
					strings.NewReader(`{"ids":["trash-a"]}`),
				),
			)
			if response.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.want, response.Body.String())
			}
		})
	}
}

func TestTrashSnapshotPreservesPendingPurgeEvidence(t *testing.T) {
	store := &trashFixture{
		snapshot: trash.Snapshot{
			Trash: []trash.Item{{ID: "trash-a"}},
			PendingPurge: &trash.PurgeJournal{
				SchemaVersion: trash.PurgeJournalSchema,
				PurgeRef: "purge-a",
				State: trash.PurgeStateStoresUpdated,
				Entries: []trash.PurgeEntry{{Item: trash.Item{ID: "trash-a"}}},
			},
		},
	}
	response := httptest.NewRecorder()
	New(store, libraryFixture, nil).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/library/trash", nil),
	)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), "purge-a") ||
		!strings.Contains(response.Body.String(), trash.PurgeStateStoresUpdated) {
		t.Fatalf("pending purge evidence lost: %d %s", response.Code, response.Body.String())
	}
}
