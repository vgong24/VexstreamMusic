package lifecycle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestReusableRequiresExactVersionAndBuildRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(Identity{
			OK:       true,
			Version:  "4.0.0",
			BuildRef: "build-a",
		})
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "v4", "current-url.txt")
	if err := Claim(path, server.URL); err != nil {
		t.Fatal(err)
	}
	if url, ok := Reusable(path, "4.0.0", "build-a"); !ok || url != server.URL {
		t.Fatalf("exact process identity not reused: url=%q ok=%v", url, ok)
	}
	for _, tc := range []struct {
		version string
		build   string
	}{
		{"3.0.0", "build-a"},
		{"4.0.0", "build-b"},
	} {
		if _, ok := Reusable(path, tc.version, tc.build); ok {
			t.Fatalf("mismatched identity reused: %+v", tc)
		}
	}
}

func TestClaimAndClearNeverDeleteAnotherProcessClaim(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v4", "current-url.txt")
	if err := Claim(path, "http://127.0.0.1:1000"); err != nil {
		t.Fatal(err)
	}
	if err := ClearIfOwned(path, "http://127.0.0.1:2000"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("foreign claim was deleted: %v", err)
	}
	if err := ClearIfOwned(path, "http://127.0.0.1:1000"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("owned claim remains: %v", err)
	}
}

func TestCurrentLiveClaimDistinguishesLiveProtectedDesktopFromStaleFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(Identity{
			OK:       true,
			Version:  "3.0.0",
			BuildRef: "vexstream.mobile-bridge/v2",
		})
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "current-url.txt")
	if err := os.WriteFile(path, []byte(server.URL+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	claim, ok := CurrentLiveClaim(path)
	if !ok ||
		claim.URL != server.URL ||
		claim.Identity.Version != "3.0.0" ||
		claim.Identity.BuildRef != "vexstream.mobile-bridge/v2" {
		t.Fatalf("live protected claim drift: %+v ok=%v", claim, ok)
	}

	if err := os.WriteFile(path, []byte("http://127.0.0.1:1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if claim, ok := CurrentLiveClaim(path); ok {
		t.Fatalf("stale protected claim treated as live: %+v", claim)
	}
}
