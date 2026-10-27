// Package catalog owns the desktop _vexmedia/library.jsonl storage format.
//
// Library discovery reads this provenance and Import acquisition writes it. The format owner does not
// decide Library membership, duplicate semantics, provider execution, or import-job lifecycle.
package catalog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func ReadRows(path string) ([]map[string]any, error) {
	rows := []map[string]any{}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return rows, nil
	}
	if err != nil {
		return rows, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var row map[string]any
		if json.Unmarshal(scanner.Bytes(), &row) != nil {
			// Preserve V3 tolerance: malformed provenance rows do not erase other readable rows.
			continue
		}
		rows = append(rows, row)
	}
	return rows, scanner.Err()
}

func WriteRowsAtomic(path string, rows []map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	file, err := os.Create(temporary)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			_ = file.Close()
			_ = os.Remove(temporary)
			return err
		}
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

// UpsertProviderRows preserves V3's provider/source identity rule.
//
// A row replaces an existing row only when provider + non-empty providerId match. Rows without a
// providerId remain independent provenance records and append.
func UpsertProviderRows(rows, entries []map[string]any) []map[string]any {
	out := cloneRows(rows)
	for _, entry := range entries {
		entry = cloneRow(entry)
		provider := stringValue(entry["provider"])
		providerID := stringValue(entry["providerId"])
		replaced := false
		if providerID != "" {
			for index, row := range out {
				if stringValue(row["provider"]) == provider &&
					stringValue(row["providerId"]) == providerID {
					out[index] = entry
					replaced = true
					break
				}
			}
		}
		if !replaced {
			out = append(out, entry)
		}
	}
	return out
}

func cloneRows(rows []map[string]any) []map[string]any {
	out := make([]map[string]any, len(rows))
	for index, row := range rows {
		out[index] = cloneRow(row)
	}
	return out
}

func cloneRow(row map[string]any) map[string]any {
	if row == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(row))
	for key, value := range row {
		out[key] = value
	}
	return out
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
