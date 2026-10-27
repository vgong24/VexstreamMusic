package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const defaultMobileDevPort = 18765

type mobileTrackProjection struct {
	TrackRef             string   `json:"trackRef"`
	MediaSHA256          string   `json:"mediaSha256"`
	MediaByteLength      int64    `json:"mediaByteLength"`
	Title                string   `json:"title"`
	Artist               string   `json:"artist"`
	ArtworkURL           string   `json:"artworkUrl,omitempty"`
	Album                string   `json:"album,omitempty"`
	Genre                string   `json:"genre,omitempty"`
	Year                 string   `json:"year,omitempty"`
	DurationSeconds      float64  `json:"durationSeconds,omitempty"`
	PlaybackStartSeconds float64  `json:"playbackStartSeconds,omitempty"`
	PlaybackEndSeconds   *float64 `json:"playbackEndSeconds,omitempty"`
}

type mobilePlaylistProjection struct {
	PlaylistRef string   `json:"playlistRef"`
	Name        string   `json:"name"`
	TrackRefs   []string `json:"trackRefs"`
}

type mobileCatalogProjection struct {
	HubRef          string                     `json:"hubRef"`
	SourceVersion   string                     `json:"sourceVersion"`
	CatalogRevision string                     `json:"catalogRevision"`
	Tracks          []mobileTrackProjection    `json:"tracks"`
	Playlists       []mobilePlaylistProjection `json:"playlists"`
}

type mobileMediaFingerprintCacheEntry struct {
	Size    int64
	ModTime int64
	SHA256  string
}

var mobileMediaFingerprintCache = struct {
	sync.Mutex
	Entries map[string]mobileMediaFingerprintCacheEntry
}{
	Entries: map[string]mobileMediaFingerprintCacheEntry{},
}

/**
 * startMobileDevBridge is deliberately a development transport, not the permanent Home gateway.
 *
 * It binds only to loopback. A physical Android device may reach it through an explicitly selected
 * development transport such as adb reverse. This surface grants no LAN reachability, pairing,
 * authentication, capability lease, or remote Home authority.
 */
func (a *App) startMobileDevBridge() error {
	port := defaultMobileDevPort
	if raw := strings.TrimSpace(os.Getenv("VEXSTREAM_MOBILE_DEV_PORT")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 65535 {
			return fmt.Errorf("invalid VEXSTREAM_MOBILE_DEV_PORT %q", raw)
		}
		port = parsed
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", a.mobileHealth)
	mux.HandleFunc("/v1/catalog", a.mobileCatalog)
	mux.HandleFunc("/v1/media/", a.mobileMedia)

	go func() {
		_ = http.Serve(ln, mux)
	}()

	return nil
}

func (a *App) mobileHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	jsonOut(w, http.StatusOK, map[string]any{
		"ok":            true,
		"bridge":        "vexstream-mobile-dev/v1",
		"sourceVersion": version,
		"loopbackOnly":  true,
		"authorizes":    false,
		"pairsDevice":   false,
	})
}

func (a *App) mobileCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	tracks, paths := a.mobileLibrarySnapshot()
	projected := make([]mobileTrackProjection, 0, len(tracks))
	for _, track := range tracks {
		path := paths[track.ID]
		if path == "" {
			jsonOut(w, http.StatusServiceUnavailable, map[string]string{
				"error": "library media path unavailable for " + track.ID,
			})
			return
		}

		sha, size, err := mobileMediaFingerprint(path)
		if err != nil {
			jsonOut(w, http.StatusServiceUnavailable, map[string]string{
				"error": "library media unavailable for " + track.ID,
			})
			return
		}

		end := track.PlaybackEnd
		if end != nil {
			copied := *end
			end = &copied
		}

		projected = append(projected, mobileTrackProjection{
			TrackRef:             track.ID,
			MediaSHA256:          sha,
			MediaByteLength:      size,
			Title:                track.Title,
			Artist:               track.Artist,
			ArtworkURL:           projectMobileArtworkURL(track.ThumbnailURL),
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
	projectedPlaylists := mobileProjectPlaylists(
		a.mobilePlaylistSnapshot(),
		availableTrackRefs,
	)

	revisionBytes, err := json.Marshal(struct {
		Tracks    []mobileTrackProjection    `json:"tracks"`
		Playlists []mobilePlaylistProjection `json:"playlists"`
	}{
		Tracks:    projected,
		Playlists: projectedPlaylists,
	})
	if err != nil {
		jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "catalog projection failed"})
		return
	}
	revision := sha256.Sum256(revisionBytes)

	w.Header().Set("Cache-Control", "no-store")
	jsonOut(w, http.StatusOK, mobileCatalogProjection{
		HubRef:          "vexstream.desktop.local",
		SourceVersion:   version,
		CatalogRevision: hex.EncodeToString(revision[:]),
		Tracks:          projected,
		Playlists:       projectedPlaylists,
	})
}

func (a *App) mobileMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or HEAD required"})
		return
	}

	trackRef := strings.TrimPrefix(r.URL.Path, "/v1/media/")
	if trackRef == "" || strings.Contains(trackRef, "/") {
		http.NotFound(w, r)
		return
	}

	a.mu.RLock()
	path := a.paths[trackRef]
	a.mu.RUnlock()
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
	http.ServeContent(w, r, filepath.Base(path), stat.ModTime(), file)
}

func (a *App) mobilePlaylistSnapshot() []Playlist {
	a.mu.RLock()
	defer a.mu.RUnlock()

	rows := make([]Playlist, len(a.playlists))
	for i, playlist := range a.playlists {
		rows[i] = playlist
		rows[i].TrackIDs = append([]string{}, playlist.TrackIDs...)
	}
	return rows
}

func mobileProjectPlaylists(
	playlists []Playlist,
	availableTrackRefs map[string]bool,
) []mobilePlaylistProjection {
	projected := make([]mobilePlaylistProjection, 0, len(playlists))
	for _, playlist := range playlists {
		name := strings.TrimSpace(playlist.Name)
		if playlist.ID == "" || name == "" {
			continue
		}

		refs := make([]string, 0, len(playlist.TrackIDs))
		for _, ref := range playlist.TrackIDs {
			if availableTrackRefs[ref] {
				refs = append(refs, ref)
			}
		}

		projected = append(projected, mobilePlaylistProjection{
			PlaylistRef: playlist.ID,
			Name:        name,
			TrackRefs:   refs,
		})
	}

	sort.Slice(projected, func(i, j int) bool {
		if projected[i].Name == projected[j].Name {
			return projected[i].PlaylistRef < projected[j].PlaylistRef
		}
		return strings.ToLower(projected[i].Name) < strings.ToLower(projected[j].Name)
	})
	return projected
}

func (a *App) mobileLibrarySnapshot() ([]Track, map[string]string) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	tracks := make([]Track, len(a.tracks))
	copy(tracks, a.tracks)

	paths := make(map[string]string, len(a.paths))
	for ref, path := range a.paths {
		paths[ref] = path
	}

	return tracks, paths
}

func mobileMediaFingerprint(path string) (string, int64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	if !stat.Mode().IsRegular() {
		return "", 0, fmt.Errorf("media path is not a regular file")
	}

	cacheKey := filepath.Clean(path)
	modTime := stat.ModTime().UnixNano()

	mobileMediaFingerprintCache.Lock()
	cached, ok := mobileMediaFingerprintCache.Entries[cacheKey]
	mobileMediaFingerprintCache.Unlock()
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
	sha := hex.EncodeToString(digest.Sum(nil))

	mobileMediaFingerprintCache.Lock()
	mobileMediaFingerprintCache.Entries[cacheKey] = mobileMediaFingerprintCacheEntry{
		Size:    stat.Size(),
		ModTime: modTime,
		SHA256:  sha,
	}
	mobileMediaFingerprintCache.Unlock()

	return sha, stat.Size(), nil
}


func projectMobileArtworkURL(value string) string {
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
