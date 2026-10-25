package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

type cacheFile struct {
	SchemaVersion string               `json:"schemaVersion"`
	GeneratedAt   string               `json:"generatedAt"`
	Entries       []library.CacheEntry `json:"entries"`
}

type CacheLoadResult struct {
	Entries         map[string]library.CacheEntry
	DiscardedLegacy bool
}

func CacheKey(path string) string {
	cleaned := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		cleaned = strings.ToLower(cleaned)
	}
	return cleaned
}

func LoadCache(path string) (CacheLoadResult, error) {
	result := CacheLoadResult{Entries: map[string]library.CacheEntry{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return result, err
	}

	var file cacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return result, err
	}
	if file.SchemaVersion == "vexstream.library-index/v1" {
		result.DiscardedLegacy = true
		return result, nil
	}
	if file.SchemaVersion != "" && file.SchemaVersion != library.CacheSchemaV2 {
		return result, fmt.Errorf("unsupported cache schema: %s", file.SchemaVersion)
	}
	for _, entry := range file.Entries {
		if entry.Path != "" {
			result.Entries[CacheKey(entry.Path)] = entry
		}
	}
	return result, nil
}

func SaveCache(path string, entries map[string]library.CacheEntry) error {
	rows := make([]library.CacheEntry, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, entry)
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].Path) < strings.ToLower(rows[j].Path)
	})
	file := cacheFile{
		SchemaVersion: library.CacheSchemaV2,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Entries:       rows,
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func CatalogStamp(root string) int64 {
	stat, err := os.Stat(filepath.Join(root, "_vexmedia", "library.jsonl"))
	if err != nil {
		return 0
	}
	return stat.ModTime().UnixNano()
}
