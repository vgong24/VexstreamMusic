package trash

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library/catalog"
)

type TrackStateCleaner interface {
	ForgetTrackState([]string) error
}

type LibraryCleaner struct {
	State TrackStateCleaner
}

func (c LibraryCleaner) CleanPurged(items []Item) error {
	if c.State == nil {
		return fmt.Errorf("Library track-state cleaner unavailable")
	}
	refs := make([]string, 0, len(items))
	bySource := map[string]map[string]bool{}
	for _, item := range items {
		if ref := strings.TrimSpace(item.Track.ID); ref != "" {
			refs = append(refs, ref)
		}
		root := strings.TrimSpace(item.Track.Source)
		filename := strings.TrimSpace(item.Track.Filename)
		if root == "" || filename == "" {
			continue
		}
		if bySource[root] == nil {
			bySource[root] = map[string]bool{}
		}
		bySource[root][filename] = true
	}
	if err := c.State.ForgetTrackState(refs); err != nil {
		return err
	}

	for root, filenames := range bySource {
		path := filepath.Join(root, "_vexmedia", "library.jsonl")
		rows, err := catalog.ReadRows(path)
		if err != nil {
			return fmt.Errorf("read purged source catalog %s: %w", path, err)
		}
		kept := make([]map[string]any, 0, len(rows))
		changed := false
		for _, row := range rows {
			filename, _ := row["filename"].(string)
			if filenames[strings.TrimSpace(filename)] {
				changed = true
				continue
			}
			kept = append(kept, row)
		}
		if changed {
			if err := catalog.WriteRowsAtomic(path, kept); err != nil {
				return fmt.Errorf("write purged source catalog %s: %w", path, err)
			}
		}
	}
	return nil
}
