package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOpenReadsExactV3ConfigShape(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.json")
	if err := os.WriteFile(
		path,
		[]byte(`{"sources":["/music"],"ignored":["/skip"]}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	service, err := Open(path, filepath.Join(root, "backup", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	got := service.Snapshot()
	if !reflect.DeepEqual(got.Sources, []string{"/music"}) ||
		!reflect.DeepEqual(got.Ignored, []string{"/skip"}) {
		t.Fatalf("V3 config compatibility drift: %+v", got)
	}
}

func TestV4WriteRemainsReadableByProtectedV3Shape(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "music")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "config.json")
	backup := filepath.Join(root, "backup", "config.json")
	initial, err := json.Marshal(State{
		Sources: []string{},
		Ignored: []string{source},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatal(err)
	}
	service, err := Open(path, backup)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddSource(source); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var v3 struct {
		Sources []string `json:"sources"`
		Ignored []string `json:"ignored"`
	}
	if err := json.Unmarshal(data, &v3); err != nil {
		t.Fatal(err)
	}
	if len(v3.Sources) != 1 || v3.Sources[0] != source ||
		len(v3.Ignored) != 0 {
		t.Fatalf("new V4 write is not V3-readable: %s", data)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != 2 || raw["sources"] == nil || raw["ignored"] == nil {
		t.Fatalf("V4 widened shared config schema: %s", data)
	}

	backupData, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if string(backupData) != string(initial) {
		t.Fatalf("pre-write V3 bytes not preserved in backup: %s", backupData)
	}
}

func TestAddRemoveAndIgnoreUseCaseInsensitivePathIdentity(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "Music")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	service, err := Open(
		filepath.Join(root, "config.json"),
		filepath.Join(root, "backup.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.Ignore(source); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddSource(source); err != nil {
		t.Fatal(err)
	}
	if !service.IsSource(source) ||
		len(service.Snapshot().Ignored) != 0 {
		t.Fatalf("AddSource did not remove ignored identity: %+v", service.Snapshot())
	}
	if _, err := service.RemoveSource(source); err != nil {
		t.Fatal(err)
	}
	if service.IsSource(source) {
		t.Fatal("RemoveSource retained source identity")
	}
}

func TestRejectedAddDoesNotCreateConfig(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.json")
	service, err := Open(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddSource(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing source accepted")
	}
	if _, err := service.AddSource(" "); err == nil {
		t.Fatal("blank source accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("rejected source created config: %v", err)
	}
}

func TestRestoreBackupIsExplicit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.json")
	backup := filepath.Join(root, "backup.json")
	if err := os.WriteFile(
		path,
		[]byte(`{"sources":["/current"],"ignored":[]}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		backup,
		[]byte(`{"sources":["/backup"],"ignored":["/skip"]}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	service, err := Open(path, backup)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RestoreBackup(); err != nil {
		t.Fatal(err)
	}
	got := service.Snapshot()
	if !reflect.DeepEqual(got.Sources, []string{"/backup"}) ||
		!reflect.DeepEqual(got.Ignored, []string{"/skip"}) {
		t.Fatalf("backup restore drift: %+v", got)
	}
}

func TestSharedConfigRejectsUnknownNewFieldBeforeBackupPropagation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.json")
	source := filepath.Join(root, "music")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		path,
		[]byte(`{"sources":[],"ignored":[],"futureOnly":true}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	service, err := Open(path, filepath.Join(root, "backup.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddSource(source); err == nil {
		t.Fatal("unknown shared config field was silently overwritten")
	}
