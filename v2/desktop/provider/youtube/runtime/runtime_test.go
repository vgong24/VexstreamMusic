package runtimeenv

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMaterializeBridgeIsExactAndIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v4", "bridge", "media_bridge.py")
	data := []byte("print('v4')\n")

	first, err := MaterializeBridge(path, data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MaterializeBridge(path, data)
	if err != nil {
		t.Fatal(err)
	}
	if first == "" || first != second {
		t.Fatalf("bridge identity drift: first=%q second=%q", first, second)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("materialized bridge drift: %q", got)
	}
}

func TestBridgeEnvCarriesUtf8ProcessContract(t *testing.T) {
	env := BridgeEnv()
	foundUTF8 := false
	foundIO := false
	for _, item := range env {
		if item == "PYTHONUTF8=1" {
			foundUTF8 = true
		}
		if item == "PYTHONIOENCODING=utf-8" {
			foundIO = true
		}
	}
	if !foundUTF8 || !foundIO {
		t.Fatalf("UTF-8 bridge environment missing: %v", env)
	}
}

func TestResolveMissingRuntimeKeepsLocalPlaybackIndependent(t *testing.T) {
	status := Resolve(filepath.Join(t.TempDir(), "missing-runtime"))
	if status.Python != "" && !status.SearchReady {
		t.Fatalf("Python path/currentness contradiction: %+v", status)
	}
	if status.ImportReady && (!status.SearchReady || !status.FFmpegReady) {
		t.Fatalf("Import readiness widened beyond dependencies: %+v", status)
	}
}
