// Package filesystem realizes VexStream library discovery from desktop files and optional
// _vexmedia provenance metadata. The parent library package owns semantic track/catalog meaning.
package filesystem

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/catalog"
)

type catalogMeta struct {
	ProviderID   string
	SourceURL    string
	Channel      string
	Description  string
	ThumbnailURL string
	IndexedAt    string
	ArtistBasis  string
	Duration     float64
	Tags         []string
	Categories   []string
}

func TrackID(path string) string {
	h := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(path))))
	return hex.EncodeToString(h[:12])
}

func ReadTrack(path, source string) library.Track {
	catalog := loadCatalog(source)
	return readTrack(path, source, catalog)
}

func readTrack(path, source string, catalog map[string]catalogMeta) library.Track {
	title, artist, album, genre, year := readID3(path)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	t := library.Track{
		ID:       TrackID(path),
		Path:     path,
		Filename: filepath.Base(path),
		Title:    title,
		Artist:   artist,
		Album:    album,
		Genre:    genre,
		Year:     year,
		Source:   source,
	}
	if c, ok := catalog[filepath.Base(path)]; ok {
		t.Duration = c.Duration
		t.ProviderID = c.ProviderID
		t.SourceURL = c.SourceURL
		t.Channel = c.Channel
		t.ArtistBasis = c.ArtistBasis
		if t.ArtistBasis == "" && c.SourceURL != "" && c.Channel != "" && strings.EqualFold(strings.TrimSpace(t.Artist), strings.TrimSpace(c.Channel)) {
			t.ArtistBasis = "legacy.youtube.channel.unverified"
		}
		t.Tags = append([]string(nil), c.Tags...)
		t.Categories = append([]string(nil), c.Categories...)
		t.Description = c.Description
		t.ThumbnailURL = c.ThumbnailURL
		t.IndexedAt = c.IndexedAt
	}
	return t
}

func loadCatalog(root string) map[string]catalogMeta {
	path := filepath.Join(root, "_vexmedia", "library.jsonl")
	rows, _ := catalog.ReadRows(path)

	out := map[string]catalogMeta{}
	for _, row := range rows {
		filename, _ := row["filename"].(string)
		providerID, _ := row["providerId"].(string)
		metadata, _ := row["metadata"].(map[string]any)
		item := catalogMeta{ProviderID: providerID, IndexedAt: stringValue(row["indexedAt"])}
		if metadata != nil {
			item.SourceURL = stringValue(metadata["sourceUrl"])
			item.Channel = stringValue(metadata["channel"])
			item.Description = stringValue(metadata["description"])
			item.Duration = numberValue(metadata["duration"])
			item.Tags = stringList(metadata["tags"])
			item.Categories = stringList(metadata["categories"])
			item.ThumbnailURL = stringValue(metadata["thumbnailUrl"])
			item.ArtistBasis = stringValue(metadata["artistBasis"])
		}
		if filename != "" {
			out[filename] = item
		}
	}
	return out
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func numberValue(value any) float64 {
	if number, ok := value.(float64); ok {
		return number
	}
	return 0
}

func stringList(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, value := range values {
		if text, ok := value.(string); ok && text != "" {
			out = append(out, text)
		}
	}
	return out
}

func readID3(path string) (title, artist, album, genre, year string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	header := make([]byte, 10)
	if _, err = io.ReadFull(f, header); err == nil && string(header[:3]) == "ID3" {
		version := header[3]
		size := synchsafe(header[6:10])
		if size > 0 && size < 16*1024*1024 {
			tag := make([]byte, size)
			if _, err = io.ReadFull(f, tag); err == nil {
				for pos := 0; pos+10 <= len(tag); {
					id := string(tag[pos : pos+4])
					if strings.Trim(id, "\x00 ") == "" {
						break
					}
					var frameSize int
					if version == 4 {
						frameSize = synchsafe(tag[pos+4 : pos+8])
					} else {
						frameSize = int(binary.BigEndian.Uint32(tag[pos+4 : pos+8]))
					}
					if frameSize <= 0 || pos+10+frameSize > len(tag) {
						break
					}
					text := decodeText(tag[pos+10 : pos+10+frameSize])
					switch id {
					case "TIT2":
						title = text
					case "TPE1":
						artist = text
					case "TALB":
						album = text
					case "TCON":
						genre = text
					case "TDRC", "TYER":
						if year == "" {
							year = text
						}
					}
					pos += 10 + frameSize
				}
			}
		}
	}
	if title == "" && artist == "" && album == "" {
		stat, _ := f.Stat()
		if stat != nil && stat.Size() >= 128 {
			block := make([]byte, 128)
			if _, err = f.ReadAt(block, stat.Size()-128); err == nil && string(block[:3]) == "TAG" {
				title = trimLatin(block[3:33])
				artist = trimLatin(block[33:63])
				album = trimLatin(block[63:93])
				year = trimLatin(block[93:97])
			}
		}
	}
	return
}

func synchsafe(value []byte) int {
	if len(value) < 4 {
		return 0
	}
	return int(value[0]&0x7f)<<21 | int(value[1]&0x7f)<<14 | int(value[2]&0x7f)<<7 | int(value[3]&0x7f)
}

func decodeText(value []byte) string {
	if len(value) < 2 {
		return ""
	}
	encoding := value[0]
	data := value[1:]
	switch encoding {
	case 0:
		return strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", ""))
	case 3:
		return strings.TrimSpace(strings.TrimRight(string(data), "\x00"))
	case 1:
		if len(data) >= 2 && data[0] == 0xff && data[1] == 0xfe {
			return decodeUTF16(data[2:], binary.LittleEndian)
		}
		if len(data) >= 2 && data[0] == 0xfe && data[1] == 0xff {
			return decodeUTF16(data[2:], binary.BigEndian)
		}
		return decodeUTF16(data, binary.LittleEndian)
	case 2:
		return decodeUTF16(data, binary.BigEndian)
	default:
		return ""
	}
}

func decodeUTF16(value []byte, order binary.ByteOrder) string {
	var units []uint16
	for i := 0; i+1 < len(value); i += 2 {
		unit := order.Uint16(value[i : i+2])
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	return strings.TrimSpace(string(utf16.Decode(units)))
}

func trimLatin(value []byte) string {
	return strings.TrimSpace(string(bytes.TrimRight(value, "\x00 ")))
}
