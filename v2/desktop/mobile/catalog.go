package mobile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library/filesystem"
	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
)

// trackProjection is the sanitized phone-facing track contract.
// Filesystem path/source identity stays inside Desktop.
type trackProjection struct {
	TrackRef             string   `json:"trackRef"`
	MediaSHA256          string   `json:"mediaSha256"`
	MediaByteLength      int64    `json:"mediaByteLength"`
	Title                string   `json:"title"`
	Artist               string   `json:"artist"`
	ArtworkURL           string   `json:"artworkUrl,omitempty"`
	ArtworkRef           string   `json:"artworkRef,omitempty"`
	Album                string   `json:"album,omitempty"`
	Genre                string   `json:"genre,omitempty"`
	Year                 string   `json:"year,omitempty"`
	DurationSeconds      float64  `json:"durationSeconds,omitempty"`
	PlaybackStartSeconds float64  `json:"playbackStartSeconds,omitempty"`
	PlaybackEndSeconds   *float64 `json:"playbackEndSeconds,omitempty"`
}

type playlistProjection struct {
	PlaylistRef string   `json:"playlistRef"`
	Name        string   `json:"name"`
	TrackRefs   []string `json:"trackRefs"`
}

type catalogProjection struct {
	HubRef          string               `json:"hubRef"`
	SourceVersion   string               `json:"sourceVersion"`
	CatalogRevision string               `json:"catalogRevision"`
	Tracks          []trackProjection    `json:"tracks"`
	Playlists       []playlistProjection `json:"playlists"`
}

type fingerprintCacheEntry struct {
	Size    int64
	ModTime int64
	SHA256  string
}

type artworkAvailabilityCacheEntry struct {
	Size      int64
	ModTime   int64
	Available bool
}

func (s *Service) mobileCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	snapshot := s.librarySnapshot()
	projected := make([]trackProjection, 0, len(snapshot.Tracks))
	for _, track := range snapshot.Tracks {
		path := snapshot.Paths[track.ID]
		if path == "" {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "library media path unavailable for " + track.ID,
			})
			return
		}

		sha, size, err := s.mediaFingerprint(path)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "library media unavailable for " + track.ID,
			})
			return
		}

		end := track.PlaybackEnd
		if end != nil {
			copied := *end
			end = &copied
		}

		artworkURL :=
			projectArtworkURL(
				track.ThumbnailURL,
			)
		artworkRef := ""
		if artworkURL == "" {
			embeddedArtwork, err :=
				s.embeddedArtworkAvailable(
					path,
				)
			if err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{
					"error": "library artwork metadata unavailable for " + track.ID,
				})
				return
			}
			artworkRef =
				projectArtworkRef(
					track.ID,
					embeddedArtwork,
				)
		}

		projected = append(projected, trackProjection{
			TrackRef:             track.ID,
			MediaSHA256:          sha,
			MediaByteLength:      size,
			Title:                track.Title,
			Artist:               track.Artist,
			ArtworkURL:           artworkURL,
			ArtworkRef:           artworkRef,
			Album:                track.Album,
			Genre:                track.Genre,
			Year:                 track.Year,
			DurationSeconds:      track.Duration,
			PlaybackStartSeconds: track.PlaybackStart,
			PlaybackEndSeconds:   end,
		})
	}

	sort.Slice(projected, func(i, j int) bool {
		return projected[i].TrackRef < projected[j].TrackRef
	})

	availableTrackRefs := make(map[string]bool, len(projected))
	for _, track := range projected {
		availableTrackRefs[track.TrackRef] = true
	}
	projectedPlaylists := projectPlaylists(
		s.playlistSnapshot(),
		availableTrackRefs,
	)

	revisionBytes, err := json.Marshal(struct {
		Tracks    []trackProjection    `json:"tracks"`
		Playlists []playlistProjection `json:"playlists"`
	}{
		Tracks:    projected,
		Playlists: projectedPlaylists,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "catalog projection failed",
		})
		return
	}
	revision := sha256.Sum256(revisionBytes)

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, catalogProjection{
		HubRef:          "vexstream.desktop.local",
		SourceVersion:   s.version,
		CatalogRevision: hex.EncodeToString(revision[:]),
		Tracks:          projected,
		Playlists:       projectedPlaylists,
	})
}

func (s *Service) mobileMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "GET or HEAD required",
		})
		return
	}

	trackRef := strings.TrimPrefix(r.URL.Path, "/v1/media/")
	if trackRef == "" || strings.ContainsAny(trackRef, "/\\") {
		http.NotFound(w, r)
		return
	}

	path := s.librarySnapshot().Paths[trackRef]
	if path == "" {
		http.NotFound(w, r)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || !stat.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(
		w,
		r,
		filepath.Base(path),
		stat.ModTime(),
		file,
	)
}

func (s *Service) mobileArtwork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(
			w,
			http.StatusMethodNotAllowed,
			map[string]string{
				"error": "GET or HEAD required",
			},
		)
		return
	}

	trackRef :=
		strings.TrimPrefix(
			r.URL.Path,
			"/v1/artwork/",
		)
	if trackRef == "" ||
		strings.ContainsAny(
			trackRef,
			"/\\",
		) {
		http.NotFound(
			w,
			r,
		)
		return
	}

	snapshot :=
		s.librarySnapshot()
	path :=
		snapshot.Paths[
			trackRef
		]
	if path == "" {
		http.NotFound(
			w,
			r,
		)
		return
	}

	mime, artwork, ok :=
		filesystem.ReadEmbeddedArtwork(
			path,
		)
	if !ok {
		http.NotFound(
			w,
			r,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		mime,
	)
	w.Header().Set(
		"Content-Length",
		fmt.Sprintf(
			"%d",
			len(artwork),
		),
	)
	w.Header().Set(
		"Cache-Control",
		"private, no-store",
	)
	w.Header().Set(
		"X-Content-Type-Options",
		"nosniff",
	)
	w.WriteHeader(
		http.StatusOK,
	)
	if r.Method == http.MethodHead {
		return
	}
	_, _ =
		w.Write(
			artwork,
		)
}

func projectArtworkRef(
	trackRef string,
	embeddedArtwork bool,
) string {
	if !embeddedArtwork {
		return ""
	}
	return trackRef
}

func projectArtworkURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return ""
	}

	return parsed.String()
}

func projectPlaylists(
	rows []playlists.Playlist,
	availableTrackRefs map[string]bool,
) []playlistProjection {
	projected := make([]playlistProjection, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if row.ID == "" || name == "" {
			continue
		}

		refs := make([]string, 0, len(row.TrackRefs))
		for _, ref := range row.TrackRefs {
			if availableTrackRefs[ref] {
				refs = append(refs, ref)
			}
		}

		projected = append(projected, playlistProjection{
			PlaylistRef: row.ID,
			Name:        name,
			TrackRefs:   refs,
		})
	}

	sort.Slice(projected, func(i, j int) bool {
		if projected[i].Name == projected[j].Name {
			return projected[i].PlaylistRef < projected[j].PlaylistRef
		}
		return strings.ToLower(projected[i].Name) <
			strings.ToLower(projected[j].Name)
	})
	return projected
}


func (s *Service) embeddedArtworkAvailable(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if !stat.Mode().IsRegular() {
		return false, errors.New("artwork media path is not a regular file")
	}

	cacheKey := filepath.Clean(path)
	modTime := stat.ModTime().UnixNano()

	s.artworkAvailabilityMu.Lock()
	cached, ok := s.artworkAvailabilityCache[cacheKey]
	s.artworkAvailabilityMu.Unlock()
	if ok &&
		cached.Size == stat.Size() &&
		cached.ModTime == modTime {
		return cached.Available, nil
	}

	available :=
		filesystem.HasEmbeddedArtwork(
			path,
		)

	s.artworkAvailabilityMu.Lock()
	s.artworkAvailabilityCache[cacheKey] =
		artworkAvailabilityCacheEntry{
			Size:      stat.Size(),
			ModTime:   modTime,
			Available: available,
		}
	s.artworkAvailabilityMu.Unlock()

	return available, nil
}

func (s *Service) mediaFingerprint(path string) (string, int64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	if !stat.Mode().IsRegular() {
		return "", 0, errors.New("media path is not a regular file")
	}

	cacheKey := filepath.Clean(path)
	modTime := stat.ModTime().UnixNano()

	s.fingerprintMu.Lock()
	cached, ok := s.fingerprintCache[cacheKey]
	s.fingerprintMu.Unlock()
	if ok && cached.Size == stat.Size() && cached.ModTime == modTime {
		return cached.SHA256, cached.Size, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", 0, err
	}
	value := hex.EncodeToString(digest.Sum(nil))

	s.fingerprintMu.Lock()
	s.fingerprintCache[cacheKey] = fingerprintCacheEntry{
		Size:    stat.Size(),
		ModTime: modTime,
		SHA256:  value,
	}
	s.fingerprintMu.Unlock()

	return value, stat.Size(), nil
}
