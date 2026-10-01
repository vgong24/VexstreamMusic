package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

//go:embed ui/index.html media_bridge.py
var assets embed.FS

const version = "2.2.0"

type Config struct {
	Sources []string `json:"sources"`
	Ignored []string `json:"ignored"`
}
type Candidate struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	MP3Count int    `json:"mp3Count"`
}
type Track struct {
	ID                string   `json:"id"`
	Path              string   `json:"path"`
	Filename          string   `json:"filename"`
	Title             string   `json:"title"`
	Artist            string   `json:"artist"`
	Album             string   `json:"album"`
	Genre             string   `json:"genre"`
	Year              string   `json:"year"`
	Duration          float64  `json:"duration"`
	Source            string   `json:"source"`
	ProviderID        string   `json:"providerId,omitempty"`
	SourceURL         string   `json:"sourceUrl,omitempty"`
	Channel           string   `json:"channel,omitempty"`
	ArtistBasis       string   `json:"artistBasis,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Categories        []string `json:"categories,omitempty"`
	Description       string   `json:"description,omitempty"`
	ThumbnailURL      string   `json:"thumbnailUrl,omitempty"`
	IndexedAt         string   `json:"indexedAt,omitempty"`
	AddedAt           string   `json:"addedAt"`
	SourceTitle       string   `json:"sourceTitle,omitempty"`
	SourceArtist      string   `json:"sourceArtist,omitempty"`
	SourceAlbum       string   `json:"sourceAlbum,omitempty"`
	SourceGenre       string   `json:"sourceGenre,omitempty"`
	OverlayUpdatedAt  string   `json:"overlayUpdatedAt,omitempty"`
	PlaybackStart     float64  `json:"playbackStart,omitempty"`
	PlaybackEnd       *float64 `json:"playbackEnd,omitempty"`
	PlaybackUpdatedAt string   `json:"playbackUpdatedAt,omitempty"`
}
type DiscoverySeed struct {
	Kind       string  `json:"kind,omitempty"`
	TrackID    string  `json:"trackId,omitempty"`
	ProviderID string  `json:"providerId,omitempty"`
	URL        string  `json:"url,omitempty"`
	Title      string  `json:"title,omitempty"`
	Artist     string  `json:"artist,omitempty"`
	Album      string  `json:"album,omitempty"`
	Genre      string  `json:"genre,omitempty"`
	Year       string  `json:"year,omitempty"`
	Channel    string  `json:"channel,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
}
type DiscoveryQuery struct {
	Axis  string `json:"axis"`
	Label string `json:"label"`
	Query string `json:"query"`
}
type DiscoveryLibraryMatch struct {
	State      string `json:"state"`
	TrackID    string `json:"trackId,omitempty"`
	Title      string `json:"title,omitempty"`
	Artist     string `json:"artist,omitempty"`
	Confidence string `json:"confidence,omitempty"`
	Relation   string `json:"relation,omitempty"`
	Score      int    `json:"score,omitempty"`
}
type DiscoveryCandidate struct {
	ProviderID   string                `json:"providerId,omitempty"`
	Title        string                `json:"title"`
	Channel      string                `json:"channel,omitempty"`
	URL          string                `json:"url,omitempty"`
	Thumbnail    string                `json:"thumbnail,omitempty"`
	Duration     float64               `json:"duration,omitempty"`
	Axis         string                `json:"axis,omitempty"`
	AxisLabel    string                `json:"axisLabel,omitempty"`
	Query        string                `json:"query,omitempty"`
	LibraryMatch DiscoveryLibraryMatch `json:"libraryMatch"`
}
type TrackOverlay struct {
	Title     *string `json:"title,omitempty"`
	Artist    *string `json:"artist,omitempty"`
	Album     *string `json:"album,omitempty"`
	Genre     *string `json:"genre,omitempty"`
	UpdatedAt string  `json:"updatedAt"`
}
type PlaybackWindow struct {
	Start     float64  `json:"start,omitempty"`
	End       *float64 `json:"end,omitempty"`
	UpdatedAt string   `json:"updatedAt"`
}
type LibraryCacheEntry struct {
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	ModTime      int64  `json:"modTime"`
	CatalogStamp int64  `json:"catalogStamp"`
	Track        Track  `json:"track"`
}
type LibraryCache struct {
	SchemaVersion string              `json:"schemaVersion"`
	GeneratedAt   string              `json:"generatedAt"`
	Entries       []LibraryCacheEntry `json:"entries"`
}
type SystemIssue struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	Technical  string `json:"technical,omitempty"`
	Repairable bool   `json:"repairable"`
}
type RepairAction struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Effect         string `json:"effect"`
	RequiresReload bool   `json:"requiresReload"`
}

type LiveInstance struct {
	PID           int      `json:"pid"`
	Version       string   `json:"version"`
	URL           string   `json:"url"`
	StartedAt     string   `json:"startedAt,omitempty"`
	TrackCount    int      `json:"trackCount"`
	SourceCount   int      `json:"sourceCount"`
	Sources       []string `json:"sources,omitempty"`
	ActiveImports int      `json:"activeImports"`
	Current       bool     `json:"current"`
}
type livePort struct {
	PID       int    `json:"pid"`
	Port      int    `json:"port"`
	StartedAt string `json:"startedAt"`
}
type liveLibraryResponse struct {
	Tracks  []Track `json:"tracks"`
	Sources []struct {
		Path       string `json:"path"`
		Name       string `json:"name"`
		TrackCount int    `json:"trackCount"`
	} `json:"sources"`
}

type DiagnosticEvent struct {
	ObservedAt string         `json:"observedAt"`
	Kind       string         `json:"kind"`
	Message    string         `json:"message"`
	Data       map[string]any `json:"data,omitempty"`
}

type RetireCandidate struct {
	PID              int    `json:"pid"`
	Version          string `json:"version"`
	URL              string `json:"url"`
	TrackCount       int    `json:"trackCount"`
	ActiveImports    int    `json:"activeImports"`
	UniqueTrackCount int    `json:"uniqueTrackCount"`
	Eligible         bool   `json:"eligible"`
	Reason           string `json:"reason"`
}

type LibraryResponse struct {
	Tracks     []Track          `json:"tracks"`
	Sources    []map[string]any `json:"sources"`
	Candidates []Candidate      `json:"candidates"`
	Ignored    []string         `json:"ignored"`
}
type ImportJob struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Creator      string         `json:"creator,omitempty"`
	Thumbnail    string         `json:"thumbnail,omitempty"`
	SourceURL    string         `json:"sourceUrl,omitempty"`
	Destination  string         `json:"destination,omitempty"`
	Status       string         `json:"status"`
	Stage        string         `json:"stage"`
	Progress     int            `json:"progress"`
	Detail       string         `json:"detail"`
	StartedAt    string         `json:"startedAt"`
	CompletedAt  string         `json:"completedAt,omitempty"`
	OutputCount  int            `json:"outputCount,omitempty"`
	TrackIDs     []string       `json:"trackIds,omitempty"`
	OwnerPID     int            `json:"ownerPid,omitempty"`
	OwnerVersion string         `json:"ownerVersion,omitempty"`
	OwnerURL     string         `json:"ownerUrl,omitempty"`
	Remote       bool           `json:"remote,omitempty"`
	Log          []string       `json:"log,omitempty"`
	Problem      map[string]any `json:"problem,omitempty"`
	StagingDir   string         `json:"-"`
}
type StagedOutput struct {
	RelativePath string         `json:"relativePath"`
	Filename     string         `json:"filename"`
	CatalogEntry map[string]any `json:"catalogEntry"`
}
type StagedManifest struct {
	SchemaVersion string         `json:"schemaVersion"`
	Destination   string         `json:"destination"`
	Outputs       []StagedOutput `json:"outputs"`
}
type TrashItem struct {
	ID           string `json:"id"`
	Track        Track  `json:"track"`
	OriginalPath string `json:"originalPath"`
	TrashPath    string `json:"trashPath"`
	TrashedAt    string `json:"trashedAt"`
}
type DeletedHistoryItem struct {
	ID           string `json:"id"`
	Track        Track  `json:"track"`
	OriginalPath string `json:"originalPath"`
	DeletedAt    string `json:"deletedAt"`
}
type Playlist struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	TrackIDs  []string `json:"trackIds"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}
type ArtistSuggestion struct {
	Artist              string `json:"artist"`
	Confidence          string `json:"confidence"`
	Basis               string `json:"basis"`
	MatchingTrackCount  int    `json:"matchingTrackCount"`
	ChannelCorroborated bool   `json:"channelCorroborated"`
}
type DuplicateMatch struct {
	TrackID           string            `json:"trackId"`
	Title             string            `json:"title"`
	Artist            string            `json:"artist"`
	Album             string            `json:"album"`
	Duration          float64           `json:"duration"`
	ThumbnailURL      string            `json:"thumbnailUrl,omitempty"`
	ProviderID        string            `json:"providerId,omitempty"`
	SourceURL         string            `json:"sourceUrl,omitempty"`
	Score             int               `json:"score"`
	Confidence        string            `json:"confidence"`
	Relation          string            `json:"relation"`
	Reasons           []string          `json:"reasons"`
	LocalQuality      AudioQuality      `json:"localQuality"`
	QualityComparison QualityComparison `json:"qualityComparison"`
}
type AudioQuality struct {
	Codec        string  `json:"codec,omitempty"`
	Container    string  `json:"container,omitempty"`
	BitrateKbps  float64 `json:"bitrateKbps,omitempty"`
	SampleRateHz int     `json:"sampleRateHz,omitempty"`
	Channels     int     `json:"channels,omitempty"`
	Basis        string  `json:"basis,omitempty"`
}
type QualityComparison struct {
	State   string `json:"state"`
	Summary string `json:"summary"`
}
type DuplicateItem struct {
	Kind          string           `json:"kind"`
	Index         int              `json:"index,omitempty"`
	Title         string           `json:"title"`
	Duration      float64          `json:"duration"`
	SourceQuality AudioQuality     `json:"sourceQuality"`
	Matches       []DuplicateMatch `json:"matches"`
}
type App struct {
	mu                                                                                                                                           sync.RWMutex
	config                                                                                                                                       Config
	tracks                                                                                                                                       []Track
	paths                                                                                                                                        map[string]string
	candidates                                                                                                                                   []Candidate
	configPath, trackStatePath, cachePath, overlayPath, playbackWindowsPath, trashStatePath, deletedHistoryPath, playlistsPath, home, currentURL string
	jobs                                                                                                                                         map[string]*ImportJob
	jobCommands                                                                                                                                  map[string]*exec.Cmd
	firstSeen                                                                                                                                    map[string]string
	overlays                                                                                                                                     map[string]TrackOverlay
	playbackWindows                                                                                                                              map[string]PlaybackWindow
	trash                                                                                                                                        []TrashItem
	deletedHistory                                                                                                                               []DeletedHistoryItem
	playlists                                                                                                                                    []Playlist
	importSlots                                                                                                                                  chan struct{}
	cache                                                                                                                                        map[string]LibraryCacheEntry
	configLoadError, trackStateLoadError, cacheLoadError                                                                                         string
	scanRunning                                                                                                                                  bool
	scanReason, scanLastError, scanStartedAt, scanCompletedAt                                                                                    string
	priorInstanceURL, priorInstanceVersion                                                                                                       string
	recoveryProjection                                                                                                                           bool
	recoveryFromURL, recoveryFromVersion                                                                                                         string
	recoverySources                                                                                                                              []string
	liveInstances                                                                                                                                []LiveInstance
	lifecycleScannedAt                                                                                                                           time.Time
	diagnosticListening                                                                                                                          bool
	diagnosticSessionID, diagnosticStartedAt                                                                                                     string
	diagnosticEvents                                                                                                                             []DiagnosticEvent
}

func main() {
	home, _ := os.UserHomeDir()
	base := appDataBase()
	a := &App{home: home, paths: map[string]string{}, configPath: filepath.Join(base, "config.json"), trackStatePath: filepath.Join(base, "track-first-seen.json"), cachePath: filepath.Join(base, "library-index.json"), overlayPath: filepath.Join(base, "library-overlays.json"), playbackWindowsPath: filepath.Join(base, "playback-windows.json"), trashStatePath: filepath.Join(base, "trash.json"), deletedHistoryPath: filepath.Join(base, "deleted-history.json"), playlistsPath: filepath.Join(base, "playlists.json"), jobs: map[string]*ImportJob{}, jobCommands: map[string]*exec.Cmd{}, firstSeen: map[string]string{}, overlays: map[string]TrackOverlay{}, playbackWindows: map[string]PlaybackWindow{}, trash: []TrashItem{}, deletedHistory: []DeletedHistoryItem{}, playlists: []Playlist{}, importSlots: make(chan struct{}, 2), cache: map[string]LibraryCacheEntry{}, diagnosticListening: true, diagnosticSessionID: randID(), diagnosticStartedAt: time.Now().UTC().Format(time.RFC3339Nano), diagnosticEvents: make([]DiagnosticEvent, 0)}
	a.loadConfig()
	a.loadTrackState()
	a.loadOverlays()
	a.loadPlaybackWindows()
	a.loadTrashState()
	a.loadDeletedHistory()
	a.loadPlaylists()
	a.loadLibraryCache()
	a.loadStagedJobs()
	a.materializeBridge()
	state := filepath.Join(appDataBase(), "current-url.txt")
	if b, e := os.ReadFile(state); e == nil {
		u := strings.TrimSpace(string(b))
		if u != "" {
			if v := runningVersion(u); v == version {
				openBrowser(u)
				return
			} else if v != "" {
				a.priorInstanceURL = u
				a.priorInstanceVersion = v
			}
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveUI)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		jsonOut(w, 200, map[string]any{"ok": true, "version": version})
	})
	mux.HandleFunc("/api/library", a.library)
	mux.HandleFunc("/api/library/rescan", a.rescanAPI)
	mux.HandleFunc("/api/library/overlay", a.libraryOverlay)
	mux.HandleFunc("/api/library/playback-window", a.libraryPlaybackWindow)
	mux.HandleFunc("/api/library/trash", a.libraryTrash)
	mux.HandleFunc("/api/library/trash/restore", a.libraryTrashRestore)
	mux.HandleFunc("/api/library/trash/purge", a.libraryTrashPurge)
	mux.HandleFunc("/api/library/reveal", a.libraryReveal)
	mux.HandleFunc("/api/playlists", a.playlistsAPI)
	mux.HandleFunc("/api/playlists/create", a.playlistCreate)
	mux.HandleFunc("/api/playlists/add", a.playlistAdd)
	mux.HandleFunc("/api/playlists/order", a.playlistOrder)
	mux.HandleFunc("/api/playlists/remove", a.playlistRemove)
	mux.HandleFunc("/api/playlists/rename", a.playlistRename)
	mux.HandleFunc("/api/playlists/delete", a.playlistDelete)
	mux.HandleFunc("/api/source/choose", a.chooseSource)
	mux.HandleFunc("/api/source/add", a.addSource)
	mux.HandleFunc("/api/source/remove", a.removeSource)
	mux.HandleFunc("/api/discover", a.discover)
	mux.HandleFunc("/api/candidate/ignore", a.ignoreCandidate)
	mux.HandleFunc("/api/candidate/unignore", a.unignoreCandidate)
	mux.HandleFunc("/api/ignored/clear", a.clearIgnored)
	mux.HandleFunc("/media/", a.media)
	mux.HandleFunc("/api/import/status", a.importStatus)
	mux.HandleFunc("/api/import/setup", a.importSetup)
	mux.HandleFunc("/api/import/search", a.importSearch)
	mux.HandleFunc("/api/discovery/radio", a.discoveryRadio)
	mux.HandleFunc("/api/import/inspect", a.importInspect)
	mux.HandleFunc("/api/import/duplicates", a.importDuplicates)
	mux.HandleFunc("/api/import/start", a.importStart)
	mux.HandleFunc("/api/import/cancel", a.importCancel)
	mux.HandleFunc("/api/import/commit", a.importCommit)
	mux.HandleFunc("/api/import/discard", a.importDiscard)
	mux.HandleFunc("/api/import/job", a.importJob)
	mux.HandleFunc("/api/import/jobs", a.importJobs)
	mux.HandleFunc("/api/diagnostics", a.diagnostics)
	mux.HandleFunc("/api/diagnostics/session", a.diagnosticSession)
	mux.HandleFunc("/api/diagnostics/event", a.diagnosticClientEvent)
	mux.HandleFunc("/api/diagnostics/listening", a.diagnosticListeningAPI)
	mux.HandleFunc("/api/diagnostics/clear", a.diagnosticClear)
	mux.HandleFunc("/api/diagnostics/run", a.diagnosticRun)
	mux.HandleFunc("/api/diagnostics/download", a.diagnosticDownload)
	mux.HandleFunc("/api/system/preflight", a.systemPreflight)
	mux.HandleFunc("/api/system/repair-plan", a.systemRepairPlan)
	mux.HandleFunc("/api/system/repair", a.systemRepair)
	mux.HandleFunc("/api/system/sessions/retire-plan", a.sessionRetirePlan)
	mux.HandleFunc("/api/system/sessions/retire", a.sessionRetire)
	port := 0
	if s := os.Getenv("VEXSTREAM_PORT"); s != "" {
		if p, e := strconv.Atoi(s); e == nil {
			port = p
		}
	}
	ln, e := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if e != nil {
		panic(e)
	}
	actual := ln.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d/", actual)
	a.currentURL = url
	a.mu.Lock()
	for _, j := range a.jobs {
		if !j.Remote {
			j.OwnerPID = os.Getpid()
			j.OwnerVersion = version
			j.OwnerURL = url
		}
	}
	a.mu.Unlock()
	os.MkdirAll(appDataBase(), 0755)
	os.WriteFile(state, []byte(url+"\n"), 0644)
	defer func() {
		if b, e := os.ReadFile(state); e == nil && strings.TrimSpace(string(b)) == url {
			os.Remove(state)
		}
	}()
	a.startReconcile("startup")
	go func() { time.Sleep(250 * time.Millisecond); openBrowser(url) }()
	http.Serve(ln, mux)
}
func appDataBase() string {
	if runtime.GOOS == "windows" {
		if p := os.Getenv("LOCALAPPDATA"); p != "" {
			return filepath.Join(p, "VexStreamMusic")
		}
	}
	if p, e := os.UserConfigDir(); e == nil {
		return filepath.Join(p, "VexStreamMusic")
	}
	return filepath.Join(os.TempDir(), "VexStreamMusic")
}
func serveUI(w http.ResponseWriter, r *http.Request) {
	b, _ := fs.ReadFile(assets, "ui/index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b)
}
func jsonOut(w http.ResponseWriter, s int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	json.NewEncoder(w).Encode(v)
}
func decode(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }
func runningVersion(url string) string {
	c := http.Client{Timeout: 700 * time.Millisecond}
	resp, e := c.Get(strings.TrimRight(url, "/") + "/health")
	if e != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	var h struct {
		OK      bool   `json:"ok"`
		Version string `json:"version"`
	}
	if json.NewDecoder(resp.Body).Decode(&h) != nil || !h.OK {
		return ""
	}
	return h.Version
}
func copyFile(src, dst string) error {
	b, e := os.ReadFile(src)
	if e != nil {
		return e
	}
	os.MkdirAll(filepath.Dir(dst), 0755)
	tmp := dst + ".tmp"
	if e = os.WriteFile(tmp, b, 0644); e != nil {
		return e
	}
	return os.Rename(tmp, dst)
}
func (a *App) configBackupPath() string {
	return filepath.Join(appDataBase(), "backups", "config.latest.json")
}
func (a *App) loadConfig() {
	a.config = Config{}
	b, e := os.ReadFile(a.configPath)
	if os.IsNotExist(e) {
		return
	}
	if e != nil {
		a.configLoadError = e.Error()
		return
	}
	if e = json.Unmarshal(b, &a.config); e != nil {
		a.configLoadError = e.Error()
		a.config = Config{}
		return
	}
	a.configLoadError = ""
	os.MkdirAll(filepath.Dir(a.configBackupPath()), 0755)
	os.WriteFile(a.configBackupPath(), b, 0644)
}
func (a *App) saveConfig() error {
	os.MkdirAll(filepath.Dir(a.configPath), 0755)
	if b, e := os.ReadFile(a.configPath); e == nil {
		os.MkdirAll(filepath.Dir(a.configBackupPath()), 0755)
		os.WriteFile(a.configBackupPath(), b, 0644)
	}
	tmp := a.configPath + ".tmp"
	b, _ := json.MarshalIndent(a.config, "", "  ")
	if e := os.WriteFile(tmp, b, 0644); e != nil {
		return e
	}
	return os.Rename(tmp, a.configPath)
}
func (a *App) loadTrackState() {
	a.firstSeen = map[string]string{}
	b, e := os.ReadFile(a.trackStatePath)
	if os.IsNotExist(e) {
		return
	}
	if e != nil {
		a.trackStateLoadError = e.Error()
		return
	}
	if e = json.Unmarshal(b, &a.firstSeen); e != nil {
		a.trackStateLoadError = e.Error()
		a.firstSeen = map[string]string{}
		return
	}
	a.trackStateLoadError = ""
}
func (a *App) saveTrackState() error {
	os.MkdirAll(filepath.Dir(a.trackStatePath), 0755)
	tmp := a.trackStatePath + ".tmp"
	b, _ := json.MarshalIndent(a.firstSeen, "", "  ")
	if e := os.WriteFile(tmp, b, 0644); e != nil {
		return e
	}
	return os.Rename(tmp, a.trackStatePath)
}
func saveJSONAtomic(path string, v any) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	tmp := path + ".tmp"
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	if e = os.WriteFile(tmp, b, 0644); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}
func (a *App) loadOverlays() {
	a.overlays = map[string]TrackOverlay{}
	b, e := os.ReadFile(a.overlayPath)
	if os.IsNotExist(e) {
		return
	}
	if e != nil {
		return
	}
	json.Unmarshal(b, &a.overlays)
}
func (a *App) saveOverlaysLocked() error { return saveJSONAtomic(a.overlayPath, a.overlays) }
func (a *App) loadTrashState() {
	a.trash = []TrashItem{}
	b, e := os.ReadFile(a.trashStatePath)
	if os.IsNotExist(e) {
		return
	}
	if e == nil {
		json.Unmarshal(b, &a.trash)
	}
}
func (a *App) saveTrashLocked() error { return saveJSONAtomic(a.trashStatePath, a.trash) }
func (a *App) loadDeletedHistory() {
	a.deletedHistory = []DeletedHistoryItem{}
	b, e := os.ReadFile(a.deletedHistoryPath)
	if os.IsNotExist(e) {
		return
	}
	if e == nil {
		json.Unmarshal(b, &a.deletedHistory)
	}
}
func (a *App) saveDeletedHistoryLocked() error {
	return saveJSONAtomic(a.deletedHistoryPath, a.deletedHistory)
}
func (a *App) loadPlaylists() {
	a.playlists = []Playlist{}
	b, e := os.ReadFile(a.playlistsPath)
	if os.IsNotExist(e) {
		return
	}
	if e == nil {
		_ = json.Unmarshal(b, &a.playlists)
	}
}
func (a *App) savePlaylistsLocked() error { return saveJSONAtomic(a.playlistsPath, a.playlists) }

func (a *App) loadPlaybackWindows() {
	a.playbackWindows = map[string]PlaybackWindow{}
	b, e := os.ReadFile(a.playbackWindowsPath)
	if os.IsNotExist(e) {
		return
	}
	if e == nil {
		_ = json.Unmarshal(b, &a.playbackWindows)
	}
}
func (a *App) savePlaybackWindowsLocked() error {
	return saveJSONAtomic(a.playbackWindowsPath, a.playbackWindows)
}
func renderTrackWithPlaybackWindow(t Track, windows map[string]PlaybackWindow) Track {
	t.PlaybackStart = 0
	t.PlaybackEnd = nil
	t.PlaybackUpdatedAt = ""
	if w, ok := windows[t.ID]; ok {
		if w.Start > 0 {
			t.PlaybackStart = w.Start
		}
		if w.End != nil {
			v := *w.End
			t.PlaybackEnd = &v
		}
		t.PlaybackUpdatedAt = w.UpdatedAt
	}
	return t
}

func renderTrackWithOverlay(t Track, overlays map[string]TrackOverlay) Track {
	t.SourceTitle = t.Title
	t.SourceArtist = t.Artist
	t.SourceAlbum = t.Album
	t.SourceGenre = t.Genre
	// A YouTube uploader/channel is provenance, not automatically a musical artist.
	// Preserve the raw value as sourceArtist but do not present it as trusted artist metadata.
	if t.ArtistBasis == "youtube.channel" || t.ArtistBasis == "legacy.youtube.channel.unverified" {
		t.Artist = ""
	}
	if o, ok := overlays[t.ID]; ok {
		if o.Title != nil {
			t.Title = *o.Title
		}
		if o.Artist != nil {
			t.Artist = *o.Artist
		}
		if o.Album != nil {
			t.Album = *o.Album
		}
		if o.Genre != nil {
			t.Genre = *o.Genre
		}
		t.OverlayUpdatedAt = o.UpdatedAt
	}
	return t
}
func (a *App) ensureAddedAtMap(firstSeen map[string]string, id string, track Track) string {
	if seen := firstSeen[id]; seen != "" {
		return seen
	}
	seed := strings.TrimSpace(track.IndexedAt)
	if seed == "" {
		seed = time.Now().UTC().Format(time.RFC3339Nano)
	}
	firstSeen[id] = seed
	return seed
}
func cacheKey(path string) string {
	p := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}
func sourceAllowed(sources []string, source string) bool {
	for _, s := range sources {
		if strings.EqualFold(filepath.Clean(s), filepath.Clean(source)) {
			return true
		}
	}
	return false
}
func (a *App) applyCacheLocked(useAll bool) {
	tracks := make([]Track, 0, len(a.cache))
	paths := map[string]string{}
	for _, entry := range a.cache {
		t := entry.Track
		if !useAll && !sourceAllowed(a.config.Sources, t.Source) {
			continue
		}
		t = renderTrackWithOverlay(t, a.overlays)
		t = renderTrackWithPlaybackWindow(t, a.playbackWindows)
		tracks = append(tracks, t)
		paths[t.ID] = entry.Path
	}
	sort.Slice(tracks, func(i, j int) bool {
		return strings.ToLower(tracks[i].Artist+" "+tracks[i].Album+" "+tracks[i].Title) < strings.ToLower(tracks[j].Artist+" "+tracks[j].Album+" "+tracks[j].Title)
	})
	a.tracks = tracks
	a.paths = paths
}
func (a *App) loadLibraryCache() {
	a.cache = map[string]LibraryCacheEntry{}
	b, e := os.ReadFile(a.cachePath)
	if os.IsNotExist(e) {
		return
	}
	if e != nil {
		a.cacheLoadError = e.Error()
		return
	}
	var c LibraryCache
	if e = json.Unmarshal(b, &c); e != nil {
		a.cacheLoadError = e.Error()
		return
	}
	if c.SchemaVersion == "vexstream.library-index/v1" {
		// v1 is disposable and lacked the source-artist basis needed to avoid
		// treating a YouTube channel/uploader as a musical artist. Rebuild it.
		a.cacheLoadError = ""
		return
	}
	if c.SchemaVersion != "" && c.SchemaVersion != "vexstream.library-index/v2" {
		a.cacheLoadError = "unsupported cache schema: " + c.SchemaVersion
		return
	}
	for _, entry := range c.Entries {
		if entry.Path != "" {
			a.cache[cacheKey(entry.Path)] = entry
		}
	}
	a.cacheLoadError = ""
	a.applyCacheLocked(a.configLoadError != "")
}
func saveLibraryCacheFile(path string, entries map[string]LibraryCacheEntry) error {
	rows := make([]LibraryCacheEntry, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, e)
	}
	sort.Slice(rows, func(i, j int) bool { return strings.ToLower(rows[i].Path) < strings.ToLower(rows[j].Path) })
	c := LibraryCache{SchemaVersion: "vexstream.library-index/v2", GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Entries: rows}
	b, _ := json.MarshalIndent(c, "", "  ")
	os.MkdirAll(filepath.Dir(path), 0755)
	tmp := path + ".tmp"
	if e := os.WriteFile(tmp, b, 0644); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}
func catalogStamp(root string) int64 {
	if s, e := os.Stat(filepath.Join(root, "_vexmedia", "library.jsonl")); e == nil {
		return s.ModTime().UnixNano()
	}
	return 0
}
func (a *App) library(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	counts := map[string]int{}
	for _, t := range a.tracks {
		counts[t.Source]++
	}
	sourcePaths := append([]string{}, a.config.Sources...)
	recovered := false
	if len(sourcePaths) == 0 && a.recoveryProjection {
		sourcePaths = append(sourcePaths, a.recoverySources...)
		recovered = true
	}
	src := make([]map[string]any, 0, len(sourcePaths))
	for _, p := range sourcePaths {
		src = append(src, map[string]any{"path": p, "name": filepath.Base(p), "trackCount": counts[p], "recovered": recovered})
	}
	jsonOut(w, 200, map[string]any{"tracks": a.tracks, "sources": src, "candidates": a.candidates, "ignored": a.config.Ignored, "recoveryProjection": a.recoveryProjection, "scan": map[string]any{"running": a.scanRunning, "reason": a.scanReason, "startedAt": a.scanStartedAt, "completedAt": a.scanCompletedAt, "lastError": a.scanLastError}})
}
func (a *App) rescanAPI(w http.ResponseWriter, r *http.Request) {
	a.startReconcile("manual rescan")
	a.library(w, r)
}
func (a *App) startReconcile(reason string) {
	a.mu.Lock()
	if a.scanRunning {
		a.mu.Unlock()
		return
	}
	if a.configLoadError != "" {
		a.scanLastError = "Library scan held because config.json could not be read."
		a.mu.Unlock()
		a.recordDiagnostic("RECONCILE_HELD", "Library reconciliation held because config could not be read.", map[string]any{"reason": reason})
		return
	}
	a.scanRunning = true
	a.scanReason = reason
	a.scanLastError = ""
	a.scanStartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	a.mu.Unlock()
	a.recordDiagnostic("RECONCILE_STARTED", "Library reconciliation started.", map[string]any{"reason": reason})
	go func() {
		err := a.reconcileLibrary()
		a.mu.Lock()
		a.scanRunning = false
		a.scanCompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err != nil {
			a.scanLastError = err.Error()
		} else {
			a.scanLastError = ""
		}
		tracks := len(a.tracks)
		a.mu.Unlock()
		if err != nil {
			a.recordDiagnostic("RECONCILE_FAILED", "Library reconciliation failed.", map[string]any{"reason": reason, "error": err.Error()})
		} else {
			a.recordDiagnostic("RECONCILE_COMPLETED", "Library reconciliation completed.", map[string]any{"reason": reason, "tracks": tracks})
		}
	}()
}
func (a *App) reconcileLibrary() error {
	a.mu.RLock()
	sources := append([]string{}, a.config.Sources...)
	old := map[string]LibraryCacheEntry{}
	for k, v := range a.cache {
		old[k] = v
	}
	seenTimes := map[string]string{}
	for k, v := range a.firstSeen {
		seenTimes[k] = v
	}
	overlays := map[string]TrackOverlay{}
	for k, v := range a.overlays {
		overlays[k] = v
	}
	playbackWindows := map[string]PlaybackWindow{}
	for k, v := range a.playbackWindows {
		if v.End != nil {
			end := *v.End
			v.End = &end
		}
		playbackWindows[k] = v
	}
	a.mu.RUnlock()
	next := map[string]LibraryCacheEntry{}
	tracks := make([]Track, 0)
	paths := map[string]string{}
	for _, root := range sources {
		if s, e := os.Stat(root); e != nil || !s.IsDir() {
			continue
		}
		stamp := catalogStamp(root)
		catalog := loadCatalog(root)
		filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if d.IsDir() {
				b := strings.ToLower(d.Name())
				if b == ".git" || b == "node_modules" || b == ".venv" || b == "__pycache__" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.ToLower(filepath.Ext(p)) != ".mp3" {
				return nil
			}
			abs, _ := filepath.Abs(p)
			st, e := os.Stat(abs)
			if e != nil {
				return nil
			}
			key := cacheKey(abs)
			entry, ok := old[key]
			var t Track
			if ok && entry.Size == st.Size() && entry.ModTime == st.ModTime().UnixNano() && entry.CatalogStamp == stamp {
				t = entry.Track
			} else {
				t = readTrack(abs, root, catalog)
			}
			t.AddedAt = a.ensureAddedAtMap(seenTimes, t.ID, t)
			entry = LibraryCacheEntry{Path: abs, Size: st.Size(), ModTime: st.ModTime().UnixNano(), CatalogStamp: stamp, Track: t}
			next[key] = entry
			display := renderTrackWithOverlay(t, overlays)
			display = renderTrackWithPlaybackWindow(display, playbackWindows)
			tracks = append(tracks, display)
			paths[t.ID] = abs
			return nil
		})
	}
	sort.Slice(tracks, func(i, j int) bool {
		return strings.ToLower(tracks[i].Artist+" "+tracks[i].Album+" "+tracks[i].Title) < strings.ToLower(tracks[j].Artist+" "+tracks[j].Album+" "+tracks[j].Title)
	})
	if e := saveLibraryCacheFile(a.cachePath, next); e != nil {
		return e
	}
	a.mu.Lock()
	a.cache = next
	a.tracks = tracks
	a.paths = paths
	a.firstSeen = seenTimes
	a.cacheLoadError = ""
	a.mu.Unlock()
	a.saveTrackState()
	return nil
}
func chooseFolder() (string, error) {
	if runtime.GOOS == "windows" {
		ps := `Add-Type -AssemblyName System.Windows.Forms;$d=New-Object System.Windows.Forms.FolderBrowserDialog;$d.Description='Choose a music folder for VexStream';$d.ShowNewFolderButton=$false;if($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK){[Console]::Write($d.SelectedPath)}`
		b, e := exec.Command("powershell.exe", "-NoProfile", "-STA", "-Command", ps).Output()
		if e != nil {
			return "", e
		}
		p := strings.TrimSpace(string(b))
		if p == "" {
			return "", errors.New("folder selection cancelled")
		}
		return p, nil
	}
	if runtime.GOOS == "darwin" {
		b, e := exec.Command("/usr/bin/osascript", "-e", `POSIX path of (choose folder with prompt "Choose a music folder for VexStream")`).Output()
		if e != nil {
			return "", e
		}
		return strings.TrimSpace(string(b)), nil
	}
	return "", errors.New("folder picker unavailable")
}
func (a *App) chooseSource(w http.ResponseWriter, r *http.Request) {
	p, e := chooseFolder()
	if e != nil {
		jsonOut(w, 400, map[string]string{"error": e.Error()})
		return
	}
	jsonOut(w, 200, map[string]string{"path": p})
}
func (a *App) addSource(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Path string `json:"path"`
	}
	if e := decode(r, &q); e != nil {
		jsonOut(w, 400, map[string]string{"error": e.Error()})
		return
	}
	p, e := filepath.Abs(strings.TrimSpace(q.Path))
	if e != nil {
		jsonOut(w, 400, map[string]string{"error": e.Error()})
		return
	}
	st, e := os.Stat(p)
	if e != nil || !st.IsDir() {
		jsonOut(w, 400, map[string]string{"error": "folder does not exist"})
		return
	}
	a.mu.Lock()
	if !containsFold(a.config.Sources, p) {
		a.config.Sources = append(a.config.Sources, p)
	}
	a.config.Ignored = removeFold(a.config.Ignored, p)
	a.candidates = removeCandidate(a.candidates, p)
	a.saveConfig()
	a.mu.Unlock()
	a.startReconcile("source added")
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) removeSource(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Path string `json:"path"`
	}
	decode(r, &q)
	a.mu.Lock()
	a.config.Sources = removeFold(a.config.Sources, q.Path)
	a.saveConfig()
	a.applyCacheLocked(false)
	a.mu.Unlock()
	a.startReconcile("source removed")
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) ignoreCandidate(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Path string `json:"path"`
	}
	decode(r, &q)
	a.mu.Lock()
	if !containsFold(a.config.Ignored, q.Path) {
		a.config.Ignored = append(a.config.Ignored, q.Path)
	}
	a.candidates = removeCandidate(a.candidates, q.Path)
	a.saveConfig()
	a.mu.Unlock()
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) unignoreCandidate(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Path string `json:"path"`
	}
	decode(r, &q)
	a.mu.Lock()
	a.config.Ignored = removeFold(a.config.Ignored, q.Path)
	a.saveConfig()
	a.mu.Unlock()
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) clearIgnored(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	a.config.Ignored = nil
	a.saveConfig()
	a.mu.Unlock()
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) discover(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Deep bool `json:"deep"`
	}
	decode(r, &q)
	var roots []string
	if q.Deep {
		roots = []string{a.home}
	} else {
		for _, n := range []string{"Music", "Media", "Desktop", "Downloads", "Documents"} {
			p := filepath.Join(a.home, n)
			if st, e := os.Stat(p); e == nil && st.IsDir() {
				roots = append(roots, p)
			}
		}
	}
	a.mu.RLock()
	inc := append([]string{}, a.config.Sources...)
	ign := append([]string{}, a.config.Ignored...)
	a.mu.RUnlock()
	c, dirs := discoverFolders(roots, inc, ign, q.Deep)
	a.mu.Lock()
	a.candidates = c
	a.mu.Unlock()
	jsonOut(w, 200, map[string]any{"candidates": c, "scannedDirs": dirs})
}
func discoverFolders(roots, inc, ign []string, deep bool) ([]Candidate, int) {
	found := map[string]int{}
	dirs := 0
	maxDirs := 80000
	maxDepth := 6
	if deep {
		maxDepth = 8
	}
	for _, root := range roots {
		bd := pathDepth(root)
		filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return nil
			}
			if dirs >= maxDirs {
				return filepath.SkipAll
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if d.IsDir() {
				if p != root {
					dirs++
				}
				if pathDepth(p)-bd > maxDepth {
					return filepath.SkipDir
				}
				b := strings.ToLower(d.Name())
				if b == "appdata" || b == ".git" || b == "node_modules" || b == ".venv" || b == "__pycache__" || b == "windows" || b == "program files" || b == "program files (x86)" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.ToLower(filepath.Ext(p)) == ".mp3" {
				found[filepath.Dir(p)]++
			}
			return nil
		})
	}
	out := make([]Candidate, 0)
	for p, n := range found {
		if containsFold(inc, p) || containsFold(ign, p) {
			continue
		}
		out = append(out, Candidate{Name: filepath.Base(p), Path: p, MP3Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MP3Count == out[j].MP3Count {
			return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path)
		}
		return out[i].MP3Count > out[j].MP3Count
	})
	return out, dirs
}
func (a *App) media(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/media/")
	a.mu.RLock()
	p := a.paths[id]
	a.mu.RUnlock()
	if p == "" {
		http.NotFound(w, r)
		return
	}
	f, e := os.Open(p)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, filepath.Base(p), st.ModTime(), f)
}
func (a *App) reapplyVisibleOverlaysLocked() {
	for i, t := range a.tracks {
		raw := t
		if t.SourceTitle != "" {
			raw.Title = t.SourceTitle
		}
		raw.Artist = t.SourceArtist
		raw.Album = t.SourceAlbum
		raw.Genre = t.SourceGenre
		raw.SourceTitle = ""
		raw.SourceArtist = ""
		raw.SourceAlbum = ""
		raw.SourceGenre = ""
		raw.OverlayUpdatedAt = ""
		raw.PlaybackStart = 0
		raw.PlaybackEnd = nil
		raw.PlaybackUpdatedAt = ""
		display := renderTrackWithOverlay(raw, a.overlays)
		a.tracks[i] = renderTrackWithPlaybackWindow(display, a.playbackWindows)
	}
}
func (a *App) libraryOverlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonOut(w, 405, map[string]string{"error": "POST required"})
		return
	}
	var q struct {
		ID      string   `json:"id"`
		IDs     []string `json:"ids"`
		Restore bool     `json:"restore"`
		Title   *string  `json:"title"`
		Artist  *string  `json:"artist"`
		Album   *string  `json:"album"`
		Genre   *string  `json:"genre"`
	}
	if e := decode(r, &q); e != nil {
		jsonOut(w, 400, map[string]string{"error": e.Error()})
		return
	}
	targets := append([]string{}, q.IDs...)
	if strings.TrimSpace(q.ID) != "" {
		targets = append(targets, q.ID)
	}
	seen := map[string]bool{}
	cleanTargets := make([]string, 0, len(targets))
	for _, id := range targets {
		id = strings.TrimSpace(id)
		if id != "" && !seen[id] {
			seen[id] = true
			cleanTargets = append(cleanTargets, id)
		}
	}
	if len(cleanTargets) == 0 {
		jsonOut(w, 400, map[string]string{"error": "track id required"})
		return
	}
	if !q.Restore && q.Title == nil && q.Artist == nil && q.Album == nil && q.Genre == nil {
		jsonOut(w, 400, map[string]string{"error": "no display fields supplied"})
		return
	}
	a.mu.Lock()
	for _, id := range cleanTargets {
		if _, ok := a.paths[id]; !ok {
			a.mu.Unlock()
			jsonOut(w, 404, map[string]string{"error": "one or more tracks are no longer available"})
			return
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, id := range cleanTargets {
		if q.Restore {
			delete(a.overlays, id)
			continue
		}
		o := a.overlays[id]
		if q.Title != nil {
			o.Title = q.Title
		}
		if q.Artist != nil {
			o.Artist = q.Artist
		}
		if q.Album != nil {
			o.Album = q.Album
		}
		if q.Genre != nil {
			o.Genre = q.Genre
		}
		o.UpdatedAt = now
		a.overlays[id] = o
	}
	e := a.saveOverlaysLocked()
	a.reapplyVisibleOverlaysLocked()
	a.mu.Unlock()
	if e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	a.recordDiagnostic("LIBRARY_DISPLAY_OVERLAY_UPDATED", "User updated reversible library display metadata.", map[string]any{"trackCount": len(cleanTargets), "restored": q.Restore})
	jsonOut(w, 200, map[string]any{"ok": true, "updated": len(cleanTargets)})
}
func (a *App) libraryPlaybackWindow(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonOut(w, 405, map[string]string{"error": "POST required"})
		return
	}
	var q struct {
		ID      string   `json:"id"`
		Start   float64  `json:"start"`
		End     *float64 `json:"end"`
		Restore bool     `json:"restore"`
	}
	if e := decode(r, &q); e != nil {
		jsonOut(w, 400, map[string]string{"error": e.Error()})
		return
	}
	q.ID = strings.TrimSpace(q.ID)
	if q.ID == "" {
		jsonOut(w, 400, map[string]string{"error": "track id required"})
		return
	}
	a.mu.Lock()
	var current Track
	found := false
	for _, t := range a.tracks {
		if t.ID == q.ID {
			current = t
			found = true
			break
		}
	}
	if !found || a.paths[q.ID] == "" {
		a.mu.Unlock()
		jsonOut(w, 404, map[string]string{"error": "track is no longer available"})
		return
	}
	if q.Restore {
		delete(a.playbackWindows, q.ID)
		e := a.savePlaybackWindowsLocked()
		a.reapplyVisibleOverlaysLocked()
		a.mu.Unlock()
		if e != nil {
			jsonOut(w, 500, map[string]string{"error": e.Error()})
			return
		}
		a.recordDiagnostic("PLAYBACK_WINDOW_RESET", "User restored full-track playback boundaries.", map[string]any{"trackId": q.ID})
		jsonOut(w, 200, map[string]any{"ok": true, "restored": true})
		return
	}
	if q.Start < 0 {
		q.Start = 0
	}
	if q.Start < 0.001 {
		q.Start = 0
	}
	duration := current.Duration
	if duration > 0 && q.Start >= duration-0.02 {
		a.mu.Unlock()
		jsonOut(w, 400, map[string]string{"error": "playback start must be before the end of the track"})
		return
	}
	var end *float64
	if q.End != nil {
		v := *q.End
		if duration > 0 && v > duration {
			v = duration
		}
		if v <= q.Start+0.02 {
			a.mu.Unlock()
			jsonOut(w, 400, map[string]string{"error": "playback end must be after playback start"})
			return
		}
		if duration <= 0 || v < duration-0.02 {
			end = &v
		}
	}
	if q.Start == 0 && end == nil {
		delete(a.playbackWindows, q.ID)
	} else {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		a.playbackWindows[q.ID] = PlaybackWindow{Start: q.Start, End: end, UpdatedAt: now}
	}
	e := a.savePlaybackWindowsLocked()
	a.reapplyVisibleOverlaysLocked()
	result := a.playbackWindows[q.ID]
	a.mu.Unlock()
	if e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	a.recordDiagnostic("PLAYBACK_WINDOW_UPDATED", "User changed non-destructive playback crop boundaries.", map[string]any{"trackId": q.ID, "start": q.Start, "hasEnd": end != nil})
	jsonOut(w, 200, map[string]any{"ok": true, "window": result})
}
func moveFileSafely(src, dst string) error {
	if src == "" || dst == "" {
		return errors.New("missing source or destination")
	}
	if _, e := os.Stat(dst); e == nil {
		return errors.New("destination already exists")
	}
	if e := os.MkdirAll(filepath.Dir(dst), 0755); e != nil {
		return e
	}
	if e := os.Rename(src, dst); e == nil {
		return nil
	}
	in, e := os.Open(src)
	if e != nil {
		return e
	}
	defer in.Close()
	tmp := dst + ".partial"
	out, e := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if e != nil {
		return e
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	if e = os.Rename(tmp, dst); e != nil {
		os.Remove(tmp)
		return e
	}
	if e = os.Remove(src); e != nil {
		os.Remove(dst)
		return e
	}
	return nil
}
func (a *App) libraryTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		a.mu.RLock()
		trash := append([]TrashItem{}, a.trash...)
		history := append([]DeletedHistoryItem{}, a.deletedHistory...)
		a.mu.RUnlock()
		sort.Slice(trash, func(i, j int) bool { return trash[i].TrashedAt > trash[j].TrashedAt })
		sort.Slice(history, func(i, j int) bool { return history[i].DeletedAt > history[j].DeletedAt })
		jsonOut(w, 200, map[string]any{"trash": trash, "deletedHistory": history})
		return
	}
	if r.Method != "POST" {
		jsonOut(w, 405, map[string]string{"error": "GET or POST required"})
		return
	}
	var q struct {
		IDs     []string `json:"ids"`
		Confirm string   `json:"confirm"`
	}
	if e := decode(r, &q); e != nil || q.Confirm != "MOVE_TO_VEXSTREAM_TRASH" {
		jsonOut(w, 400, map[string]string{"error": "explicit trash confirmation missing"})
		return
	}
	if len(q.IDs) == 0 {
		jsonOut(w, 400, map[string]string{"error": "no tracks selected"})
		return
	}
	a.mu.RLock()
	selected := make([]struct {
		Track Track
		Path  string
	}, 0, len(q.IDs))
	for _, id := range q.IDs {
		p := a.paths[id]
		if p == "" {
			a.mu.RUnlock()
			jsonOut(w, 404, map[string]string{"error": "one or more selected tracks are no longer available"})
			return
		}
		var t Track
		for _, row := range a.tracks {
			if row.ID == id {
				t = row
				break
			}
		}
		selected = append(selected, struct {
			Track Track
			Path  string
		}{t, p})
	}
	a.mu.RUnlock()
	moved := make([]TrashItem, 0, len(selected))
	for _, x := range selected {
		id := randID()
		ext := filepath.Ext(x.Path)
		if ext == "" {
			ext = ".mp3"
		}
		tp := filepath.Join(appDataBase(), "trash", "files", id+ext)
		if e := moveFileSafely(x.Path, tp); e != nil {
			for i := len(moved) - 1; i >= 0; i-- {
				_ = moveFileSafely(moved[i].TrashPath, moved[i].OriginalPath)
			}
			jsonOut(w, 500, map[string]string{"error": "Could not move selected tracks to VexStream Trash: " + e.Error()})
			return
		}
		moved = append(moved, TrashItem{ID: id, Track: x.Track, OriginalPath: x.Path, TrashPath: tp, TrashedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	}
	a.mu.Lock()
	a.trash = append(a.trash, moved...)
	for _, x := range moved {
		delete(a.paths, x.Track.ID)
		delete(a.cache, cacheKey(x.OriginalPath))
	}
	kept := a.tracks[:0]
	removed := map[string]bool{}
	for _, x := range moved {
		removed[x.Track.ID] = true
	}
	for _, t := range a.tracks {
		if !removed[t.ID] {
			kept = append(kept, t)
		}
	}
	a.tracks = kept
	e1 := a.saveTrashLocked()
	cacheCopy := map[string]LibraryCacheEntry{}
	for k, v := range a.cache {
		cacheCopy[k] = v
	}
	a.mu.Unlock()
	_ = saveLibraryCacheFile(a.cachePath, cacheCopy)
	a.startReconcile("tracks moved to trash")
	if e1 != nil {
		jsonOut(w, 500, map[string]string{"error": e1.Error()})
		return
	}
	a.recordDiagnostic("TRACKS_TRASHED", "User moved tracks to reversible VexStream Trash.", map[string]any{"count": len(moved)})
	jsonOut(w, 200, map[string]any{"ok": true, "moved": len(moved)})
}
func (a *App) libraryTrashRestore(w http.ResponseWriter, r *http.Request) {
	var q struct {
		IDs []string `json:"ids"`
	}
	if decode(r, &q) != nil || len(q.IDs) == 0 {
		jsonOut(w, 400, map[string]string{"error": "no trash items selected"})
		return
	}
	a.mu.RLock()
	items := append([]TrashItem{}, a.trash...)
	a.mu.RUnlock()
	wanted := map[string]bool{}
	for _, id := range q.IDs {
		wanted[id] = true
	}
	chosen := []TrashItem{}
	for _, x := range items {
		if wanted[x.ID] {
			chosen = append(chosen, x)
		}
	}
	if len(chosen) != len(wanted) {
		jsonOut(w, 404, map[string]string{"error": "one or more trash items were not found"})
		return
	}
	for _, x := range chosen {
		if _, e := os.Stat(x.OriginalPath); e == nil {
			jsonOut(w, 409, map[string]string{"error": "Restore blocked because an original path is already occupied: " + x.OriginalPath})
			return
		}
	}
	restored := []TrashItem{}
	for _, x := range chosen {
		if e := moveFileSafely(x.TrashPath, x.OriginalPath); e != nil {
			for i := len(restored) - 1; i >= 0; i-- {
				_ = moveFileSafely(restored[i].OriginalPath, restored[i].TrashPath)
			}
			jsonOut(w, 500, map[string]string{"error": "Restore failed: " + e.Error()})
			return
		}
		restored = append(restored, x)
	}
	a.mu.Lock()
	next := a.trash[:0]
	for _, x := range a.trash {
		if !wanted[x.ID] {
			next = append(next, x)
		}
	}
	a.trash = next
	e := a.saveTrashLocked()
	a.mu.Unlock()
	a.startReconcile("tracks restored from trash")
	if e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true, "restored": len(restored)})
}
func removeCatalogRowsForPurged(items []TrashItem) []string {
	bySource := map[string]map[string]bool{}
	for _, x := range items {
		root := strings.TrimSpace(x.Track.Source)
		filename := strings.TrimSpace(x.Track.Filename)
		if root == "" || filename == "" {
			continue
		}
		if bySource[root] == nil {
			bySource[root] = map[string]bool{}
		}
		bySource[root][filename] = true
	}
	warnings := []string{}
	for root, names := range bySource {
		catalogPath := filepath.Join(root, "_vexmedia", "library.jsonl")
		rows, err := readCatalogRows(catalogPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", catalogPath, err))
			continue
		}
		kept := make([]map[string]any, 0, len(rows))
		changed := false
		for _, row := range rows {
			if names[strVal(row["filename"])] {
				changed = true
				continue
			}
			kept = append(kept, row)
		}
		if !changed {
			continue
		}
		if err = writeCatalogRows(catalogPath, kept); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", catalogPath, err))
		}
	}
	return warnings
}

func (a *App) libraryTrashPurge(w http.ResponseWriter, r *http.Request) {
	var q struct {
		IDs              []string `json:"ids"`
		Confirm          string   `json:"confirm"`
		PreserveMetadata bool     `json:"preserveMetadata"`
	}
	if decode(r, &q) != nil || q.Confirm != "PERMANENTLY_DELETE_TRASH" || len(q.IDs) == 0 {
		jsonOut(w, 400, map[string]string{"error": "explicit permanent-delete confirmation missing"})
		return
	}
	wanted := map[string]bool{}
	for _, id := range q.IDs {
		wanted[id] = true
	}
	a.mu.Lock()
	next := make([]TrashItem, 0, len(a.trash))
	purged := make([]TrashItem, 0, len(wanted))
	deleted := 0
	failed := make([]map[string]any, 0)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, x := range a.trash {
		if !wanted[x.ID] {
			next = append(next, x)
			continue
		}
		if e := os.Remove(x.TrashPath); e != nil && !os.IsNotExist(e) {
			failed = append(failed, map[string]any{"id": x.ID, "error": e.Error()})
			next = append(next, x)
			continue
		}
		deleted++
		purged = append(purged, x)
		if q.PreserveMetadata {
			a.deletedHistory = append(a.deletedHistory, DeletedHistoryItem{ID: randID(), Track: x.Track, OriginalPath: x.OriginalPath, DeletedAt: now})
		}
		delete(a.overlays, x.Track.ID)
		delete(a.playbackWindows, x.Track.ID)
		delete(a.firstSeen, x.Track.ID)
	}
	a.trash = next
	e1 := a.saveTrashLocked()
	e2 := a.saveDeletedHistoryLocked()
	e3 := a.saveOverlaysLocked()
	e4 := a.saveTrackState()
	e5 := a.savePlaybackWindowsLocked()
	a.mu.Unlock()
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil {
		jsonOut(w, 500, map[string]string{"error": "Permanent deletion completed but metadata state could not be fully persisted."})
		return
	}
	catalogWarnings := removeCatalogRowsForPurged(purged)
	if len(catalogWarnings) > 0 {
		a.recordDiagnostic("TRASH_CATALOG_CLEANUP_WARNING", "Audio deletion completed, but one or more source-side catalog rows could not be removed.", map[string]any{"warnings": catalogWarnings})
	}
	a.recordDiagnostic("TRASH_PERMANENTLY_DELETED", "User permanently deleted files from VexStream Trash.", map[string]any{"count": deleted, "preservedMetadata": q.PreserveMetadata, "catalogWarnings": len(catalogWarnings)})
	jsonOut(w, 200, map[string]any{"ok": len(failed) == 0, "deleted": deleted, "failed": failed, "preservedMetadata": q.PreserveMetadata, "catalogWarnings": catalogWarnings})
}
func (a *App) libraryReveal(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID string `json:"id"`
	}
	if decode(r, &q) != nil {
		jsonOut(w, 400, map[string]string{"error": "invalid track"})
		return
	}
	a.mu.RLock()
	p := a.paths[q.ID]
	a.mu.RUnlock()
	if p == "" {
		jsonOut(w, 404, map[string]string{"error": "track not found"})
		return
	}
	var e error
	if runtime.GOOS == "windows" {
		e = exec.Command("explorer.exe", "/select,", p).Start()
	} else if runtime.GOOS == "darwin" {
		e = exec.Command("/usr/bin/open", "-R", p).Start()
	} else {
		e = errors.New("Reveal file is not implemented on this platform")
	}
	if e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
func cleanPlaylistName(name string) string {
	name = strings.TrimSpace(name)
	runes := []rune(name)
	if len(runes) > 120 {
		name = string(runes[:120])
	}
	return name
}
func dedupeTrackIDs(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
func (a *App) playlistIndexLocked(id string) int {
	for i := range a.playlists {
		if a.playlists[i].ID == id {
			return i
		}
	}
	return -1
}
func (a *App) playlistsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonOut(w, 405, map[string]string{"error": "GET required"})
		return
	}
	a.mu.RLock()
	rows := append([]Playlist{}, a.playlists...)
	a.mu.RUnlock()
	jsonOut(w, 200, map[string]any{"playlists": rows})
}
func (a *App) playlistCreate(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Name     string   `json:"name"`
		TrackIDs []string `json:"trackIds"`
	}
	if decode(r, &q) != nil {
		jsonOut(w, 400, map[string]string{"error": "invalid playlist request"})
		return
	}
	q.Name = cleanPlaylistName(q.Name)
	if q.Name == "" {
		jsonOut(w, 400, map[string]string{"error": "playlist name required"})
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	row := Playlist{ID: randID(), Name: q.Name, TrackIDs: dedupeTrackIDs(q.TrackIDs), CreatedAt: now, UpdatedAt: now}
	a.mu.Lock()
	a.playlists = append(a.playlists, row)
	e := a.savePlaylistsLocked()
	a.mu.Unlock()
	if e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	a.recordDiagnostic("PLAYLIST_CREATED", "User created a durable local playlist.", map[string]any{"playlistId": row.ID, "trackCount": len(row.TrackIDs)})
	jsonOut(w, 200, map[string]any{"ok": true, "playlist": row})
}
func (a *App) playlistAdd(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID       string   `json:"id"`
		TrackIDs []string `json:"trackIds"`
	}
	if decode(r, &q) != nil || q.ID == "" {
		jsonOut(w, 400, map[string]string{"error": "playlist id required"})
		return
	}
	incoming := dedupeTrackIDs(q.TrackIDs)
	a.mu.Lock()
	i := a.playlistIndexLocked(q.ID)
	if i < 0 {
		a.mu.Unlock()
		jsonOut(w, 404, map[string]string{"error": "playlist not found"})
		return
	}
	seen := map[string]bool{}
	for _, id := range a.playlists[i].TrackIDs {
		seen[id] = true
	}
	added := 0
	for _, id := range incoming {
		if !seen[id] {
			a.playlists[i].TrackIDs = append(a.playlists[i].TrackIDs, id)
			seen[id] = true
			added++
		}
	}
	a.playlists[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	row := a.playlists[i]
	e := a.savePlaylistsLocked()
	a.mu.Unlock()
	if e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true, "added": added, "playlist": row})
}
func (a *App) playlistOrder(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID       string   `json:"id"`
		TrackIDs []string `json:"trackIds"`
	}
	if decode(r, &q) != nil || q.ID == "" {
		jsonOut(w, 400, map[string]string{"error": "playlist id required"})
		return
	}
	a.mu.Lock()
	i := a.playlistIndexLocked(q.ID)
	if i < 0 {
		a.mu.Unlock()
		jsonOut(w, 404, map[string]string{"error": "playlist not found"})
		return
	}
	old := dedupeTrackIDs(a.playlists[i].TrackIDs)
	ordered := dedupeTrackIDs(q.TrackIDs)
	oldSet, newSet := map[string]bool{}, map[string]bool{}
	for _, id := range old {
		oldSet[id] = true
	}
	for _, id := range ordered {
		newSet[id] = true
	}
	if len(oldSet) != len(newSet) {
		a.mu.Unlock()
		jsonOut(w, 409, map[string]string{"error": "playlist order must contain the same tracks exactly once"})
		return
	}
	for id := range oldSet {
		if !newSet[id] {
			a.mu.Unlock()
			jsonOut(w, 409, map[string]string{"error": "playlist order must contain the same tracks exactly once"})
			return
		}
	}
	a.playlists[i].TrackIDs = ordered
	a.playlists[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	e := a.savePlaylistsLocked()
	a.mu.Unlock()
	if e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) playlistRemove(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID       string   `json:"id"`
		TrackIDs []string `json:"trackIds"`
	}
	if decode(r, &q) != nil || q.ID == "" {
		jsonOut(w, 400, map[string]string{"error": "playlist id required"})
		return
	}
	wanted := map[string]bool{}
	for _, id := range q.TrackIDs {
		wanted[id] = true
	}
	a.mu.Lock()
	i := a.playlistIndexLocked(q.ID)
	if i < 0 {
		a.mu.Unlock()
		jsonOut(w, 404, map[string]string{"error": "playlist not found"})
		return
	}
	next := make([]string, 0, len(a.playlists[i].TrackIDs))
	for _, id := range a.playlists[i].TrackIDs {
		if !wanted[id] {
			next = append(next, id)
		}
	}
	a.playlists[i].TrackIDs = next
	a.playlists[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	e := a.savePlaylistsLocked()
	a.mu.Unlock()
	if e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) playlistRename(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if decode(r, &q) != nil || q.ID == "" {
		jsonOut(w, 400, map[string]string{"error": "playlist id required"})
		return
	}
	q.Name = cleanPlaylistName(q.Name)
	if q.Name == "" {
		jsonOut(w, 400, map[string]string{"error": "playlist name required"})
		return
	}
	a.mu.Lock()
	i := a.playlistIndexLocked(q.ID)
	if i < 0 {
		a.mu.Unlock()
		jsonOut(w, 404, map[string]string{"error": "playlist not found"})
		return
	}
	a.playlists[i].Name = q.Name
	a.playlists[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	e := a.savePlaylistsLocked()
	a.mu.Unlock()
	if e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) playlistDelete(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID      string `json:"id"`
		Confirm string `json:"confirm"`
	}
	if decode(r, &q) != nil || q.ID == "" || q.Confirm != "DELETE_PLAYLIST_ONLY" {
		jsonOut(w, 400, map[string]string{"error": "explicit playlist-delete confirmation missing"})
		return
	}
	a.mu.Lock()
	i := a.playlistIndexLocked(q.ID)
	if i < 0 {
		a.mu.Unlock()
		jsonOut(w, 404, map[string]string{"error": "playlist not found"})
		return
	}
	a.playlists = append(a.playlists[:i], a.playlists[i+1:]...)
	e := a.savePlaylistsLocked()
	a.mu.Unlock()
	if e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}

func containsFold(xs []string, p string) bool {
	for _, x := range xs {
		if strings.EqualFold(filepath.Clean(x), filepath.Clean(p)) {
			return true
		}
	}
	return false
}
func removeFold(xs []string, p string) []string {
	var o []string
	for _, x := range xs {
		if !strings.EqualFold(filepath.Clean(x), filepath.Clean(p)) {
			o = append(o, x)
		}
	}
	return o
}
func removeCandidate(xs []Candidate, p string) []Candidate {
	var o []Candidate
	for _, x := range xs {
		if !strings.EqualFold(filepath.Clean(x.Path), filepath.Clean(p)) {
			o = append(o, x)
		}
	}
	return o
}
func pathDepth(p string) int { return len(strings.Split(filepath.Clean(p), string(os.PathSeparator))) }
func trackID(p string) string {
	h := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(p))))
	return hex.EncodeToString(h[:12])
}

type CatalogMeta struct {
	ProviderID, SourceURL, Channel, Description, ThumbnailURL, IndexedAt, ArtistBasis string
	Duration                                                                          float64
	Tags, Categories                                                                  []string
}

func loadCatalog(root string) map[string]CatalogMeta {
	path := filepath.Join(root, "_vexmedia", "library.jsonl")
	f, e := os.Open(path)
	if e != nil {
		return nil
	}
	defer f.Close()
	out := map[string]CatalogMeta{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var row map[string]any
		if json.Unmarshal(sc.Bytes(), &row) != nil {
			continue
		}
		fn, _ := row["filename"].(string)
		pid, _ := row["providerId"].(string)
		meta, _ := row["metadata"].(map[string]any)
		m := CatalogMeta{ProviderID: pid, IndexedAt: strVal(row["indexedAt"])}
		if meta != nil {
			m.SourceURL = strVal(meta["sourceUrl"])
			m.Channel = strVal(meta["channel"])
			m.Description = strVal(meta["description"])
			m.Duration = numVal(meta["duration"])
			m.Tags = strList(meta["tags"])
			m.Categories = strList(meta["categories"])
			m.ThumbnailURL = strVal(meta["thumbnailUrl"])
			m.ArtistBasis = strVal(meta["artistBasis"])
		}
		if fn != "" {
			out[fn] = m
		}
	}
	return out
}
func strVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
func numVal(v any) float64 {
	if x, ok := v.(float64); ok {
		return x
	}
	return 0
}
func strList(v any) []string {
	a, ok := v.([]any)
	if !ok {
		return nil
	}
	var o []string
	for _, x := range a {
		if s, ok := x.(string); ok && s != "" {
			o = append(o, s)
		}
	}
	return o
}
func readTrack(path, source string, catalog map[string]CatalogMeta) Track {
	title, artist, album, genre, year := readID3(path)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	t := Track{ID: trackID(path), Path: path, Filename: filepath.Base(path), Title: title, Artist: artist, Album: album, Genre: genre, Year: year, Source: source}
	if c, ok := catalog[filepath.Base(path)]; ok {
		t.Duration = c.Duration
		t.ProviderID = c.ProviderID
		t.SourceURL = c.SourceURL
		t.Channel = c.Channel
		t.ArtistBasis = c.ArtistBasis
		if t.ArtistBasis == "" && c.SourceURL != "" && c.Channel != "" && strings.EqualFold(strings.TrimSpace(t.Artist), strings.TrimSpace(c.Channel)) {
			t.ArtistBasis = "legacy.youtube.channel.unverified"
		}
		t.Tags = c.Tags
		t.Categories = c.Categories
		t.Description = c.Description
		t.ThumbnailURL = c.ThumbnailURL
		t.IndexedAt = c.IndexedAt
	}
	return t
}
func readID3(path string) (title, artist, album, genre, year string) {
	f, e := os.Open(path)
	if e != nil {
		return
	}
	defer f.Close()
	h := make([]byte, 10)
	if _, e = io.ReadFull(f, h); e == nil && string(h[:3]) == "ID3" {
		ver := h[3]
		size := synchsafe(h[6:10])
		if size > 0 && size < 16*1024*1024 {
			tag := make([]byte, size)
			if _, e = io.ReadFull(f, tag); e == nil {
				for pos := 0; pos+10 <= len(tag); {
					id := string(tag[pos : pos+4])
					if strings.Trim(id, "\x00 ") == "" {
						break
					}
					var n int
					if ver == 4 {
						n = synchsafe(tag[pos+4 : pos+8])
					} else {
						n = int(binary.BigEndian.Uint32(tag[pos+4 : pos+8]))
					}
					if n <= 0 || pos+10+n > len(tag) {
						break
					}
					txt := decodeText(tag[pos+10 : pos+10+n])
					switch id {
					case "TIT2":
						title = txt
					case "TPE1":
						artist = txt
					case "TALB":
						album = txt
					case "TCON":
						genre = txt
					case "TDRC", "TYER":
						if year == "" {
							year = txt
						}
					}
					pos += 10 + n
				}
			}
		}
	}
	if title == "" && artist == "" && album == "" {
		st, _ := f.Stat()
		if st != nil && st.Size() >= 128 {
			b := make([]byte, 128)
			if _, e = f.ReadAt(b, st.Size()-128); e == nil && string(b[:3]) == "TAG" {
				title = trimLatin(b[3:33])
				artist = trimLatin(b[33:63])
				album = trimLatin(b[63:93])
				year = trimLatin(b[93:97])
			}
		}
	}
	return
}
func synchsafe(b []byte) int {
	if len(b) < 4 {
		return 0
	}
	return int(b[0]&0x7f)<<21 | int(b[1]&0x7f)<<14 | int(b[2]&0x7f)<<7 | int(b[3]&0x7f)
}
func decodeText(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	enc := b[0]
	d := b[1:]
	switch enc {
	case 0:
		return strings.TrimSpace(strings.ReplaceAll(string(d), "\x00", ""))
	case 3:
		return strings.TrimSpace(strings.TrimRight(string(d), "\x00"))
	case 1:
		if len(d) >= 2 && d[0] == 0xff && d[1] == 0xfe {
			return decodeUTF16(d[2:], binary.LittleEndian)
		}
		if len(d) >= 2 && d[0] == 0xfe && d[1] == 0xff {
			return decodeUTF16(d[2:], binary.BigEndian)
		}
		return decodeUTF16(d, binary.LittleEndian)
	case 2:
		return decodeUTF16(d, binary.BigEndian)
	}
	return ""
}
func decodeUTF16(b []byte, order binary.ByteOrder) string {
	var u []uint16
	for i := 0; i+1 < len(b); i += 2 {
		v := order.Uint16(b[i : i+2])
		if v == 0 {
			break
		}
		u = append(u, v)
	}
	return strings.TrimSpace(string(utf16.Decode(u)))
}
func trimLatin(b []byte) string { return strings.TrimSpace(string(bytes.TrimRight(b, "\x00 "))) }

// ----- YouTube import runtime / bridge -----

func trimDiagnosticText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…[truncated]"
}
func copyData(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func (a *App) recordDiagnostic(kind, message string, data map[string]any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.diagnosticListening {
		return
	}
	e := DiagnosticEvent{ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), Kind: trimDiagnosticText(kind, 120), Message: trimDiagnosticText(message, 4000), Data: copyData(data)}
	a.diagnosticEvents = append(a.diagnosticEvents, e)
	if len(a.diagnosticEvents) > 500 {
		a.diagnosticEvents = append([]DiagnosticEvent{}, a.diagnosticEvents[len(a.diagnosticEvents)-500:]...)
	}
}
func (a *App) diagnosticSnapshot() map[string]any {
	a.mu.RLock()
	events := append([]DiagnosticEvent{}, a.diagnosticEvents...)
	enabled := a.diagnosticListening
	sid := a.diagnosticSessionID
	started := a.diagnosticStartedAt
	tracks := len(a.tracks)
	sources := append([]string{}, a.config.Sources...)
	scan := map[string]any{"running": a.scanRunning, "reason": a.scanReason, "lastError": a.scanLastError, "startedAt": a.scanStartedAt, "completedAt": a.scanCompletedAt}
	recovery := a.recoveryProjection
	a.mu.RUnlock()
	providerProblems := make([]map[string]any, 0)
	for _, e := range events {
		if e.Kind == "PROVIDER_PROBLEM" {
			providerProblems = append(providerProblems, e.Data)
		}
	}
	return map[string]any{"schemaVersion": "vexstream.diagnostic-packet/v2", "appVersion": version, "platform": runtime.GOOS, "pid": os.Getpid(), "sessionId": sid, "sessionStartedAt": started, "listening": enabled, "eventCount": len(events), "events": events, "library": map[string]any{"tracks": tracks, "sources": sources, "recoveryProjection": recovery}, "scan": scan, "recentProviderProblems": providerProblems, "note": "Session diagnostics are held in memory only by this version. No browser cookies, provider tokens, or private session credentials are collected."}
}
func (a *App) diagnosticSession(w http.ResponseWriter, r *http.Request) {
	jsonOut(w, 200, a.diagnosticSnapshot())
}
func (a *App) diagnosticClientEvent(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Kind    string `json:"kind"`
		Message string `json:"message"`
		Stack   string `json:"stack"`
		Context string `json:"context"`
	}
	if decode(r, &q) != nil {
		jsonOut(w, 400, map[string]string{"error": "invalid diagnostic event"})
		return
	}
	data := map[string]any{}
	if q.Stack != "" {
		data["stack"] = trimDiagnosticText(q.Stack, 12000)
	}
	if q.Context != "" {
		data["context"] = trimDiagnosticText(q.Context, 2000)
	}
	a.recordDiagnostic("CLIENT_"+trimDiagnosticText(q.Kind, 80), q.Message, data)
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) diagnosticListeningAPI(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Enabled bool `json:"enabled"`
	}
	if decode(r, &q) != nil {
		jsonOut(w, 400, map[string]string{"error": "invalid listening setting"})
		return
	}
	a.mu.Lock()
	previous := a.diagnosticListening
	if previous && !q.Enabled {
		a.diagnosticEvents = append(a.diagnosticEvents, DiagnosticEvent{ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), Kind: "DIAGNOSTICS_DISABLED", Message: "Diagnostic listening disabled for this session."})
	}
	a.diagnosticListening = q.Enabled
	a.mu.Unlock()
	if q.Enabled && !previous {
		a.recordDiagnostic("DIAGNOSTICS_ENABLED", "Diagnostic listening enabled for this session.", nil)
	}
	jsonOut(w, 200, map[string]any{"enabled": q.Enabled})
}
func (a *App) diagnosticClear(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	a.diagnosticEvents = nil
	a.diagnosticEvents = make([]DiagnosticEvent, 0)
	a.mu.Unlock()
	a.recordDiagnostic("DIAGNOSTICS_CLEARED", "Session diagnostics were cleared by the user.", nil)
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) runDiagnosticSnapshot() map[string]any {
	instances := a.liveInstancesSnapshot(true)
	a.tryLiveRecovery(instances)
	a.mu.RLock()
	sources := append([]string{}, a.config.Sources...)
	tracks := len(a.tracks)
	cacheEntries := len(a.cache)
	configErr := a.configLoadError
	cacheErr := a.cacheLoadError
	stateErr := a.trackStateLoadError
	scan := map[string]any{"running": a.scanRunning, "reason": a.scanReason, "lastError": a.scanLastError}
	a.mu.RUnlock()
	sourceState := make([]map[string]any, 0, len(sources))
	for _, p := range sources {
		row := map[string]any{"path": p}
		if s, e := os.Stat(p); e == nil {
			row["exists"] = s.IsDir()
		} else {
			row["exists"] = false
			row["error"] = e.Error()
		}
		sourceState = append(sourceState, row)
	}
	py := a.importPython()
	return map[string]any{"schemaVersion": "vexstream.diagnostic-run/v1", "observedAt": time.Now().UTC().Format(time.RFC3339Nano), "appVersion": version, "pid": os.Getpid(), "currentURL": a.currentURL, "library": map[string]any{"tracks": tracks, "cacheEntries": cacheEntries, "sources": sourceState}, "state": map[string]any{"configError": configErr, "cacheError": cacheErr, "trackStateError": stateErr}, "scan": scan, "liveInstances": instances, "importRuntime": map[string]any{"ready": py != "", "path": py}}
}
func (a *App) diagnosticRun(w http.ResponseWriter, r *http.Request) {
	run := a.runDiagnosticSnapshot()
	a.recordDiagnostic("DIAGNOSTIC_RUN", "User ran current-session diagnostics.", run)
	jsonOut(w, 200, run)
}
func zipJSON(zw *zip.Writer, name string, v any) {
	w, e := zw.Create(name)
	if e != nil {
		return
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	w.Write(b)
}
func (a *App) diagnosticDownload(w http.ResponseWriter, r *http.Request) {
	run := a.runDiagnosticSnapshot()
	snap := a.diagnosticSnapshot()
	instances, _ := run["liveInstances"].([]LiveInstance)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zipJSON(zw, "manifest.json", map[string]any{"schemaVersion": "vexstream.diagnostic-zip/v1", "appVersion": version, "sessionId": snap["sessionId"], "generatedAt": time.Now().UTC().Format(time.RFC3339Nano), "note": "Generated in memory. VexStream 0.9.1 does not persist session diagnostics to disk by default."})
	zipJSON(zw, "diagnostic-run.json", run)
	zipJSON(zw, "session.json", snap)
	zipJSON(zw, "live-instances.json", instances)
	if ew, e := zw.Create("events.jsonl"); e == nil {
		for _, ev := range snap["events"].([]DiagnosticEvent) {
			b, _ := json.Marshal(ev)
			ew.Write(append(b, '\n'))
		}
	}
	if rw, e := zw.Create("README.txt"); e == nil {
		rw.Write([]byte("VexStream current-session diagnostics.\nNo browser cookies, provider tokens, or private session credentials are intentionally collected.\nSession events are memory-only until you explicitly download this ZIP.\n"))
	}
	zw.Close()
	a.recordDiagnostic("DIAGNOSTICS_DOWNLOADED", "User downloaded a current-session diagnostics ZIP.", map[string]any{"bytes": buf.Len()})
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="VexStream-Diagnostics-%s.zip"`, time.Now().Format("20060102-150405")))
	w.WriteHeader(200)
	w.Write(buf.Bytes())
}
func (a *App) materializeBridge() {
	b, e := fs.ReadFile(assets, "media_bridge.py")
	if e != nil {
		return
	}
	p := filepath.Join(appDataBase(), "bridge", "media_bridge.py")
	os.MkdirAll(filepath.Dir(p), 0755)
	os.WriteFile(p, b, 0644)
}
func (a *App) bridgePath() string { return filepath.Join(appDataBase(), "bridge", "media_bridge.py") }
func bridgeEnv() []string {
	env := os.Environ()
	if runtime.GOOS == "darwin" {
		basePath := os.Getenv("PATH")
		macPath := strings.Join([]string{"/opt/homebrew/bin", "/usr/local/bin", "/opt/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin", basePath}, string(os.PathListSeparator))
		filtered := make([]string, 0, len(env)+3)
		for _, item := range env {
			if !strings.HasPrefix(item, "PATH=") {
				filtered = append(filtered, item)
			}
		}
		env = append(filtered, "PATH="+macPath)
	}
	return append(env, "PYTHONUTF8=1", "PYTHONIOENCODING=utf-8")
}

type ProviderFailure struct{ Problem map[string]any }

func (e *ProviderFailure) Error() string {
	title, _ := e.Problem["title"].(string)
	msg, _ := e.Problem["message"].(string)
	action, _ := e.Problem["suggestedAction"].(string)
	out := strings.TrimSpace(title + " " + msg)
	if action != "" {
		out += " " + action
	}
	return strings.TrimSpace(out)
}
func parseProviderFailure(b []byte, stage string) map[string]any {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		var row map[string]any
		if json.Unmarshal([]byte(lines[i]), &row) == nil && row["type"] == "error" {
			if p, ok := row["problem"].(map[string]any); ok {
				return p
			}
		}
	}
	return map[string]any{"schemaVersion": "vexstream.provider-problem/v1", "appVersion": version, "provider": "youtube", "stage": stage, "code": "YOUTUBE_PROVIDER_ERROR", "title": "YouTube could not complete this request.", "message": "The provider/runtime returned an unexpected error.", "retryable": true, "suggestedAction": "Retry once. If it repeats, copy the diagnostic packet for your assistant.", "technical": strings.TrimSpace(string(b)), "observedAt": time.Now().UTC().Format(time.RFC3339)}
}
func (a *App) recordProblem(p map[string]any) {
	a.recordDiagnostic("PROVIDER_PROBLEM", "Provider/runtime problem recorded.", p)
}
func (a *App) diagnostics(w http.ResponseWriter, r *http.Request) {
	jsonOut(w, 200, a.diagnosticSnapshot())
}
func validPython(p string) bool {
	if p == "" {
		return false
	}
	if _, e := os.Stat(p); e != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, p, "-c", "import yt_dlp, curl_cffi")
	c.Env = bridgeEnv()
	c.SysProcAttr = sysProcHidden()
	return c.Run() == nil
}
func python310OrNewer(p string) bool {
	if p == "" {
		return false
	}
	if _, e := os.Stat(p); e != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, p, "-c", "import sys; raise SystemExit(0 if sys.version_info >= (3,10) else 1)")
	c.Env = bridgeEnv()
	c.SysProcAttr = sysProcHidden()
	return c.Run() == nil
}
func importRuntimePythonPath() string {
	venv := filepath.Join(appDataBase(), "runtime", "venv")
	if runtime.GOOS == "windows" {
		return filepath.Join(venv, "Scripts", "python.exe")
	}
	return filepath.Join(venv, "bin", "python3")
}
func firstExecutable(candidates ...string) string {
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		if info, e := os.Stat(candidate); e == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}
func findDarwinPython() string {
	candidates := []string{
		"/opt/homebrew/bin/python3",
		"/usr/local/bin/python3",
		"/Library/Frameworks/Python.framework/Versions/Current/bin/python3",
		"/opt/local/bin/python3",
		"/usr/bin/python3",
	}
	if p, e := exec.LookPath("python3"); e == nil {
		candidates = append([]string{p}, candidates...)
	}
	if matches, _ := filepath.Glob("/Library/Frameworks/Python.framework/Versions/*/bin/python3"); len(matches) > 0 {
		candidates = append(candidates, matches...)
	}
	for _, p := range candidates {
		if python310OrNewer(p) {
			return p
		}
	}
	return ""
}
func findBrew() string {
	candidates := []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"}
	if p, e := exec.LookPath("brew"); e == nil {
		candidates = append([]string{p}, candidates...)
	}
	return firstExecutable(candidates...)
}
func findWindowsWinGetFFmpeg(local string) string {
	packages := filepath.Join(local, "Microsoft", "WinGet", "Packages")
	roots, _ := filepath.Glob(filepath.Join(packages, "Gyan.FFmpeg_*"))
	for _, root := range roots {
		found := ""
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			if strings.EqualFold(info.Name(), "ffmpeg.exe") {
				found = path
				return filepath.SkipAll
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}
func ffmpegPath() string {
	candidates := []string{}
	if p, e := exec.LookPath("ffmpeg"); e == nil {
		candidates = append(candidates, p)
	}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, "/opt/homebrew/bin/ffmpeg", "/usr/local/bin/ffmpeg", "/opt/local/bin/ffmpeg")
	}
	if runtime.GOOS == "windows" {
		local := os.Getenv("LOCALAPPDATA")
		candidates = append(candidates,
			filepath.Join(local, "Microsoft", "WinGet", "Links", "ffmpeg.exe"),
			findWindowsWinGetFFmpeg(local),
		)
	}
	return firstExecutable(candidates...)
}
func runSetupCommand(timeout time.Duration, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, name, args...)
	c.Env = bridgeEnv()
	c.SysProcAttr = sysProcConsole()
	out, e := c.CombinedOutput()
	if e != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = e.Error()
		}
		return fmt.Errorf("%s failed: %s", filepath.Base(name), msg)
	}
	return nil
}
func (a *App) importPython() string {
	if p := importRuntimePythonPath(); validPython(p) {
		return p
	}
	if runtime.GOOS == "windows" {
		local := os.Getenv("LOCALAPPDATA")
		for _, p := range []string{filepath.Join(local, "VexMediaRescue", "runtime", "venv", "Scripts", "python.exe")} {
			if validPython(p) {
				return p
			}
		}
	}
	if runtime.GOOS == "darwin" {
		for _, p := range []string{"/opt/homebrew/bin/python3", "/usr/local/bin/python3", "/Library/Frameworks/Python.framework/Versions/Current/bin/python3", "/opt/local/bin/python3", "/usr/bin/python3"} {
			if validPython(p) {
				return p
			}
		}
	}
	if p, e := exec.LookPath("python3"); e == nil && validPython(p) {
		return p
	}
	if p, e := exec.LookPath("python"); e == nil && validPython(p) {
		return p
	}
	return ""
}
func (a *App) importStatus(w http.ResponseWriter, r *http.Request) {
	py := a.importPython()
	ff := ffmpegPath()
	if py != "" {
		detail := "Using VexStream import runtime."
		if strings.Contains(strings.ToLower(py), "vexmediarescue") {
			detail = "Reusing your existing VexMedia Rescue runtime — no duplicate install."
		}
		if ff == "" {
			detail = "YouTube search and source inspection are ready. MP3 import still needs FFmpeg; run setup/repair to finish the import runtime."
		}
		jsonOut(w, 200, map[string]any{"ready": true, "searchReady": true, "ffmpegReady": ff != "", "importReady": ff != "", "python": py, "ffmpeg": ff, "detail": detail})
		return
	}
	jsonOut(w, 200, map[string]any{"ready": false, "searchReady": false, "ffmpegReady": ff != "", "importReady": false, "detail": "You can still play local music. Set up import tools only when you want YouTube search/download."})
}
func (a *App) setupWindowsImportTools() (string, error) {
	local := os.Getenv("LOCALAPPDATA")
	venv := filepath.Join(local, "VexStreamMusic", "runtime", "venv")
	py := filepath.Join(venv, "Scripts", "python.exe")
	if _, e := os.Stat(py); e != nil {
		base := ""
		args := []string{}
		if p, lookErr := exec.LookPath("py.exe"); lookErr == nil {
			base = p
			args = []string{"-3", "-m", "venv", venv}
		} else if p, lookErr := exec.LookPath("python.exe"); lookErr == nil {
			base = p
			args = []string{"-m", "venv", venv}
		} else {
			return "", errors.New("Python 3 was not found. Install Python 3.10+ first, then retry")
		}
		if e = runSetupCommand(5*time.Minute, base, args...); e != nil {
			return "", fmt.Errorf("could not create the private VexStream Python runtime: %w", e)
		}
	}
	if e := runSetupCommand(15*time.Minute, py, "-m", "pip", "install", "--upgrade", "pip", "yt-dlp[default,curl-cffi]"); e != nil {
		return "", fmt.Errorf("could not install YouTube discovery dependencies: %w", e)
	}
	if !validPython(py) {
		return "", errors.New("the private VexStream Python runtime was created but yt-dlp/curl-cffi did not validate")
	}
	if ffmpegPath() == "" {
		if winget, e := exec.LookPath("winget.exe"); e == nil {
			installErr := runSetupCommand(20*time.Minute, winget, "install", "--id", "Gyan.FFmpeg", "-e", "--accept-package-agreements", "--accept-source-agreements", "--silent")
			if ffmpegPath() != "" {
				return "YouTube search, inspection, and MP3 import tools are ready on this PC.", nil
			}
			if installErr != nil {
				return "YouTube search and source inspection are ready, but FFmpeg is still not discoverable. WinGet may already have FFmpeg installed or may have declined an upgrade. VexStream will keep search available; run setup again after FFmpeg is available.", nil
			}
		}
	}
	if ffmpegPath() == "" {
		return "YouTube search and source inspection are ready. MP3 import still needs FFmpeg. Install Gyan.FFmpeg with WinGet (or put ffmpeg.exe on PATH), then run setup again.", nil
	}
	return "YouTube search, inspection, and MP3 import tools are ready on this PC.", nil
}
func (a *App) setupDarwinImportTools() (string, error) {
	base := findDarwinPython()
	brew := findBrew()
	if base == "" && brew != "" {
		if e := runSetupCommand(20*time.Minute, brew, "install", "python"); e != nil {
			return "", fmt.Errorf("Python 3.10+ was not available and Homebrew could not install it: %w", e)
		}
		base = findDarwinPython()
	}
	if base == "" {
		return "", errors.New("Python 3.10+ was not found. Install Python 3 from python.org or Homebrew once, then retry setup")
	}
	venv := filepath.Join(appDataBase(), "runtime", "venv")
	py := filepath.Join(venv, "bin", "python3")
	if _, e := os.Stat(py); e != nil {
		if e = os.MkdirAll(filepath.Dir(venv), 0755); e != nil {
			return "", e
		}
		if e = runSetupCommand(5*time.Minute, base, "-m", "venv", venv); e != nil {
			return "", fmt.Errorf("could not create the private VexStream Python runtime: %w", e)
		}
	}
	if e := runSetupCommand(10*time.Minute, py, "-m", "pip", "install", "--upgrade", "pip"); e != nil {
		return "", fmt.Errorf("could not update the private VexStream Python runtime: %w", e)
	}
	if e := runSetupCommand(15*time.Minute, py, "-m", "pip", "install", "yt-dlp[default,curl-cffi]"); e != nil {
		return "", fmt.Errorf("could not install YouTube discovery dependencies: %w", e)
	}
	if !validPython(py) {
		return "", errors.New("the private VexStream Python runtime was created but yt-dlp/curl-cffi did not validate")
	}
	if ffmpegPath() == "" && brew != "" {
		if e := runSetupCommand(30*time.Minute, brew, "install", "ffmpeg"); e != nil {
			return "YouTube search and source inspection are ready, but Homebrew could not finish FFmpeg setup. You can search now; MP3 import still needs FFmpeg.", nil
		}
	}
	if ffmpegPath() == "" {
		return "YouTube search and source inspection are ready. MP3 import still needs FFmpeg. Install FFmpeg once (Homebrew is supported), then run setup again.", nil
	}
	return "YouTube search, inspection, and MP3 import tools are ready on this Mac.", nil
}
func (a *App) importSetup(w http.ResponseWriter, r *http.Request) {
	var message string
	var e error
	switch runtime.GOOS {
	case "windows":
		message, e = a.setupWindowsImportTools()
	case "darwin":
		message, e = a.setupDarwinImportTools()
	default:
		jsonOut(w, 400, map[string]string{"error": "Automatic YouTube import setup is not yet implemented for this platform."})
		return
	}
	if e != nil {
		a.recordDiagnostic("IMPORT_RUNTIME_SETUP_FAILED", "YouTube import runtime setup failed.", map[string]any{"platform": runtime.GOOS, "error": e.Error()})
		jsonOut(w, 500, map[string]string{"error": "Import tool setup failed: " + e.Error()})
		return
	}
	a.recordDiagnostic("IMPORT_RUNTIME_SETUP_COMPLETED", "YouTube import runtime setup completed.", map[string]any{"platform": runtime.GOOS, "pythonReady": a.importPython() != "", "ffmpegReady": ffmpegPath() != ""})
	jsonOut(w, 200, map[string]string{"message": message})
}
func (a *App) bridgeJSON(stage string, args ...string) (map[string]any, error) {
	py := a.importPython()
	if py == "" {
		p := map[string]any{"schemaVersion": "vexstream.provider-problem/v1", "appVersion": version, "provider": "youtube", "stage": stage, "code": "IMPORT_RUNTIME_NOT_READY", "title": "YouTube import tools are not ready.", "message": "Local playback still works, but YouTube search/import needs its one-time runtime setup.", "retryable": false, "suggestedAction": "Open Discover → YouTube and run setup.", "technical": "no compatible Python/yt-dlp runtime found", "observedAt": time.Now().UTC().Format(time.RFC3339)}
		return nil, &ProviderFailure{Problem: p}
	}
	all := append([]string{a.bridgePath()}, args...)
	c := exec.Command(py, all...)
	c.Env = bridgeEnv()
	c.SysProcAttr = sysProcHidden()
	b, e := c.CombinedOutput()
	if e != nil {
		p := parseProviderFailure(b, stage)
		a.recordProblem(p)
		return nil, &ProviderFailure{Problem: p}
	}
	var out map[string]any
	if e = json.Unmarshal(b, &out); e != nil {
		p := parseProviderFailure(b, stage)
		a.recordProblem(p)
		return nil, &ProviderFailure{Problem: p}
	}
	return out, nil
}
func (a *App) importSearch(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Query string `json:"query"`
	}
	decode(r, &q)
	out, e := a.bridgeJSON("search", "search", q.Query, "--limit", "8")
	if e != nil {
		if pf, ok := e.(*ProviderFailure); ok {
			jsonOut(w, 502, map[string]any{"error": pf.Error(), "problem": pf.Problem})
			return
		}
		jsonOut(w, 502, map[string]string{"error": e.Error()})
		return
	}
	jsonOut(w, 200, out)
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
func appendUniqueString(rows []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return rows
	}
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row), value) {
			return rows
		}
	}
	return append(rows, value)
}
func appendDiscoveryQuery(rows []DiscoveryQuery, axis, label, query string) []DiscoveryQuery {
	query = strings.TrimSpace(query)
	if query == "" {
		return rows
	}
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.Query), query) {
			return rows
		}
	}
	return append(rows, DiscoveryQuery{Axis: axis, Label: label, Query: query})
}
func discoveryQueries(seed DiscoverySeed) []DiscoveryQuery {
	artist := firstNonEmpty(seed.Artist, seed.Channel)
	title := strings.TrimSpace(seed.Title)
	album := strings.TrimSpace(seed.Album)
	genre := strings.TrimSpace(seed.Genre)
	year := strings.TrimSpace(seed.Year)
	queries := []DiscoveryQuery{}
	if artist != "" && album != "" {
		queries = appendDiscoveryQuery(queries, "CLOSER", "Closer", artist+" "+album)
	} else if artist != "" && genre != "" {
		queries = appendDiscoveryQuery(queries, "CLOSER", "Closer", artist+" "+genre)
	} else if artist != "" {
		queries = appendDiscoveryQuery(queries, "CLOSER", "Closer", artist+" music")
	} else if title != "" {
		queries = appendDiscoveryQuery(queries, "CLOSER", "Closer", title)
	}
	if genre != "" {
		queries = appendDiscoveryQuery(queries, "NEIGHBORHOOD", "Neighborhood", genre+" music")
	} else if year != "" {
		queries = appendDiscoveryQuery(queries, "NEIGHBORHOOD", "Neighborhood", year+" music")
	}
	if year != "" && genre != "" {
		queries = appendDiscoveryQuery(queries, "ERA", "Same era", year+" "+genre+" music")
	} else if year != "" {
		queries = appendDiscoveryQuery(queries, "ERA", "Same era", year+" music")
	}
	if title != "" {
		queries = appendDiscoveryQuery(queries, "VERSIONS", "Versions", title+" cover remix live")
	}
	return queries
}
func discoveryCreatorKey(c DiscoveryCandidate) string {
	return normalizeMusicText(c.Channel)
}
func selectDiscoveryCandidates(pools [][]DiscoveryCandidate, limit, creatorCap int) []DiscoveryCandidate {
	if limit <= 0 {
		return nil
	}
	if creatorCap <= 0 {
		creatorCap = limit
	}
	positions := make([]int, len(pools))
	seen := map[string]bool{}
	creatorCounts := map[string]int{}
	selected := []DiscoveryCandidate{}
	for len(selected) < limit {
		addedRound := false
		for poolIndex, pool := range pools {
			for positions[poolIndex] < len(pool) {
				candidate := pool[positions[poolIndex]]
				positions[poolIndex]++
				key := discoveryCandidateKey(candidate)
				if key == "" || seen[key] {
					continue
				}
				creator := discoveryCreatorKey(candidate)
				if creator != "" && creatorCounts[creator] >= creatorCap {
					continue
				}
				seen[key] = true
				if creator != "" {
					creatorCounts[creator]++
				}
				selected = append(selected, candidate)
				addedRound = true
				break
			}
			if len(selected) >= limit {
				break
			}
		}
		if !addedRound {
			break
		}
	}
	return selected
}
func discoveryCandidateKey(c DiscoveryCandidate) string {
	if id := strings.ToLower(strings.TrimSpace(c.ProviderID)); id != "" {
		return "provider:" + id
	}
	if u := strings.ToLower(strings.TrimSpace(c.URL)); u != "" {
		return "url:" + u
	}
	return "text:" + normalizeMusicText(c.Title) + "|" + normalizeMusicText(c.Channel)
}
func mapString(row map[string]any, key string) string {
	if value, ok := row[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}
func mapFloat(row map[string]any, key string) float64 {
	switch value := row[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case json.Number:
		v, _ := value.Float64()
		return v
	}
	return 0
}
func discoveryCandidateFromMap(row map[string]any) DiscoveryCandidate {
	return DiscoveryCandidate{
		ProviderID: mapString(row, "id"),
		Title:      mapString(row, "title"),
		Channel:    mapString(row, "channel"),
		URL:        mapString(row, "url"),
		Thumbnail:  mapString(row, "thumbnail"),
		Duration:   mapFloat(row, "duration"),
	}
}
func discoverySameSeed(seed DiscoverySeed, candidate DiscoveryCandidate) bool {
	if seed.ProviderID != "" && candidate.ProviderID != "" && strings.EqualFold(seed.ProviderID, candidate.ProviderID) {
		return true
	}
	if seed.URL != "" && candidate.URL != "" && strings.EqualFold(seed.URL, candidate.URL) {
		return true
	}
	return normalizeMusicText(seed.Title) != "" && normalizeMusicText(seed.Title) == normalizeMusicText(candidate.Title) && normalizeMusicText(firstNonEmpty(seed.Artist, seed.Channel)) == normalizeMusicText(candidate.Channel)
}
func (a *App) discoveryLibraryMatch(candidate DiscoveryCandidate) DiscoveryLibraryMatch {
	matches := a.findDuplicateMatches(candidate.Title, "", candidate.Channel, candidate.ProviderID, candidate.URL, candidate.Duration, AudioQuality{})
	if len(matches) == 0 {
		return DiscoveryLibraryMatch{State: "NOT_IN_LIBRARY"}
	}
	m := matches[0]
	state := "POSSIBLE_MATCH"
	if m.Relation == "SAME_SOURCE" {
		state = "IN_LIBRARY"
	} else if m.Score >= 75 {
		state = "LIKELY_MATCH"
	}
	return DiscoveryLibraryMatch{State: state, TrackID: m.TrackID, Title: m.Title, Artist: m.Artist, Confidence: m.Confidence, Relation: m.Relation, Score: m.Score}
}
func (a *App) discoveryRadio(w http.ResponseWriter, r *http.Request) {
	var q struct {
		TrackID string         `json:"trackId"`
		Seed    *DiscoverySeed `json:"seed"`
	}
	if e := decode(r, &q); e != nil {
		jsonOut(w, 400, map[string]string{"error": e.Error()})
		return
	}
	var seed DiscoverySeed
	if id := strings.TrimSpace(q.TrackID); id != "" {
		a.mu.RLock()
		for _, track := range a.tracks {
			if track.ID == id {
				seed = DiscoverySeed{Kind: "library", TrackID: track.ID, ProviderID: track.ProviderID, URL: track.SourceURL, Title: track.Title, Artist: firstNonEmpty(track.Artist, track.SourceArtist), Album: track.Album, Genre: track.Genre, Year: track.Year, Channel: track.Channel, Duration: track.Duration}
				break
			}
		}
		a.mu.RUnlock()
		if seed.TrackID == "" {
			jsonOut(w, 404, map[string]string{"error": "Discovery seed track is no longer in this library."})
			return
		}
	} else if q.Seed != nil {
		seed = *q.Seed
		seed.Kind = "provider"
	} else {
		jsonOut(w, 400, map[string]string{"error": "Discovery requires a local track or provider candidate seed."})
		return
	}
	if strings.TrimSpace(seed.Title) == "" {
		jsonOut(w, 400, map[string]string{"error": "Discovery seed is missing a title."})
		return
	}
	queries := discoveryQueries(seed)
	if len(queries) == 0 {
		jsonOut(w, 400, map[string]string{"error": "Explore could not form a provider search from this seed."})
		return
	}
	pools := make([][]DiscoveryCandidate, 0, len(queries))
	for _, query := range queries {
		out, e := a.bridgeJSON("search", "search", query.Query, "--limit", "6")
		if e != nil {
			if pf, ok := e.(*ProviderFailure); ok {
				jsonOut(w, 502, map[string]any{"error": pf.Error(), "problem": pf.Problem})
				return
			}
			jsonOut(w, 502, map[string]string{"error": e.Error()})
			return
		}
		rows, _ := out["results"].([]any)
		pool := []DiscoveryCandidate{}
		poolSeen := map[string]bool{}
		for _, value := range rows {
			row, ok := value.(map[string]any)
			if !ok {
				continue
			}
			candidate := discoveryCandidateFromMap(row)
			if candidate.Title == "" || discoverySameSeed(seed, candidate) {
				continue
			}
			key := discoveryCandidateKey(candidate)
			if key == "" || poolSeen[key] {
				continue
			}
			poolSeen[key] = true
			candidate.Axis = query.Axis
			candidate.AxisLabel = query.Label
			candidate.Query = query.Query
			candidate.LibraryMatch = a.discoveryLibraryMatch(candidate)
			pool = append(pool, candidate)
		}
		pools = append(pools, pool)
	}
	candidates := selectDiscoveryCandidates(pools, 16, 2)
	jsonOut(w, 200, map[string]any{
		"schemaVersion": "vexstream.discovery-candidates/v2",
		"seed":          seed,
		"basis":         map[string]any{"provider": "youtube", "method": "SEARCH_DERIVED_MULTI_AXIS", "queries": queries, "creatorCap": 2, "selection": "ROUND_ROBIN"},
		"candidates":    candidates,
		"generatedAt":   time.Now().UTC().Format(time.RFC3339Nano),
	})
}
func titleStartsWithArtistCredit(title, artist string) bool {
	title = strings.TrimSpace(title)
	artist = strings.TrimSpace(artist)
	if title == "" || artist == "" {
		return false
	}
	// Artist inference requires an explicit title-credit boundary. A raw substring
	// match is intentionally insufficient (for example artist "Yes" in "Yes We Can").
	lowerTitle := strings.ToLower(title)
	lowerArtist := strings.ToLower(artist)
	if !strings.HasPrefix(lowerTitle, lowerArtist) {
		return false
	}
	rest := strings.TrimSpace(title[len(artist):])
	if rest == "" {
		return false
	}
	for _, sep := range []string{"-", "–", "—", "|", ":", "•"} {
		if strings.HasPrefix(rest, sep) {
			return true
		}
	}
	return false
}

func (a *App) inferExistingArtistSuggestion(title, sourceCreator string) *ArtistSuggestion {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}
	type candidate struct {
		artist  string
		count   int
		channel bool
	}
	a.mu.RLock()
	counts := map[string]*candidate{}
	for _, t := range a.tracks {
		artist := strings.TrimSpace(t.Artist)
		if artist == "" || strings.EqualFold(artist, "unknown artist") || strings.EqualFold(artist, "unknown") {
			continue
		}
		key := normalizeMusicText(artist)
		if key == "" {
			continue
		}
		c := counts[key]
		if c == nil {
			c = &candidate{artist: artist}
			counts[key] = c
		}
		c.count++
	}
	a.mu.RUnlock()

	creatorNorm := normalizeMusicText(sourceCreator)
	var best *candidate
	for key, c := range counts {
		if !titleStartsWithArtistCredit(title, c.artist) {
			continue
		}
		c.channel = creatorNorm != "" && creatorNorm == key
		if best == nil || len([]rune(c.artist)) > len([]rune(best.artist)) || (len([]rune(c.artist)) == len([]rune(best.artist)) && c.count > best.count) {
			best = c
		}
	}
	if best == nil {
		return nil
	}
	basis := "existing-library-artist-title-prefix"
	confidence := "HIGH"
	if best.channel {
		basis += "+source-creator-corroboration"
	}
	return &ArtistSuggestion{Artist: best.artist, Confidence: confidence, Basis: basis, MatchingTrackCount: best.count, ChannelCorroborated: best.channel}
}

func (a *App) importInspect(w http.ResponseWriter, r *http.Request) {
	var q struct {
		URL string `json:"url"`
	}
	decode(r, &q)
	out, e := a.bridgeJSON("inspect", "inspect", q.URL)
	if e != nil {
		if pf, ok := e.(*ProviderFailure); ok {
			jsonOut(w, 502, map[string]any{"error": pf.Error(), "problem": pf.Problem})
			return
		}
		jsonOut(w, 502, map[string]string{"error": e.Error()})
		return
	}
	artist, _ := out["artist"].(string)
	if strings.TrimSpace(artist) == "" {
		title, _ := out["title"].(string)
		creator, _ := out["channel"].(string)
		if suggestion := a.inferExistingArtistSuggestion(title, creator); suggestion != nil {
			out["artistSuggestion"] = suggestion
		}
	}
	jsonOut(w, 200, out)
}
func randID() string { b := make([]byte, 8); rand.Read(b); return hex.EncodeToString(b) }

func normalizeMusicText(s string) string {
	s = strings.ToLower(s)
	for _, n := range []string{"official video", "official audio", "lyrics", "lyric video", "music video", "hd", "4k", "remastered", "visualizer", "full album", "playlist"} {
		s = strings.ReplaceAll(s, n, " ")
	}
	var b strings.Builder
	space := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
			b.WriteRune(r)
			space = false
		} else if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
func compactMusicText(s string) string {
	return strings.ReplaceAll(normalizeMusicText(s), " ", "")
}
func musicTitleVariants(title string, prefixes ...string) []string {
	base := normalizeMusicText(title)
	if base == "" {
		return nil
	}
	out := []string{base}
	seen := map[string]bool{base: true}
	for _, prefix := range prefixes {
		p := normalizeMusicText(prefix)
		if p == "" || !strings.HasPrefix(base, p+" ") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(base, p))
		if len(strings.Fields(rest)) < 2 || seen[rest] {
			continue
		}
		seen[rest] = true
		out = append(out, rest)
	}
	return out
}
func musicTokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, x := range strings.Fields(normalizeMusicText(s)) {
		if len([]rune(x)) >= 2 {
			out[x] = true
		}
	}
	return out
}
func maxTitleSimilarity(incomingTitle, incomingArtist, sourceCreator string, t Track) (float64, bool) {
	incoming := musicTitleVariants(incomingTitle, incomingArtist, sourceCreator)
	existing := musicTitleVariants(t.Title, t.Artist, t.Channel)
	best := 0.0
	exact := false
	for _, a := range incoming {
		for _, b := range existing {
			if a == b || (compactMusicText(a) != "" && compactMusicText(a) == compactMusicText(b)) {
				exact = true
				best = 1
				continue
			}
			s := tokenSimilarity(a, b)
			if s > best {
				best = s
			}
		}
	}
	return best, exact
}
func estimatedLocalQuality(t Track) AudioQuality {
	q := AudioQuality{Codec: strings.TrimPrefix(strings.ToLower(filepath.Ext(t.Path)), "."), Container: strings.TrimPrefix(strings.ToLower(filepath.Ext(t.Path)), "."), Basis: "local.file-size-duration-estimate"}
	if q.Codec == "" {
		q.Codec = "unknown"
	}
	if t.Duration > 0 {
		if st, err := os.Stat(t.Path); err == nil && st.Size() > 0 {
			q.BitrateKbps = float64(st.Size()) * 8 / 1000 / t.Duration
		}
	}
	return q
}
func codecFamily(codec string) string {
	c := strings.ToLower(strings.TrimSpace(codec))
	switch {
	case strings.Contains(c, "mp3") || strings.Contains(c, "mp3lame"):
		return "mp3"
	case strings.Contains(c, "opus"):
		return "opus"
	case strings.Contains(c, "aac") || strings.Contains(c, "mp4a"):
		return "aac"
	case strings.Contains(c, "vorbis"):
		return "vorbis"
	default:
		return c
	}
}
func compareAudioQuality(source, local AudioQuality) QualityComparison {
	if source.BitrateKbps <= 0 || local.BitrateKbps <= 0 {
		return QualityComparison{State: "INSUFFICIENT_DATA", Summary: "Not enough format data to compare audio quality before download."}
	}
	sf, lf := codecFamily(source.Codec), codecFamily(local.Codec)
	if sf == "" || lf == "" || sf == "unknown" || lf == "unknown" || sf != lf {
		return QualityComparison{State: "NOT_DIRECTLY_COMPARABLE", Summary: "Different codecs or containers; bitrate alone cannot prove which copy sounds better."}
	}
	ratio := source.BitrateKbps / local.BitrateKbps
	if ratio >= 0.88 && ratio <= 1.12 {
		return QualityComparison{State: "SIMILAR_INDICATORS", Summary: "The available codec/bitrate indicators are in a similar range."}
	}
	if ratio > 1.12 {
		return QualityComparison{State: "SOURCE_HIGHER_BITRATE_SAME_CODEC", Summary: "The source reports a higher bitrate in the same codec family; listening is still the final quality check."}
	}
	return QualityComparison{State: "LOCAL_HIGHER_BITRATE_SAME_CODEC", Summary: "Your existing local copy has the higher bitrate in the same codec family; listening is still the final quality check."}
}
func tokenSimilarity(a, b string) float64 {
	aa, bb := musicTokens(a), musicTokens(b)
	if len(aa) == 0 || len(bb) == 0 {
		return 0
	}
	inter, union := 0, len(aa)
	for x := range bb {
		if aa[x] {
			inter++
		} else {
			union++
		}
	}
	return float64(inter) / float64(union)
}
func duplicateScore(title, artist, sourceCreator, providerID, sourceURL string, duration float64, t Track) (int, []string, string) {
	score := 0
	reasons := []string{}
	relation := "POSSIBLE_MATCH"
	if providerID != "" && t.ProviderID != "" && providerID == t.ProviderID {
		score += 100
		reasons = append(reasons, "same provider item")
		relation = "SAME_SOURCE"
	}
	if sourceURL != "" && t.SourceURL != "" && strings.EqualFold(sourceURL, t.SourceURL) {
		score += 100
		reasons = append(reasons, "same source URL")
		relation = "SAME_SOURCE"
	}
	sim, exactTitle := maxTitleSimilarity(title, artist, sourceCreator, t)
	if exactTitle {
		score += 55
		reasons = append(reasons, "same normalized song title")
	} else {
		if sim >= 0.82 {
			score += 42
			reasons = append(reasons, "very similar title")
		} else if sim >= 0.62 {
			score += 27
			reasons = append(reasons, "similar title")
		}
	}
	existingArtist := t.Artist
	if existingArtist == "" {
		existingArtist = t.Channel
	}
	if artist != "" && existingArtist != "" {
		if normalizeMusicText(artist) == normalizeMusicText(existingArtist) {
			score += 20
			reasons = append(reasons, "same artist/creator")
		} else if tokenSimilarity(artist, existingArtist) >= 0.7 {
			score += 10
			reasons = append(reasons, "similar artist/creator")
		}
	}
	if sourceCreator != "" && existingArtist != "" && normalizeMusicText(sourceCreator) == normalizeMusicText(existingArtist) {
		score += 18
		reasons = append(reasons, "source creator matches existing artist")
	}
	if duration > 0 && t.Duration > 0 {
		diff := duration - t.Duration
		if diff < 0 {
			diff = -diff
		}
		if diff <= 3 {
			score += 22
			reasons = append(reasons, "duration within 3s")
		} else if diff <= 10 {
			score += 13
			reasons = append(reasons, "duration within 10s")
		} else if diff <= 30 {
			score += 5
			reasons = append(reasons, "similar duration")
		} else if exactTitle {
			score -= 6
			reasons = append(reasons, fmt.Sprintf("duration differs by %ds; may be a different edit", int(diff+0.5)))
		}
		if relation != "SAME_SOURCE" && exactTitle {
			if diff <= 10 {
				relation = "LIKELY_SAME_RECORDING"
			} else {
				relation = "SAME_SONG_DIFFERENT_EDIT_POSSIBLE"
			}
		}
	}
	if relation == "POSSIBLE_MATCH" && exactTitle {
		relation = "SAME_SONG_POSSIBLE"
	}
	return score, reasons, relation
}
func confidenceForScore(score int) string {
	if score >= 95 {
		return "very likely"
	}
	if score >= 75 {
		return "likely"
	}
	return "possible"
}
func (a *App) findDuplicateMatches(title, artist, sourceCreator, providerID, sourceURL string, duration float64, sourceQuality AudioQuality) []DuplicateMatch {
	a.mu.RLock()
	tracks := append([]Track{}, a.tracks...)
	a.mu.RUnlock()
	matches := make([]DuplicateMatch, 0)
	for _, t := range tracks {
		score, reasons, relation := duplicateScore(title, artist, sourceCreator, providerID, sourceURL, duration, t)
		if score < 55 {
			continue
		}
		ea := t.Artist
		if ea == "" {
			ea = t.Channel
		}
		localQuality := estimatedLocalQuality(t)
		matches = append(matches, DuplicateMatch{TrackID: t.ID, Title: t.Title, Artist: ea, Album: t.Album, Duration: t.Duration, ThumbnailURL: t.ThumbnailURL, ProviderID: t.ProviderID, SourceURL: t.SourceURL, Score: score, Confidence: confidenceForScore(score), Relation: relation, Reasons: reasons, LocalQuality: localQuality, QualityComparison: compareAudioQuality(sourceQuality, localQuality)})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return strings.ToLower(matches[i].Title) < strings.ToLower(matches[j].Title)
		}
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > 4 {
		matches = matches[:4]
	}
	return matches
}
func (a *App) importDuplicates(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID            string       `json:"id"`
		URL           string       `json:"url"`
		Title         string       `json:"title"`
		Artist        string       `json:"artist"`
		Channel       string       `json:"channel"`
		Duration      float64      `json:"duration"`
		SourceQuality AudioQuality `json:"sourceQuality"`
		Sections      []struct {
			Index int     `json:"index"`
			Title string  `json:"title"`
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"sections"`
	}
	if e := decode(r, &q); e != nil {
		jsonOut(w, 400, map[string]string{"error": e.Error()})
		return
	}
	// Do not treat a YouTube channel/uploader as the musical artist for duplicate matching.
	artist := q.Artist
	items := make([]DuplicateItem, 0)
	if m := a.findDuplicateMatches(q.Title, artist, q.Channel, q.ID, q.URL, q.Duration, q.SourceQuality); len(m) > 0 {
		items = append(items, DuplicateItem{Kind: "whole", Title: q.Title, Duration: q.Duration, SourceQuality: q.SourceQuality, Matches: m})
	}
	for _, s := range q.Sections {
		dur := s.End - s.Start
		if dur < 0 {
			dur = 0
		}
		if m := a.findDuplicateMatches(s.Title, artist, q.Channel, "", "", dur, q.SourceQuality); len(m) > 0 {
			items = append(items, DuplicateItem{Kind: "section", Index: s.Index, Title: s.Title, Duration: dur, SourceQuality: q.SourceQuality, Matches: m})
		}
	}
	jsonOut(w, 200, map[string]any{"items": items, "matchCount": len(items), "advisory": true})
}
func (a *App) stagingRoot() string          { return filepath.Join(appDataBase(), "staging") }
func (a *App) jobStageDir(id string) string { return filepath.Join(a.stagingRoot(), id) }
func (a *App) persistJob(id string) {
	a.mu.RLock()
	j := a.jobs[id]
	if j == nil {
		a.mu.RUnlock()
		return
	}
	c := copyImportJob(j)
	stage := j.StagingDir
	a.mu.RUnlock()
	if stage == "" {
		return
	}
	saveJSONAtomic(filepath.Join(stage, "job.json"), c)
}
func (a *App) loadStagedJobs() {
	root := a.stagingRoot()
	rows, e := os.ReadDir(root)
	if e != nil {
		return
	}
	for _, d := range rows {
		if !d.IsDir() {
			continue
		}
		stage := filepath.Join(root, d.Name())
		b, e := os.ReadFile(filepath.Join(stage, "job.json"))
		if e != nil {
			continue
		}
		var j ImportJob
		if json.Unmarshal(b, &j) != nil || j.ID == "" {
			continue
		}
		if j.Status != "ready" {
			continue
		}
		if _, e = os.Stat(filepath.Join(stage, "manifest.json")); e != nil {
			continue
		}
		j.StagingDir = stage
		j.OwnerPID = os.Getpid()
		j.OwnerVersion = version
		j.OwnerURL = a.currentURL
		j.Remote = false
		a.jobs[j.ID] = &j
	}
}
func (a *App) importStart(w http.ResponseWriter, r *http.Request) {
	if a.importPython() == "" {
		jsonOut(w, 409, map[string]string{"error": "YouTube import tools are not ready. Run Discover → YouTube setup first."})
		return
	}
	if ffmpegPath() == "" {
		jsonOut(w, 409, map[string]string{"error": "FFmpeg is required before VexStream can download and add MP3 output. Run YouTube import setup/repair first."})
		return
	}
	var plan map[string]any
	if e := decode(r, &plan); e != nil {
		jsonOut(w, 400, map[string]string{"error": e.Error()})
		return
	}
	dest, _ := plan["destination"].(string)
	a.mu.RLock()
	ok := containsFold(a.config.Sources, dest)
	a.mu.RUnlock()
	if !ok {
		jsonOut(w, 400, map[string]string{"error": "Import destination must be an included VexStream music folder."})
		return
	}
	id := randID()
	title, _ := plan["sourceTitle"].(string)
	if strings.TrimSpace(title) == "" {
		title, _ = plan["wholeTitle"].(string)
	}
	if strings.TrimSpace(title) == "" {
		title = "YouTube import"
	}
	creator, _ := plan["sourceCreator"].(string)
	thumb, _ := plan["thumbnail"].(string)
	sourceURL, _ := plan["url"].(string)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	stage := a.jobStageDir(id)
	if e := os.MkdirAll(stage, 0755); e != nil {
		jsonOut(w, 500, map[string]string{"error": e.Error()})
		return
	}
	plan["stagingDirectory"] = stage
	plan["cancelPath"] = filepath.Join(stage, ".cancel")
	j := &ImportJob{ID: id, Title: title, Creator: creator, Thumbnail: thumb, SourceURL: sourceURL, Destination: dest, Status: "queued", Stage: "Waiting for an import slot", Progress: 0, Detail: "Queued — temporary staging is used while the download is active; finished outputs are added to your library automatically", StartedAt: now, OwnerPID: os.Getpid(), OwnerVersion: version, OwnerURL: a.currentURL, StagingDir: stage}
	a.mu.Lock()
	a.jobs[id] = j
	a.mu.Unlock()
	a.persistJob(id)
	a.recordDiagnostic("IMPORT_QUEUED", "YouTube import queued into reversible staging.", map[string]any{"jobId": id, "title": title, "creator": creator})
	go a.runImport(id, plan)
	jsonOut(w, 200, map[string]string{"jobId": id})
}
func (a *App) runImport(id string, plan map[string]any) {
	a.importSlots <- struct{}{}
	defer func() { <-a.importSlots }()
	a.mu.RLock()
	j := a.jobs[id]
	status := ""
	stage := ""
	if j != nil {
		status = j.Status
		stage = j.StagingDir
	}
	a.mu.RUnlock()
	if status == "cancelled" || status == "discarded" {
		return
	}
	if stage == "" {
		a.setJob(id, "failed", "Staging unavailable", 100, "Import staging directory is unavailable.")
		return
	}
	if _, e := os.Stat(filepath.Join(stage, ".cancel")); e == nil {
		a.setJob(id, "cancelled", "Cancelled", 100, "Cancelled before download started")
		os.RemoveAll(stage)
		return
	}
	a.setJob(id, "running", "Downloading into temporary staging", 1, "Temporary staging protects the library while this download is active. You can cancel before completion.")
	py := a.importPython()
	if py == "" {
		a.setJob(id, "failed", "Import tools unavailable", 100, "Set up YouTube import tools first.")
		return
	}
	planPath := filepath.Join(stage, "request.json")
	b, _ := json.Marshal(plan)
	if e := os.WriteFile(planPath, b, 0644); e != nil {
		a.setJob(id, "failed", "Could not create staging request", 100, e.Error())
		return
	}
	c := exec.Command(py, a.bridgePath(), "import", planPath)
	c.Env = bridgeEnv()
	c.SysProcAttr = sysProcHidden()
	stdout, e := c.StdoutPipe()
	if e != nil {
		a.setJob(id, "failed", "Could not start import", 100, e.Error())
		return
	}
	stderr, _ := c.StderrPipe()
	if e = c.Start(); e != nil {
		a.setJob(id, "failed", "Could not start import", 100, e.Error())
		return
	}
	a.mu.Lock()
	a.jobCommands[id] = c
	a.mu.Unlock()
	defer func() { a.mu.Lock(); delete(a.jobCommands, id); a.mu.Unlock() }()
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			a.appendJobLog(id, sc.Text())
		}
	}()
	sc := bufio.NewScanner(stdout)
	var final map[string]any
	cancelled := false
	for sc.Scan() {
		line := sc.Bytes()
		var m map[string]any
		if json.Unmarshal(line, &m) == nil {
			typ, _ := m["type"].(string)
			if typ == "progress" {
				stageText, _ := m["stage"].(string)
				progress := int(numVal(m["progress"]))
				a.setJob(id, "running", stageText, progress, stageText)
				continue
			}
			if typ == "cancelled" {
				cancelled = true
				a.setJob(id, "cancelled", "Cancelled", 100, "Download/import preparation cancelled; staged files were removed.")
				continue
			}
			if typ == "error" {
				if p, ok := m["problem"].(map[string]any); ok {
					a.recordProblem(p)
					a.mu.Lock()
					if j := a.jobs[id]; j != nil {
						j.Problem = p
					}
					a.mu.Unlock()
					pf := &ProviderFailure{Problem: p}
					a.setJob(id, "failed", "Import failed", 100, pf.Error())
				} else {
					a.setJob(id, "failed", "Import failed", 100, "The provider returned an unexpected error.")
				}
				continue
			}
			if _, ok := m["ok"]; ok {
				final = m
			}
		}
		a.appendJobLog(id, string(line))
	}
	e = c.Wait()
	if cancelled {
		os.RemoveAll(stage)
		return
	}
	if e != nil {
		a.mu.RLock()
		status = ""
		if j := a.jobs[id]; j != nil {
			status = j.Status
		}
		a.mu.RUnlock()
		if status == "cancelling" || status == "cancelled" {
			a.setJob(id, "cancelled", "Cancelled", 100, "Download/import preparation cancelled; staged files were removed.")
			os.RemoveAll(stage)
			return
		}
		if status != "failed" {
			a.setJob(id, "failed", "Import failed", 100, e.Error())
		}
		return
	}
	if final != nil {
		outs, _ := final["outputs"].([]any)
		a.mu.Lock()
		if j := a.jobs[id]; j != nil {
			j.OutputCount = len(outs)
		}
		a.mu.Unlock()
		a.setJob(id, "committing", "Adding to library", 100, fmt.Sprintf("%d file(s) downloaded. Finishing the automatic library add…", len(outs)))
		a.persistJob(id)
		if _, e := a.commitImportJobLocal(id); e != nil {
			a.setJob(id, "ready", "Needs attention", 100, "Download finished, but VexStream could not complete the automatic library add. Retry or discard the temporary files. "+e.Error())
			a.persistJob(id)
		}
		return
	}
	a.setJob(id, "failed", "Import did not produce a staging manifest", 100, "No staged output was returned.")
}
func (a *App) setJob(id, status, stage string, progress int, detail string) {
	terminal := false
	title := ""
	a.mu.Lock()
	if j := a.jobs[id]; j != nil {
		j.Status = status
		j.Stage = stage
		j.Progress = progress
		j.Detail = detail
		title = j.Title
		if (status == "done" || status == "failed" || status == "cancelled" || status == "discarded") && j.CompletedAt == "" {
			j.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
			terminal = true
		}
	}
	a.mu.Unlock()
	a.persistJob(id)
	if terminal {
		a.recordDiagnostic("IMPORT_"+strings.ToUpper(status), "Import job reached terminal state.", map[string]any{"jobId": id, "title": title, "status": status, "detail": detail})
	}
}
func (a *App) appendJobLog(id, line string) {
	a.mu.Lock()
	if j := a.jobs[id]; j != nil {
		j.Log = append(j.Log, line)
		if len(j.Log) > 100 {
			j.Log = j.Log[len(j.Log)-100:]
		}
	}
	a.mu.Unlock()
}
func copyImportJob(j *ImportJob) ImportJob {
	c := *j
	if j.Log != nil {
		c.Log = append([]string{}, j.Log...)
	}
	if j.TrackIDs != nil {
		c.TrackIDs = append([]string{}, j.TrackIDs...)
	}
	if j.Problem != nil {
		c.Problem = map[string]any{}
		for k, v := range j.Problem {
			c.Problem[k] = v
		}
	}
	return c
}
func importJobInGroups(id string, groups struct {
	Active    []ImportJob `json:"active"`
	Ready     []ImportJob `json:"ready"`
	Completed []ImportJob `json:"completed"`
}) bool {
	for _, rows := range [][]ImportJob{groups.Active, groups.Ready, groups.Completed} {
		for _, j := range rows {
			if j.ID == id {
				return true
			}
		}
	}
	return false
}
func postJSON(url string, payload any, target any) error {
	b, _ := json.Marshal(payload)
	c := http.Client{Timeout: 3 * time.Second}
	resp, e := c.Post(url, "application/json", bytes.NewReader(b))
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if msg := strVal(body["error"]); msg != "" {
			return errors.New(msg)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}
func (a *App) proxyImportAction(id, endpoint string) (map[string]any, error) {
	instances := a.liveInstancesSnapshot(false)
	for _, inst := range instances {
		if inst.Current || inst.URL == "" {
			continue
		}
		var groups struct {
			Active    []ImportJob `json:"active"`
			Ready     []ImportJob `json:"ready"`
			Completed []ImportJob `json:"completed"`
		}
		if fetchJSON(inst.URL+"api/import/jobs?local=1", &groups) != nil || !importJobInGroups(id, groups) {
			continue
		}
		var out map[string]any
		e := postJSON(inst.URL+endpoint+"?local=1", map[string]any{"id": id}, &out)
		if e != nil {
			return nil, fmt.Errorf("The import is owned by VexStream %s (PID %d), but that session could not perform this action remotely: %w. Open the owner session from Import activity if it is an older build", inst.Version, inst.PID, e)
		}
		return out, nil
	}
	return nil, errors.New("import job was not found in any live VexStream session")
}

func (a *App) importCancel(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID string `json:"id"`
	}
	if decode(r, &q) != nil || q.ID == "" {
		jsonOut(w, 400, map[string]string{"error": "job id required"})
		return
	}
	a.mu.Lock()
	j := a.jobs[q.ID]
	if j == nil {
		a.mu.Unlock()
		if r.URL.Query().Get("local") == "1" {
			jsonOut(w, 404, map[string]string{"error": "job not found in this VexStream process"})
			return
		}
		out, e := a.proxyImportAction(q.ID, "api/import/cancel")
		if e != nil {
			jsonOut(w, 409, map[string]string{"error": e.Error()})
			return
		}
		jsonOut(w, 200, out)
		return
	}
	status := j.Status
	stage := j.StagingDir
	if status == "queued" {
		j.Status = "cancelled"
		j.Stage = "Cancelled"
		j.Progress = 100
		j.Detail = "Cancelled before download started"
		j.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	} else if status == "running" || status == "cancelling" {
		j.Status = "cancelling"
		j.Stage = "Cancelling…"
		j.Detail = "Stopping provider/transcode work and removing temporary staging."
	} else {
		a.mu.Unlock()
		jsonOut(w, 409, map[string]string{"error": "this job is no longer cancellable"})
		return
	}
	cmd := a.jobCommands[q.ID]
	a.mu.Unlock()
	if stage != "" {
		os.MkdirAll(stage, 0755)
		os.WriteFile(filepath.Join(stage, ".cancel"), []byte("cancel\n"), 0644)
	}
	a.persistJob(q.ID)
	if status == "queued" {
		os.RemoveAll(stage)
	} else if cmd != nil {
		go func() {
			time.Sleep(4 * time.Second)
			a.mu.RLock()
			jj := a.jobs[q.ID]
			still := jj != nil && jj.Status == "cancelling"
			a.mu.RUnlock()
			if still && cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}()
	}
	responseStatus := "cancelling"
	if status == "queued" {
		responseStatus = "cancelled"
	}
	jsonOut(w, 200, map[string]any{"ok": true, "status": responseStatus})
}
func chooseAvailableFile(destDir, filename string) string {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)
	candidate := filepath.Join(destDir, filename)
	if _, e := os.Stat(candidate); os.IsNotExist(e) {
		return candidate
	}
	for i := 2; i < 10000; i++ {
		candidate = filepath.Join(destDir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, e := os.Stat(candidate); os.IsNotExist(e) {
			return candidate
		}
	}
	return filepath.Join(destDir, randID()+ext)
}
func readCatalogRows(path string) ([]map[string]any, error) {
	rows := []map[string]any{}
	f, e := os.Open(path)
	if os.IsNotExist(e) {
		return rows, nil
	}
	if e != nil {
		return nil, e
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var row map[string]any
		if json.Unmarshal(sc.Bytes(), &row) == nil {
			rows = append(rows, row)
		}
	}
	return rows, sc.Err()
}
func writeCatalogRows(path string, rows []map[string]any) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	tmp := path + ".tmp"
	f, e := os.Create(tmp)
	if e != nil {
		return e
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	for _, row := range rows {
		if e = enc.Encode(row); e != nil {
			f.Close()
			os.Remove(tmp)
			return e
		}
	}
	if e = f.Close(); e != nil {
		os.Remove(tmp)
		return e
	}
	return os.Rename(tmp, path)
}
func upsertCatalogRows(rows []map[string]any, entries []map[string]any) []map[string]any {
	for _, entry := range entries {
		provider := strVal(entry["provider"])
		pid := strVal(entry["providerId"])
		replaced := false
		for i, row := range rows {
			if strVal(row["provider"]) == provider && strVal(row["providerId"]) == pid && pid != "" {
				rows[i] = entry
				replaced = true
				break
			}
		}
		if !replaced {
			rows = append(rows, entry)
		}
	}
	return rows
}
func (a *App) commitImportJobLocal(id string) (int, error) {
	a.mu.RLock()
	j := a.jobs[id]
	if j == nil {
		a.mu.RUnlock()
		return 0, errors.New("job not found")
	}
	job := copyImportJob(j)
	stage := j.StagingDir
	a.mu.RUnlock()
	if job.Status != "ready" && job.Status != "committing" {
		return 0, errors.New("job is not ready to add")
	}
	a.mu.RLock()
	allowed := containsFold(a.config.Sources, job.Destination)
	a.mu.RUnlock()
	if !allowed {
		return 0, errors.New("the destination folder is no longer included in VexStream")
	}
	b, e := os.ReadFile(filepath.Join(stage, "manifest.json"))
	if e != nil {
		return 0, errors.New("staging manifest is missing: " + e.Error())
	}
	var manifest StagedManifest
	if e = json.Unmarshal(b, &manifest); e != nil {
		return 0, errors.New("staging manifest is invalid: " + e.Error())
	}
	if len(manifest.Outputs) == 0 {
		return 0, errors.New("staging manifest has no outputs")
	}
	type movedPair struct{ From, To string }
	moved := []movedPair{}
	entries := []map[string]any{}
	trackIDs := []string{}
	for _, out := range manifest.Outputs {
		src := filepath.Join(stage, filepath.FromSlash(out.RelativePath))
		if !strings.HasPrefix(filepath.Clean(src), filepath.Clean(stage)+string(os.PathSeparator)) {
			return 0, errors.New("invalid staged output path")
		}
		filename := out.Filename
		if filename == "" {
			filename = filepath.Base(src)
		}
		dst := chooseAvailableFile(job.Destination, filename)
		if e = moveFileSafely(src, dst); e != nil {
			for i := len(moved) - 1; i >= 0; i-- {
				_ = moveFileSafely(moved[i].To, moved[i].From)
			}
			return 0, errors.New("could not add staged files to the library: " + e.Error())
		}
		moved = append(moved, movedPair{src, dst})
		trackIDs = append(trackIDs, trackID(dst))
		entry := out.CatalogEntry
		if entry == nil {
			entry = map[string]any{}
		}
		entry["filename"] = filepath.Base(dst)
		entry["relativePath"] = filepath.Base(dst)
		if st, er := os.Stat(dst); er == nil {
			entry["bytes"] = st.Size()
		}
		entries = append(entries, entry)
	}
	catalogPath := filepath.Join(job.Destination, "_vexmedia", "library.jsonl")
	rows, e := readCatalogRows(catalogPath)
	if e == nil {
		rows = upsertCatalogRows(rows, entries)
		e = writeCatalogRows(catalogPath, rows)
	}
	if e != nil {
		for i := len(moved) - 1; i >= 0; i-- {
			_ = moveFileSafely(moved[i].To, moved[i].From)
		}
		return 0, errors.New("could not commit source metadata; staged files were rolled back: " + e.Error())
	}
	a.mu.Lock()
	if jj := a.jobs[id]; jj != nil {
		jj.StagingDir = ""
		jj.TrackIDs = append([]string{}, trackIDs...)
	}
	a.mu.Unlock()
	a.setJob(id, "done", "Added to library", 100, fmt.Sprintf("%d file(s) added to your library", len(moved)))
	os.RemoveAll(stage)
	a.startReconcile("automatic staged import committed")
	return len(moved), nil
}
func (a *App) importCommit(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID string `json:"id"`
	}
	if decode(r, &q) != nil || q.ID == "" {
		jsonOut(w, 400, map[string]string{"error": "job id required"})
		return
	}
	a.mu.RLock()
	j := a.jobs[q.ID]
	a.mu.RUnlock()
	if j == nil {
		if r.URL.Query().Get("local") == "1" {
			jsonOut(w, 404, map[string]string{"error": "job not found"})
			return
		}
		out, e := a.proxyImportAction(q.ID, "api/import/commit")
		if e != nil {
			jsonOut(w, 409, map[string]string{"error": e.Error()})
			return
		}
		jsonOut(w, 200, out)
		return
	}
	added, e := a.commitImportJobLocal(q.ID)
	if e != nil {
		jsonOut(w, 409, map[string]string{"error": e.Error()})
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true, "added": added})
}
func (a *App) importDiscard(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID string `json:"id"`
	}
	if decode(r, &q) != nil || q.ID == "" {
		jsonOut(w, 400, map[string]string{"error": "job id required"})
		return
	}
	a.mu.RLock()
	j := a.jobs[q.ID]
	if j == nil {
		a.mu.RUnlock()
		if r.URL.Query().Get("local") == "1" {
			jsonOut(w, 404, map[string]string{"error": "job not found"})
			return
		}
		out, e := a.proxyImportAction(q.ID, "api/import/discard")
		if e != nil {
			jsonOut(w, 409, map[string]string{"error": e.Error()})
			return
		}
		jsonOut(w, 200, out)
		return
	}
	status := j.Status
	stage := j.StagingDir
	a.mu.RUnlock()
	if status != "ready" && status != "failed" {
		jsonOut(w, 409, map[string]string{"error": "only prepared or failed staged jobs can be discarded"})
		return
	}
	a.mu.Lock()
	if jj := a.jobs[q.ID]; jj != nil {
		jj.StagingDir = ""
	}
	a.mu.Unlock()
	a.setJob(q.ID, "discarded", "Discarded", 100, "Temporary staged files were discarded; nothing was added to your library.")
	if stage != "" {
		os.RemoveAll(stage)
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (a *App) importJob(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	a.mu.RLock()
	j := a.jobs[id]
	if j == nil {
		a.mu.RUnlock()
		jsonOut(w, 404, map[string]string{"error": "job not found"})
		return
	}
	c := copyImportJob(j)
	a.mu.RUnlock()
	jsonOut(w, 200, c)
}
func classifyJobs(jobs map[string]*ImportJob) (active, ready, completed []ImportJob) {
	for _, j := range jobs {
		c := copyImportJob(j)
		switch c.Status {
		case "ready":
			ready = append(ready, c)
		case "done", "failed", "cancelled", "discarded":
			completed = append(completed, c)
		default:
			active = append(active, c)
		}
	}
	return
}
func sortJobGroups(active, ready, completed []ImportJob) {
	rank := func(s string) int {
		if s == "running" || s == "cancelling" || s == "committing" {
			return 0
		}
		if s == "queued" {
			return 1
		}
		return 2
	}
	sort.Slice(active, func(i, j int) bool {
		ri, rj := rank(active[i].Status), rank(active[j].Status)
		if ri != rj {
			return ri < rj
		}
		return active[i].StartedAt < active[j].StartedAt
	})
	sort.Slice(ready, func(i, j int) bool { return ready[i].StartedAt > ready[j].StartedAt })
	sort.Slice(completed, func(i, j int) bool { return completed[i].CompletedAt > completed[j].CompletedAt })
}
func (a *App) importJobs(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	localMap := map[string]*ImportJob{}
	for k, v := range a.jobs {
		c := copyImportJob(v)
		localMap[k] = &c
	}
	a.mu.RUnlock()
	active, ready, completed := classifyJobs(localMap)
	if r.URL.Query().Get("local") != "1" {
		instances := a.liveInstancesSnapshot(false)
		for _, inst := range instances {
			if inst.Current || inst.URL == "" {
				continue
			}
			var remote struct {
				Active    []ImportJob `json:"active"`
				Ready     []ImportJob `json:"ready"`
				Completed []ImportJob `json:"completed"`
			}
			if fetchJSON(inst.URL+"api/import/jobs?local=1", &remote) != nil {
				continue
			}
			mark := func(rows []ImportJob) []ImportJob {
				for i := range rows {
					rows[i].Remote = true
					rows[i].OwnerPID = inst.PID
					rows[i].OwnerVersion = inst.Version
					rows[i].OwnerURL = inst.URL
				}
				return rows
			}
			active = append(active, mark(remote.Active)...)
			ready = append(ready, mark(remote.Ready)...)
			completed = append(completed, mark(remote.Completed)...)
		}
	}
	sortJobGroups(active, ready, completed)
	if len(completed) > 40 {
		completed = completed[:40]
	}
	jsonOut(w, 200, map[string]any{"active": active, "ready": ready, "completed": completed, "maxConcurrent": 2, "currentPID": os.Getpid(), "currentVersion": version})
}

func quarantinePath(path string) string {
	return path + ".corrupt." + time.Now().UTC().Format("20060102T150405Z")
}
func validBackupConfig(path string) (Config, bool) {
	b, e := os.ReadFile(path)
	if e != nil {
		return Config{}, false
	}
	var c Config
	if json.Unmarshal(b, &c) != nil {
		return Config{}, false
	}
	return c, true
}

func fetchJSON(url string, target any) error {
	c := http.Client{Timeout: 1400 * time.Millisecond}
	resp, e := c.Get(url)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
func discoverWindowsPorts() []livePort {
	if runtime.GOOS != "windows" {
		return nil
	}
	script := `$procs=@(Get-CimInstance Win32_Process | Where-Object {$_.Name -like 'VexStreamMusic*.exe'});$rows=@();foreach($p in $procs){$started='';if($p.CreationDate){$started=$p.CreationDate.ToString('yyyy-MM-ddTHH:mm:ssK')};$ls=@(Get-NetTCPConnection -State Listen -OwningProcess $p.ProcessId -ErrorAction SilentlyContinue);foreach($l in $ls){$rows += [PSCustomObject]@{pid=[int]$p.ProcessId;port=[int]$l.LocalPort;startedAt=$started}}};$rows | ConvertTo-Json -Compress`
	c := exec.Command("powershell.exe", "-NoProfile", "-Command", script)
	c.SysProcAttr = sysProcHidden()
	b, e := c.Output()
	if e != nil || len(strings.TrimSpace(string(b))) == 0 {
		return nil
	}
	var raw any
	if json.Unmarshal(b, &raw) != nil {
		return nil
	}
	rows := make([]livePort, 0)
	add := func(m map[string]any) {
		rows = append(rows, livePort{PID: int(numVal(m["pid"])), Port: int(numVal(m["port"])), StartedAt: strVal(m["startedAt"])})
	}
	switch v := raw.(type) {
	case []any:
		for _, x := range v {
			if m, ok := x.(map[string]any); ok {
				add(m)
			}
		}
	case map[string]any:
		add(v)
	}
	return rows
}
func (a *App) scanLiveInstances() []LiveInstance {
	rows := discoverWindowsPorts()
	seen := map[string]bool{}
	out := make([]LiveInstance, 0)
	for _, row := range rows {
		if row.Port <= 0 {
			continue
		}
		url := fmt.Sprintf("http://127.0.0.1:%d/", row.Port)
		if seen[url] {
			continue
		}
		seen[url] = true
		var health struct {
			OK      bool   `json:"ok"`
			Version string `json:"version"`
		}
		if fetchJSON(url+"health", &health) != nil || !health.OK || health.Version == "" {
			continue
		}
		inst := LiveInstance{PID: row.PID, Version: health.Version, URL: url, StartedAt: row.StartedAt, Current: url == a.currentURL}
		var lib liveLibraryResponse
		if fetchJSON(url+"api/library", &lib) == nil {
			inst.TrackCount = len(lib.Tracks)
			for _, s := range lib.Sources {
				if s.Path != "" && !containsFold(inst.Sources, s.Path) {
					inst.Sources = append(inst.Sources, s.Path)
				}
			}
			inst.SourceCount = len(inst.Sources)
		}
		var jobs struct {
			Active []ImportJob `json:"active"`
		}
		if fetchJSON(url+"api/import/jobs?local=1", &jobs) == nil {
			inst.ActiveImports = len(jobs.Active)
		}
		out = append(out, inst)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Current != out[j].Current {
			return out[i].Current
		}
		if out[i].Version == out[j].Version {
			return out[i].PID < out[j].PID
		}
		return out[i].Version > out[j].Version
	})
	return out
}
func (a *App) liveInstancesSnapshot(force bool) []LiveInstance {
	a.mu.RLock()
	cached := append([]LiveInstance{}, a.liveInstances...)
	last := a.lifecycleScannedAt
	a.mu.RUnlock()
	if !force && !last.IsZero() && time.Since(last) < 8*time.Second {
		return cached
	}
	fresh := a.scanLiveInstances()
	a.mu.Lock()
	a.liveInstances = append([]LiveInstance{}, fresh...)
	a.lifecycleScannedAt = time.Now()
	a.mu.Unlock()
	return fresh
}
func chooseRecoveryInstance(instances []LiveInstance) *LiveInstance {
	var best *LiveInstance
	for i := range instances {
		inst := instances[i]
		if inst.Current || inst.TrackCount == 0 || inst.SourceCount == 0 {
			continue
		}
		if best == nil || inst.TrackCount > best.TrackCount {
			copy := inst
			best = &copy
		}
	}
	return best
}
func (a *App) tryLiveRecovery(instances []LiveInstance) bool {
	a.mu.RLock()
	eligible := len(a.config.Sources) == 0 && len(a.tracks) == 0 && len(a.cache) == 0 && !a.recoveryProjection
	a.mu.RUnlock()
	if !eligible {
		return false
	}
	best := chooseRecoveryInstance(instances)
	if best == nil {
		return false
	}
	var lib liveLibraryResponse
	if fetchJSON(best.URL+"api/library", &lib) != nil || len(lib.Tracks) == 0 {
		return false
	}
	paths := map[string]string{}
	for _, t := range lib.Tracks {
		if t.ID != "" && t.Path != "" {
			paths[t.ID] = t.Path
		}
	}
	sources := make([]string, 0)
	for _, s := range lib.Sources {
		if s.Path != "" && !containsFold(sources, s.Path) {
			sources = append(sources, s.Path)
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.config.Sources) != 0 || len(a.tracks) != 0 || len(a.cache) != 0 || a.recoveryProjection {
		return false
	}
	a.tracks = lib.Tracks
	a.paths = paths
	a.recoveryProjection = true
	a.recoveryFromURL = best.URL
	a.recoveryFromVersion = best.Version
	a.recoverySources = sources
	go a.recordDiagnostic("LIVE_RECOVERY_ACTIVATED", "Recovered a read-only library projection from another live VexStream session.", map[string]any{"version": best.Version, "url": best.URL, "tracks": len(lib.Tracks), "sources": len(sources)})
	return true
}
func (a *App) systemIssuesWith(instances []LiveInstance) []SystemIssue {
	a.mu.RLock()
	defer a.mu.RUnlock()
	issues := make([]SystemIssue, 0)
	if a.configLoadError != "" {
		_, ok := validBackupConfig(a.configBackupPath())
		issues = append(issues, SystemIssue{Code: "CONFIG_INVALID", Severity: "error", Title: "Library folder settings could not be read", Message: "VexStream kept the last-known cached library visible where possible instead of silently treating the library as empty.", Technical: a.configLoadError, Repairable: ok})
	}
	if a.trackStateLoadError != "" {
		issues = append(issues, SystemIssue{Code: "TRACK_STATE_INVALID", Severity: "warning", Title: "Recently Added history could not be read", Message: "Playback can continue. VexStream can preserve the unreadable file and rebuild this optional state.", Technical: a.trackStateLoadError, Repairable: true})
	}
	if a.cacheLoadError != "" {
		issues = append(issues, SystemIssue{Code: "INDEX_INVALID", Severity: "warning", Title: "Library index cache could not be read", Message: "Your source folders are unchanged. VexStream can quarantine the cache and rebuild it.", Technical: a.cacheLoadError, Repairable: true})
	}
	if s, e := os.Stat(filepath.Join(appDataBase(), "diagnostics", "provider-problems.jsonl")); e == nil && s.Size() > 0 {
		issues = append(issues, SystemIssue{Code: "LEGACY_DIAGNOSTIC_LOG", Severity: "warning", Title: "A legacy persistent diagnostic log exists", Message: "Older VexStream builds stored provider problems on disk. 0.9.1 uses memory-only session diagnostics by default.", Technical: filepath.Join(appDataBase(), "diagnostics", "provider-problems.jsonl"), Repairable: true})
	}
	for _, root := range a.config.Sources {
		if s, e := os.Stat(root); e != nil || !s.IsDir() {
			tech := "not a directory"
			if e != nil {
				tech = e.Error()
			}
			issues = append(issues, SystemIssue{Code: "SOURCE_MISSING", Severity: "warning", Title: "A music folder is unavailable", Message: root, Technical: tech, Repairable: false})
		}
	}
	others := make([]LiveInstance, 0)
	active := 0
	for _, inst := range instances {
		if !inst.Current {
			others = append(others, inst)
			active += inst.ActiveImports
		}
	}
	if len(others) > 0 {
		parts := make([]string, 0, len(others))
		for _, x := range others {
			parts = append(parts, fmt.Sprintf("v%s PID %d · %d tracks · %d active imports", x.Version, x.PID, x.TrackCount, x.ActiveImports))
		}
		msg := "Other VexStream processes are still alive: " + strings.Join(parts, "; ")
		if active > 0 {
			msg += ". At least one older process is still importing; do not terminate it until those jobs finish."
		}
		issues = append(issues, SystemIssue{Code: "MULTIPLE_LIVE_INSTANCES", Severity: "warning", Title: "Multiple VexStream sessions are running", Message: msg, Technical: "Lifecycle scan checks live VexStream localhost listeners.", Repairable: false})
	}
	if a.recoveryProjection {
		issues = append(issues, SystemIssue{Code: "RECOVERY_PROJECTION_ACTIVE", Severity: "warning", Title: "Showing a recovered live-library projection", Message: "This window recovered the visible library from live VexStream v" + a.recoveryFromVersion + ". The songs are usable, but the folder references are not yet persisted in this version's config.", Technical: a.recoveryFromURL, Repairable: len(a.recoverySources) > 0})
	}
	if a.scanLastError != "" {
		issues = append(issues, SystemIssue{Code: "INDEX_RECONCILE_FAILED", Severity: "warning", Title: "Background library reconciliation did not complete", Message: "The last-known cached library remains available. You can review diagnostics and retry the index.", Technical: a.scanLastError, Repairable: true})
	}
	return issues
}
func (a *App) repairPlan(instances []LiveInstance) []RepairAction {
	issues := a.systemIssuesWith(instances)
	seen := map[string]bool{}
	actions := make([]RepairAction, 0)
	add := func(x RepairAction) {
		if !seen[x.ID] {
			actions = append(actions, x)
			seen[x.ID] = true
		}
	}
	for _, i := range issues {
		switch i.Code {
		case "CONFIG_INVALID":
			if _, ok := validBackupConfig(a.configBackupPath()); ok {
				add(RepairAction{ID: "RESTORE_CONFIG_BACKUP", Title: "Restore last valid library-folder settings", Description: "Replace unreadable config.json with the latest automatically saved valid copy.", Effect: "Changes VexStream folder settings only; no media files are moved or deleted.", RequiresReload: true})
			}
		case "TRACK_STATE_INVALID":
			add(RepairAction{ID: "QUARANTINE_TRACK_STATE", Title: "Preserve and rebuild Recently Added state", Description: "Rename the unreadable track-first-seen file and start a clean state file.", Effect: "Keeps the unreadable file as a .corrupt copy. Audio files are untouched.", RequiresReload: false})
		case "INDEX_INVALID":
			add(RepairAction{ID: "QUARANTINE_INDEX", Title: "Preserve and rebuild library index", Description: "Rename the unreadable library-index file and rebuild from configured source folders.", Effect: "No media or folder settings are deleted.", RequiresReload: false})
		case "LEGACY_DIAGNOSTIC_LOG":
			add(RepairAction{ID: "CLEAR_LEGACY_DIAGNOSTICS", Title: "Clear legacy persistent diagnostic log", Description: "Delete the old provider-problems.jsonl written by pre-0.9.1 builds.", Effect: "Deletes diagnostic text only. No media, library config, or current in-memory session diagnostics are changed.", RequiresReload: false})
		case "INDEX_RECONCILE_FAILED":
			add(RepairAction{ID: "REBUILD_INDEX", Title: "Retry library index reconciliation", Description: "Re-read configured source folders in the background.", Effect: "No media files are changed.", RequiresReload: false})
		case "RECOVERY_PROJECTION_ACTIVE":
			add(RepairAction{ID: "ADOPT_RECOVERY_SOURCES", Title: "Adopt recovered music folders", Description: "Persist folder references recovered from the older live VexStream session into this version's config.", Effect: "Copies folder references only. No MP3 is moved, renamed, or deleted.", RequiresReload: false})
		}
	}
	a.mu.RLock()
	sources := len(a.config.Sources)
	cacheCount := len(a.cache)
	running := a.scanRunning
	a.mu.RUnlock()
	if sources > 0 && !running && cacheCount == 0 {
		add(RepairAction{ID: "REBUILD_INDEX", Title: "Build local library index", Description: "Read configured source folders in the background and create VexStream's additive cache.", Effect: "Existing config and media are unchanged.", RequiresReload: false})
	}
	return actions
}
func (a *App) systemPreflight(w http.ResponseWriter, r *http.Request) {
	instances := a.liveInstancesSnapshot(false)
	recovered := a.tryLiveRecovery(instances)
	a.mu.RLock()
	scan := map[string]any{"running": a.scanRunning, "reason": a.scanReason, "startedAt": a.scanStartedAt, "completedAt": a.scanCompletedAt, "lastError": a.scanLastError}
	tracks := len(a.tracks)
	sources := len(a.config.Sources)
	if a.recoveryProjection && sources == 0 {
		sources = len(a.recoverySources)
	}
	cached := len(a.cache)
	configPath := a.configPath
	cachePath := a.cachePath
	recovery := a.recoveryProjection
	a.mu.RUnlock()
	issues := a.systemIssuesWith(instances)
	count := 0
	for _, i := range issues {
		if i.Severity == "warning" || i.Severity == "error" {
			count++
		}
	}
	jsonOut(w, 200, map[string]any{"version": version, "issues": issues, "issueCount": count, "scan": scan, "library": map[string]any{"tracks": tracks, "sources": sources, "cachedEntries": cached}, "liveInstances": instances, "recoveryProjection": recovery, "recoveredNow": recovered, "paths": map[string]any{"config": configPath, "index": cachePath}, "compatibility": "Existing config.json and track-first-seen.json are read directly. library-index.json is additive/disposable. Live prior instances can be used as a read-only recovery source."})
}
func (a *App) systemRepairPlan(w http.ResponseWriter, r *http.Request) {
	instances := a.liveInstancesSnapshot(true)
	a.tryLiveRecovery(instances)
	jsonOut(w, 200, map[string]any{"actions": a.repairPlan(instances), "note": "No repair action deletes media files. Plans are shown before execution."})
}
func (a *App) systemRepair(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Confirm string   `json:"confirm"`
		Actions []string `json:"actions"`
	}
	if decode(r, &q) != nil || q.Confirm != "RUN_SAFE_REPAIRS" {
		jsonOut(w, 400, map[string]string{"error": "Explicit repair confirmation missing."})
		return
	}
	instances := a.liveInstancesSnapshot(true)
	a.tryLiveRecovery(instances)
	allowed := map[string]RepairAction{}
	for _, x := range a.repairPlan(instances) {
		allowed[x.ID] = x
	}
	ran := make([]string, 0)
	reload := false
	for _, id := range q.Actions {
		if _, ok := allowed[id]; !ok {
			continue
		}
		switch id {
		case "RESTORE_CONFIG_BACKUP":
			if e := copyFile(a.configBackupPath(), a.configPath); e == nil {
				a.loadConfig()
				ran = append(ran, id)
				reload = true
				a.startReconcile("config restored")
			}
		case "QUARANTINE_TRACK_STATE":
			if _, e := os.Stat(a.trackStatePath); e == nil {
				os.Rename(a.trackStatePath, quarantinePath(a.trackStatePath))
			}
			a.mu.Lock()
			a.firstSeen = map[string]string{}
			a.trackStateLoadError = ""
			a.mu.Unlock()
			a.saveTrackState()
			ran = append(ran, id)
			a.startReconcile("track state repaired")
		case "QUARANTINE_INDEX":
			if _, e := os.Stat(a.cachePath); e == nil {
				os.Rename(a.cachePath, quarantinePath(a.cachePath))
			}
			a.mu.Lock()
			a.cache = map[string]LibraryCacheEntry{}
			a.cacheLoadError = ""
			a.mu.Unlock()
			ran = append(ran, id)
			a.startReconcile("index repaired")
		case "REBUILD_INDEX":
			ran = append(ran, id)
			a.startReconcile("diagnostic repair")
		case "CLEAR_LEGACY_DIAGNOSTICS":
			os.Remove(filepath.Join(appDataBase(), "diagnostics", "provider-problems.jsonl"))
			ran = append(ran, id)
			a.recordDiagnostic("LEGACY_DIAGNOSTICS_CLEARED", "Legacy persistent provider diagnostic log was deleted.", nil)
		case "ADOPT_RECOVERY_SOURCES":
			a.mu.Lock()
			for _, s := range a.recoverySources {
				if !containsFold(a.config.Sources, s) {
					a.config.Sources = append(a.config.Sources, s)
				}
			}
			a.saveConfig()
			a.recoveryProjection = false
			a.recoveryFromURL = ""
			a.recoveryFromVersion = ""
			a.recoverySources = nil
			a.mu.Unlock()
			ran = append(ran, id)
			a.startReconcile("recovered sources adopted")
		}
	}
	jsonOut(w, 200, map[string]any{"ran": ran, "reloadRecommended": reload, "message": "Safe repairs started/completed. Background index work remains visible in System status."})
}

func trackIDSet(tracks []Track) map[string]bool {
	out := map[string]bool{}
	for _, t := range tracks {
		if t.ID != "" {
			out[t.ID] = true
		}
	}
	return out
}
func (a *App) sessionRetireCandidates() []RetireCandidate {
	instances := a.liveInstancesSnapshot(true)
	a.mu.RLock()
	currentTracks := append([]Track{}, a.tracks...)
	a.mu.RUnlock()
	currentSet := trackIDSet(currentTracks)
	rows := make([]RetireCandidate, 0)
	for _, inst := range instances {
		if inst.Current {
			continue
		}
		row := RetireCandidate{PID: inst.PID, Version: inst.Version, URL: inst.URL, TrackCount: inst.TrackCount, ActiveImports: inst.ActiveImports}
		if inst.ActiveImports > 0 {
			row.Reason = fmt.Sprintf("%d active import(s) still owned by this session", inst.ActiveImports)
			rows = append(rows, row)
			continue
		}
		var lib liveLibraryResponse
		if fetchJSON(inst.URL+"api/library", &lib) != nil {
			row.Reason = "Could not verify this session's live library"
			rows = append(rows, row)
			continue
		}
		unique := 0
		for _, t := range lib.Tracks {
			if t.ID != "" && !currentSet[t.ID] {
				unique++
			}
		}
		row.UniqueTrackCount = unique
		if unique > 0 {
			row.Reason = fmt.Sprintf("Holds %d track(s) not visible in the current session", unique)
		} else {
			row.Eligible = true
			row.Reason = "Idle and its visible track IDs are covered by the current session"
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Eligible != rows[j].Eligible {
			return rows[i].Eligible
		}
		if rows[i].Version == rows[j].Version {
			return rows[i].PID < rows[j].PID
		}
		return rows[i].Version > rows[j].Version
	})
	return rows
}
func (a *App) sessionRetirePlan(w http.ResponseWriter, r *http.Request) {
	rows := a.sessionRetireCandidates()
	eligible := make([]RetireCandidate, 0)
	held := make([]RetireCandidate, 0)
	for _, x := range rows {
		if x.Eligible {
			eligible = append(eligible, x)
		} else {
			held = append(held, x)
		}
	}
	jsonOut(w, 200, map[string]any{"eligible": eligible, "held": held, "currentPID": os.Getpid(), "warning": "Retiring an older server can stop playback in that older browser tab. Sessions with active imports or unique live tracks are held automatically."})
}
func stopWindowsPID(pid int) error {
	if runtime.GOOS != "windows" {
		return errors.New("session retirement is currently implemented for Windows")
	}
	if pid <= 0 || pid == os.Getpid() {
		return errors.New("refusing to stop the current VexStream process")
	}
	script := fmt.Sprintf(`$ErrorActionPreference='Stop';$p=Get-CimInstance Win32_Process -Filter "ProcessId=%d";if(-not $p){exit 0};if($p.Name -notlike 'VexStreamMusic*.exe'){throw 'PID no longer belongs to VexStreamMusic'};Stop-Process -Id %d -ErrorAction Stop`, pid, pid)
	c := exec.Command("powershell.exe", "-NoProfile", "-Command", script)
	c.SysProcAttr = sysProcHidden()
	return c.Run()
}
func (a *App) sessionRetire(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Confirm string `json:"confirm"`
		PIDs    []int  `json:"pids"`
	}
	if decode(r, &q) != nil || q.Confirm != "RETIRE_OLD_IDLE_SESSIONS" {
		jsonOut(w, 400, map[string]string{"error": "Explicit old-session retirement confirmation is missing."})
		return
	}
	plan := a.sessionRetireCandidates()
	allowed := map[int]RetireCandidate{}
	for _, x := range plan {
		if x.Eligible {
			allowed[x.PID] = x
		}
	}
	retired := make([]int, 0)
	failed := make([]map[string]any, 0)
	for _, pid := range q.PIDs {
		cand, ok := allowed[pid]
		if !ok {
			failed = append(failed, map[string]any{"pid": pid, "error": "session is no longer eligible or was held for safety"})
			continue
		}
		if e := stopWindowsPID(pid); e != nil {
			failed = append(failed, map[string]any{"pid": pid, "version": cand.Version, "error": e.Error()})
			continue
		}
		retired = append(retired, pid)
		a.recordDiagnostic("OLD_SESSION_RETIRED", "User retired an idle older VexStream session.", map[string]any{"pid": pid, "version": cand.Version})
	}
	a.mu.Lock()
	a.lifecycleScannedAt = time.Time{}
	a.mu.Unlock()
	time.Sleep(250 * time.Millisecond)
	jsonOut(w, 200, map[string]any{"retired": retired, "failed": failed, "message": fmt.Sprintf("Retired %d older VexStream session(s).", len(retired))})
}
func openBrowser(url string) {
	if runtime.GOOS == "windows" {
		exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start()
	} else if runtime.GOOS == "darwin" {
		exec.Command("/usr/bin/open", url).Start()
	}
}
