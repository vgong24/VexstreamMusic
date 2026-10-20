package lifecycle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassifySessionsRetiresOnlyIdleCoveredSession(t *testing.T) {
	sessions := []LiveSession{
		{
			PID: 100, Version: "4.0.0", Current: true,
			LibraryVerified: true, ImportsVerified: true,
			TrackRefs: []string{"a", "b"}, TrackCount: 2,
		},
		{
			PID: 200, Version: "3.0.0",
			LibraryVerified: true, ImportsVerified: true,
			TrackRefs: []string{"a"}, TrackCount: 1,
		},
		{
			PID: 300, Version: "3.0.0",
			LibraryVerified: true, ImportsVerified: true,
			TrackRefs: []string{"a"}, TrackCount: 1, ActiveImports: 1,
		},
		{
			PID: 400, Version: "3.0.0",
			LibraryVerified: true, ImportsVerified: true,
			TrackRefs: []string{"c"}, TrackCount: 1,
		},
		{
			PID: 500, Version: "3.0.0",
			LibraryVerified: true, ImportsVerified: false,
			TrackRefs: []string{"a"}, TrackCount: 1,
		},
		{
			PID: 600, Version: "3.0.0",
			LibraryVerified: false, ImportsVerified: true,
		},
	}

	plan := ClassifySessions(100, []string{"a", "b"}, sessions)
	if len(plan.Eligible) != 1 || plan.Eligible[0].PID != 200 {
		t.Fatalf("eligible session drift: %+v", plan.Eligible)
	}
	if len(plan.Held) != 4 {
		t.Fatalf("held session count drift: %+v", plan.Held)
	}

	reasons := map[int]string{}
	for _, held := range plan.Held {
		reasons[held.PID] = held.Reason
	}
	if reasons[300] == "" ||
		reasons[400] == "" ||
		reasons[500] == "" ||
		reasons[600] == "" {
		t.Fatalf("held session reasons incomplete: %+v", plan.Held)
	}
	if plan.Eligible[0].UniqueTrackCount != 0 ||
		!plan.Eligible[0].Eligible {
		t.Fatalf("eligible safety projection drift: %+v", plan.Eligible[0])
	}
}

func TestSessionInspectionUnderstandsV3AndV4JobShapes(t *testing.T) {
	v4 := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		switch r.URL.Path {
		case "/api/library":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tracks": []map[string]any{{"id": "a"}, {"id": "b"}},
			})
		case "/api/import/jobs":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"groups": map[string]any{
					"active": []map[string]any{{"id": "import-v4"}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer v4.Close()

	if refs, ok := sessionTrackRefs(v4.URL); !ok ||
		len(refs) != 2 ||
		refs[0] != "a" ||
		refs[1] != "b" {
		t.Fatalf("V4 Library inspection drift: refs=%v ok=%v", refs, ok)
	}
	if active, ok := sessionActiveImports(v4.URL); !ok || active != 1 {
		t.Fatalf("V4 import inspection drift: active=%d ok=%v", active, ok)
	}

	v3 := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		switch r.URL.Path {
		case "/api/library":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tracks": []map[string]any{{"id": "legacy-a"}},
			})
		case "/api/import/jobs":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"active": []map[string]any{
					{"id": "legacy-1"},
					{"id": "legacy-2"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer v3.Close()

	if refs, ok := sessionTrackRefs(v3.URL); !ok ||
		len(refs) != 1 ||
		refs[0] != "legacy-a" {
		t.Fatalf("V3 Library inspection drift: refs=%v ok=%v", refs, ok)
	}
	if active, ok := sessionActiveImports(v3.URL); !ok || active != 2 {
		t.Fatalf("V3 import inspection drift: active=%d ok=%v", active, ok)
	}
}

func TestRetirementRefusesCurrentOrHeldPIDBeforePlatformEffect(t *testing.T) {
	plan := SessionPlan{
		CurrentPID: 100,
		Eligible: []RetireCandidate{{
			PID: 200, Eligible: true,
		}},
	}
	retired, failed := RetireEligibleSessions(100, []int{100, 300}, plan)
	if len(retired) != 0 || len(failed) != 2 {
		t.Fatalf("unsafe retirement admission: retired=%v failed=%+v", retired, failed)
	}
}
