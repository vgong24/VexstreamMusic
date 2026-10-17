package youtube

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func bridgeWithRunner(
	t *testing.T,
	output []byte,
	runErr error,
) (*Bridge, *[][]string) {
	t.Helper()
	seen := [][]string{}
	bridge := New("python3", "/tmp/media_bridge.py", []string{"A=B"})
	bridge.run = func(_ context.Context, command []string) ([]byte, error) {
		seen = append(seen, append([]string(nil), command...))
		return append([]byte(nil), output...), runErr
	}
	return bridge, &seen
}

func TestSearchBuildsBridgeCommandAndDecodesResults(t *testing.T) {
	bridge, seen := bridgeWithRunner(
		t,
		[]byte(`{"results":[{"id":"abc","title":"Song","channel":"Creator","url":"https://example","duration":123}]}`),
		nil,
	)

	rows, err := bridge.Search(context.Background(), "  dream pop  ", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ProviderID != "abc" || rows[0].Title != "Song" {
		t.Fatalf("search decode drift: %+v", rows)
	}
	want := []string{
		"python3",
		"/tmp/media_bridge.py",
		"search",
		"dream pop",
		"--limit",
		"5",
	}
	if len(*seen) != 1 || !reflect.DeepEqual((*seen)[0], want) {
		t.Fatalf("command drift: got=%v want=%v", *seen, want)
	}
}

func TestInspectDecodesProviderMetadataWithoutOwningImportLifecycle(t *testing.T) {
	bridge, seen := bridgeWithRunner(
		t,
		[]byte(`{
		  "id":"abc",
		  "url":"https://example",
		  "title":"Song",
		  "channel":"Creator",
		  "duration":123,
		  "artist":"Artist",
		  "album":"Album",
		  "genre":"Dream Pop",
		  "sections":[{"index":1,"start":0,"end":30,"title":"Part 1","selected":true}],
		  "hasSections":true,
		  "sourceQuality":{"codec":"opus","bitrateKbps":160}
		}`),
		nil,
	)

	got, err := bridge.Inspect(context.Background(), " https://example ")
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderID != "abc" || got.Artist != "Artist" ||
		len(got.Sections) != 1 || !got.HasSections ||
		got.SourceQuality.Codec != "opus" {
		t.Fatalf("inspect decode drift: %+v", got)
	}
	want := []string{
		"python3",
		"/tmp/media_bridge.py",
		"inspect",
		"https://example",
	}
	if len(*seen) != 1 || !reflect.DeepEqual((*seen)[0], want) {
		t.Fatalf("inspect command drift: got=%v want=%v", *seen, want)
	}
}

func TestProviderProblemEnvelopeSurvivesBridgeFailure(t *testing.T) {
	payload := []byte(
		"progress noise\n" +
			`{"type":"error","problem":{"schemaVersion":"vexstream.provider-problem/v1","provider":"youtube","stage":"search","code":"RATE_LIMIT","title":"Provider busy","message":"Retry later","retryable":true}}` +
			"\n",
	)
	bridge, _ := bridgeWithRunner(t, payload, errors.New("exit status 2"))

	_, err := bridge.Search(context.Background(), "song", 8)
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("provider failure type drift: %T %v", err, err)
	}
	if failure.Problem.Code != "RATE_LIMIT" ||
		failure.Problem.Stage != "search" ||
		!failure.Problem.Retryable {
		t.Fatalf("provider problem drift: %+v", failure.Problem)
	}
}

func TestMalformedSuccessOutputBecomesProviderFailureInsteadOfInventingData(t *testing.T) {
	bridge, _ := bridgeWithRunner(t, []byte("not-json"), nil)

	_, err := bridge.Search(context.Background(), "song", 8)
	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("malformed output error drift: %T %v", err, err)
	}
	if failure.Problem.Code != "YOUTUBE_PROVIDER_ERROR" ||
		failure.Problem.Technical != "not-json" {
		t.Fatalf("fallback problem drift: %+v", failure.Problem)
	}
}

func TestRuntimeAndRequiredInputsFailClosed(t *testing.T) {
	if _, err := New("", "", nil).Search(context.Background(), "song", 8); err == nil {
		t.Fatal("missing runtime accepted")
	}
	bridge := New("python3", "media_bridge.py", nil)
	if _, err := bridge.Search(context.Background(), " ", 8); err == nil {
		t.Fatal("blank query accepted")
	}
	if _, err := bridge.Inspect(context.Background(), " "); err == nil {
		t.Fatal("blank inspect URL accepted")
	}
}
