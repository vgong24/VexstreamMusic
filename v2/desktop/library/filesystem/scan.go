package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

type ScanInput struct {
	Sources   []string
	Previous  map[string]library.CacheEntry
	FirstSeen map[string]string
}

type ScanResult struct {
	Entries   map[string]library.CacheEntry
	FirstSeen map[string]string
}

func Scan(input ScanInput) (ScanResult, error) {
	previous := cloneEntries(input.Previous)
	firstSeen := cloneStrings(input.FirstSeen)
	next := map[string]library.CacheEntry{}

	for _, root := range input.Sources {
		stat, err := os.Stat(root)
		if err != nil || !stat.IsDir() {
			continue
		}
		stamp := CatalogStamp(root)
		catalog := loadCatalog(root)
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if entry.IsDir() {
				switch strings.ToLower(entry.Name()) {
				case ".git", "node_modules", ".venv", "__pycache__":
					return filepath.SkipDir
				default:
					return nil
				}
			}
			if strings.ToLower(filepath.Ext(path)) != ".mp3" {
				return nil
			}

			absolute, err := filepath.Abs(path)
			if err != nil {
				return nil
			}
			fileStat, err := os.Stat(absolute)
			if err != nil {
				return nil
			}
			key := CacheKey(absolute)
			cached, reusable := previous[key]
			var track library.Track
			if reusable && cached.Size == fileStat.Size() && cached.ModTime == fileStat.ModTime().UnixNano() && cached.CatalogStamp == stamp {
				track = cached.Track
			} else {
				track = readTrack(absolute, root, catalog)
			}
			track.AddedAt = ensureAddedAt(firstSeen, track)
			next[key] = library.CacheEntry{
				Path:         absolute,
				Size:         fileStat.Size(),
				ModTime:      fileStat.ModTime().UnixNano(),
				CatalogStamp: stamp,
				Track:        track,
			}
			return nil
		})
		if err != nil {
			return ScanResult{}, err
		}
	}

	return ScanResult{Entries: next, FirstSeen: firstSeen}, nil
}

func ensureAddedAt(firstSeen map[string]string, track library.Track) string {
	if seen := firstSeen[track.ID]; seen != "" {
		return seen
	}
	seed := strings.TrimSpace(track.IndexedAt)
	if seed == "" {
		seed = time.Now().UTC().Format(time.RFC3339Nano)
	}
	firstSeen[track.ID] = seed
	return seed
}

func cloneEntries(input map[string]library.CacheEntry) map[string]library.CacheEntry {
	out := make(map[string]library.CacheEntry, len(input))
	for key, entry := range input {
		out[key] = entry
	}
	return out
}

func cloneStrings(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
