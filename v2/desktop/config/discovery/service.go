// Package discovery finds candidate music folders without admitting them into desktop config.
//
// Candidate discovery is temporary evidence. Only the config owner may include or ignore a folder.
package discovery

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Candidate struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	MP3Count int    `json:"mp3Count"`
}

type Result struct {
	Candidates  []Candidate `json:"candidates"`
	ScannedDirs int         `json:"scannedDirs"`
}

func DefaultRoots(home string, deep bool) []string {
	home = strings.TrimSpace(home)
	if home == "" {
		return nil
	}
	if deep {
		if info, err := os.Stat(home); err == nil && info.IsDir() {
			return []string{filepath.Clean(home)}
		}
		return nil
	}
	roots := []string{}
	for _, name := range []string{
		"Music",
		"Media",
		"Desktop",
		"Downloads",
		"Documents",
	} {
		path := filepath.Join(home, name)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			roots = append(roots, path)
		}
	}
	return roots
}

func Scan(
	roots []string,
	included []string,
	ignored []string,
	deep bool,
) Result {
	found := map[string]int{}
	scanned := 0
	maxDirs := 80000
	maxDepth := 6
	if deep {
		maxDepth = 8
	}

	for _, root := range normalizePaths(roots) {
		_ = filepath.WalkDir(root, func(
			path string,
			entry fs.DirEntry,
			err error,
		) error {
			if err != nil {
				return nil
			}
			if scanned >= maxDirs {
				return filepath.SkipAll
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if entry.IsDir() {
				if path != root {
					scanned++
				}
				if depth(root, path) > maxDepth {
					return filepath.SkipDir
				}
				if path != root && ignoredDirectoryName(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}

			if strings.EqualFold(filepath.Ext(path), ".mp3") {
				found[filepath.Dir(path)]++
			}
			return nil
		})
	}

	includedSet := pathSet(included)
	ignoredSet := pathSet(ignored)
	candidates := make([]Candidate, 0, len(found))
	for path, count := range found {
		key := pathKey(path)
		if includedSet[key] || ignoredSet[key] {
			continue
		}
		candidates = append(candidates, Candidate{
			Name:     filepath.Base(path),
			Path:     filepath.Clean(path),
			MP3Count: count,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].MP3Count == candidates[j].MP3Count {
			return strings.ToLower(candidates[i].Path) <
				strings.ToLower(candidates[j].Path)
		}
		return candidates[i].MP3Count > candidates[j].MP3Count
	})
	return Result{
		Candidates:  candidates,
		ScannedDirs: scanned,
	}
}

func depth(root, path string) int {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." {
		return 0
	}
	count := 0
	for _, part := range strings.Split(
		filepath.Clean(relative),
		string(os.PathSeparator),
	) {
		if part != "" && part != "." {
			count++
		}
	}
	return count
}

func ignoredDirectoryName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "appdata",
		".git",
		"node_modules",
		".venv",
		"__pycache__",
		"windows",
		"program files",
		"program files (x86)":
		return true
	default:
		return false
	}
}

func normalizePaths(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		value = filepath.Clean(value)
		key := pathKey(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func pathSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range normalizePaths(values) {
		out[pathKey(value)] = true
	}
	return out
}

func pathKey(path string) string {
	return strings.ToLower(filepath.Clean(path))
}
