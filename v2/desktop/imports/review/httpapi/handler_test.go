package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/duplicates"
	importprovider "github.com/vgong24/VexstreamMusic/v2/desktop/imports/provider"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/review"
	youtubeprovider "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube"
)

type reviewerFixture struct {
	result review.Result
	err    error
	seen   string
}

func (f *reviewerFixture) InspectAndReview(
	_ context.Context,
	url string,
) (review.Result, error) {
	f.seen = url
	return f.result, f.err
}

func TestInspectReturnsProviderMetadataAndAdvisoryMatchesWithoutMutation(t *testing.T) {
	reviewer := &reviewerFixture{
		result: review.Result{
			Inspection: importprovider.Inspection{
				ProviderID: "abc",
				URL: "https://example/song",
				Title: "Song",
				Artist: "Artist",
				Channel: "Uploader",
				Duration: 180,
				Sections: []importprovider.Section{{
					Index: 1,
					Start: 0,
					End: 30,
					Title: "Part 1",
					Selected: true,
				}},
				HasSections: true,
				SourceQuality: duplicates.AudioQuality{
					Codec: "opus",
					BitrateKbps: 160,
				},
			},
			Items: []duplicates.Item{{
				Kind: "whole",
				Title: "Song",
				Matches: []duplicates.Match{{
					TrackRef: "track-a",
					Score: 100,
					Relation: "SAME_SOURCE",
				}},
			}},
			MatchCount: 1,
			Advisory: true,
		},
	}
	handler := New(reviewer)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/inspect",
			strings.NewReader(`{"url":" https://example/song "}`),
		),
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if reviewer.seen != "https://example/song" {
		t.Fatalf("URL normalization drift: %q", reviewer.seen)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["advisory"] != true || body["matchCount"] != float64(1) {
		t.Fatalf("review envelope drift: %+v", body)
	}
	inspection, _ := body["inspection"].(map[string]any)
	if inspection["id"] != "abc" || inspection["title"] != "Song" {
		t.Fatalf("inspection projection drift: %+v", inspection)
	}
}

func TestInspectRejectsWrongMethodUnknownFieldsAndBlankURL(t *testing.T) {
	handler := New(&reviewerFixture{})

	wrong := httptest.NewRecorder()
	handler.ServeHTTP(
		wrong,
		httptest.NewRequest(http.MethodGet, "/api/import/inspect", nil),
	)
	if wrong.Code != http.StatusMethodNotAllowed ||
		wrong.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("method contract drift: %d %q", wrong.Code, wrong.Header().Get("Allow"))
	}

	unknown := httptest.NewRecorder()
	handler.ServeHTTP(
		unknown,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/inspect",
			strings.NewReader(`{"url":"https://example","commit":true}`),
		),
	)
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown field accepted: %d", unknown.Code)
	}

	blank := httptest.NewRecorder()
	handler.ServeHTTP(
		blank,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/inspect",
			strings.NewReader(`{"url":" "}`),
		),
	)
	if blank.Code != http.StatusBadRequest {
		t.Fatalf("blank URL accepted: %d", blank.Code)
	}
}

func TestProviderFailurePreservesStructuredProblem(t *testing.T) {
	reviewer := &reviewerFixture{
		err: &youtubeprovider.Failure{
			Problem: youtubeprovider.Problem{
				SchemaVersion: "vexstream.provider-problem/v1",
				Provider: "youtube",
				Stage: "inspect",
				Code: "RATE_LIMIT",
				Title: "Provider busy",
				Message: "Retry later",
				Retryable: true,
			},
		},
	}
	handler := New(reviewer)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/inspect",
			strings.NewReader(`{"url":"https://example/song"}`),
		),
	)
	if response.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "RATE_LIMIT") ||
		!strings.Contains(response.Body.String(), "problem") {
		t.Fatalf("structured provider problem lost: %s", response.Body.String())
	}
}

func TestUnexpectedReviewFailureRemainsBadGateway(t *testing.T) {
	handler := New(&reviewerFixture{err: errors.New("review unavailable")})
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"/api/import/inspect",
			strings.NewReader(`{"url":"https://example/song"}`),
		),
	)
	if response.Code != http.StatusBadGateway ||
		!strings.Contains(response.Body.String(), "review unavailable") {
		t.Fatalf("failure mapping drift: %d %s", response.Code, response.Body.String())
	}
}
