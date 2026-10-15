package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	importmodel "github.com/vgong24/VexstreamMusic/v2/desktop/imports"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/jobs"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/workflow"
)

type managerFixture struct {
	snapshot jobs.Snapshot
	startJob importmodel.Job
	startErr error
	startReq workflow.Request

	cancelErr error
	cancelID string

	retryJob importmodel.Job
	retryErr error
	retryID string

	recoverJob importmodel.Job
	recoverErr error
	recoverID string

	discardJob importmodel.Job
	discardErr error
	discardID string
}

func (m *managerFixture) Snapshot() jobs.Snapshot { return m.snapshot }

func (m *managerFixture) Start(
	request workflow.Request,
) (importmodel.Job, error) {
	m.startReq = request
	return m.startJob, m.startErr
}

func (m *managerFixture) Cancel(id string) error {
	m.cancelID = id
	return m.cancelErr
}

func (m *managerFixture) RetryCommit(
	id string,
) (importmodel.Job, error) {
	m.retryID = id
	return m.retryJob, m.retryErr
}

func (m *managerFixture) RecoverCommit(
	id string,
) (importmodel.Job, error) {
	m.recoverID = id
	return m.recoverJob, m.recoverErr
}

func (m *managerFixture) Discard(
	id string,
) (importmodel.Job, error) {
	m.discardID = id
	return m.discardJob, m.discardErr
}

func TestJobsReturnsCurrentGroupedSnapshot(t *testing.T) {
	manager := &managerFixture{
		snapshot: jobs.Snapshot{
			Groups: importmodel.Groups{
				Ready: []importmodel.Job{{
					ID: "ready",
					Status: importmodel.StatusReady,
				}},
			},
			Recovery: map[string]importmodel.RestartDisposition{
				"ready": importmodel.RestartReadyStaging,
			},
		},
	}
	response := httptest.NewRecorder()
	New(manager).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/import/jobs", nil),
	)

	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"ready"`) ||
		!strings.Contains(response.Body.String(), "READY_STAGING") {
		t.Fatalf("job snapshot drift: %d %s", response.Code, response.Body.String())
	}
}

func TestStartMapsTypedPlanAndDefaultsSectionSelection(t *testing.T) {
	manager := &managerFixture{
		startJob: importmodel.Job{
			ID: "job-a",
			Status: importmodel.StatusQueued,
		},
	}
	response := httptest.NewRecorder()
	New(manager).ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/start",
			strings.NewReader(`{
			  "sourceTitle":" Song ",
			  "destination":" /music ",
			  "url":" https://example/song ",
			  "mode":"both",
			  "sections":[
			    {"index":1,"title":"One","start":0,"end":30},
			    {"index":2,"title":"Two","start":30,"end":60,"selected":false}
			  ]
			}`),
		),
	)

	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"jobId":"job-a"`) {
		t.Fatalf("start response drift: %d %s", response.Code, response.Body.String())
	}
	if manager.startReq.Title != "Song" ||
		manager.startReq.Destination != "/music" ||
		manager.startReq.URL != "https://example/song" ||
		len(manager.startReq.Sections) != 2 ||
		!manager.startReq.Sections[0].Selected ||
		manager.startReq.Sections[1].Selected {
		t.Fatalf("typed start mapping drift: %+v", manager.startReq)
	}
}

func TestStartRejectsUnknownFieldsBeforeManager(t *testing.T) {
	manager := &managerFixture{}
	response := httptest.NewRecorder()
	New(manager).ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/start",
			strings.NewReader(`{"url":"x","destination":"/music","autoCommitAnything":true}`),
		),
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown start field accepted: %d", response.Code)
	}
	if manager.startReq.URL != "" {
		t.Fatalf("manager saw rejected start request: %+v", manager.startReq)
	}
}

func TestCancelAndRecoveryRemainDifferentActions(t *testing.T) {
	manager := &managerFixture{
		recoverJob: importmodel.Job{
			ID: "uncertain",
			Status: importmodel.StatusDone,
		},
	}
	handler := New(manager)

	cancel := httptest.NewRecorder()
	handler.ServeHTTP(
		cancel,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/cancel",
			strings.NewReader(`{"id":"queued"}`),
		),
	)
	if cancel.Code != http.StatusOK || manager.cancelID != "queued" {
		t.Fatalf("cancel route drift: %d id=%q", cancel.Code, manager.cancelID)
	}

	recover := httptest.NewRecorder()
	handler.ServeHTTP(
		recover,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/recover-commit",
			strings.NewReader(`{"id":"uncertain"}`),
		),
	)
	if recover.Code != http.StatusOK ||
		manager.recoverID != "uncertain" ||
		manager.retryID != "" {
		t.Fatalf(
			"recovery bypassed explicit owner: code=%d recover=%q retry=%q",
			recover.Code,
			manager.recoverID,
			manager.retryID,
		)
	}
}

func TestManagerErrorsMapToStableHTTPMeaning(t *testing.T) {
	cases := []struct {
		name string
		path string
		err error
		want int
	}{
		{"missing", "/api/import/cancel", jobs.ErrNotFound, http.StatusNotFound},
		{"cancel-conflict", "/api/import/cancel", jobs.ErrNotCancellable, http.StatusConflict},
		{"retry-conflict", "/api/import/commit", jobs.ErrNotRetryable, http.StatusConflict},
		{"discard-conflict", "/api/import/discard", jobs.ErrNotDiscardable, http.StatusConflict},
		{"recovery-conflict", "/api/import/recover-commit", jobs.ErrRecoveryNotReady, http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := &managerFixture{}
			switch tc.path {
			case "/api/import/cancel":
				manager.cancelErr = tc.err
			case "/api/import/commit":
				manager.retryErr = tc.err
			case "/api/import/discard":
				manager.discardErr = tc.err
			case "/api/import/recover-commit":
				manager.recoverErr = tc.err
			}
			response := httptest.NewRecorder()
			New(manager).ServeHTTP(
				response,
				httptest.NewRequest(
					http.MethodPost,
					tc.path,
					strings.NewReader(`{"id":"job"}`),
				),
			)
			if response.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.want, response.Body.String())
			}
		})
	}
}

func TestWrongMethodsAndBlankJobIDFailClosed(t *testing.T) {
	handler := New(&managerFixture{})

	wrong := httptest.NewRecorder()
	handler.ServeHTTP(
		wrong,
		httptest.NewRequest(http.MethodPost, "/api/import/jobs", nil),
	)
	if wrong.Code != http.StatusMethodNotAllowed ||
		wrong.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("jobs method drift: %d %q", wrong.Code, wrong.Header().Get("Allow"))
	}

	blank := httptest.NewRecorder()
	handler.ServeHTTP(
		blank,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/discard",
			strings.NewReader(`{"id":" "}`),
		),
	)
	if blank.Code != http.StatusBadRequest {
		t.Fatalf("blank job id accepted: %d", blank.Code)
	}
}

func TestUnexpectedManagerFailureStaysServerFailure(t *testing.T) {
	manager := &managerFixture{
		discardErr: errors.New("filesystem unavailable"),
	}
	response := httptest.NewRecorder()
	New(manager).ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/discard",
			strings.NewReader(`{"id":"job"}`),
		),
	)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected error mapping drift: %d", response.Code)
	}
}
