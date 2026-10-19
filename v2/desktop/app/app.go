// Package app composes the local Desktop V4 product.
//
// Feature semantics remain with their owners. This package chooses persistence placement, connects
// dependencies, exposes one loopback browser/admin handler, and reconnects the already-proven
// bounded VexStream phone producer without widening desktop administration to the LAN.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"

	v4app "github.com/vgong24/VexstreamMusic/v2"
	"github.com/vgong24/VexstreamMusic/v2/desktop/appdata"
	desktopconfig "github.com/vgong24/VexstreamMusic/v2/desktop/config"
	discoveryhttp "github.com/vgong24/VexstreamMusic/v2/desktop/config/discovery/httpapi"
	confighttp "github.com/vgong24/VexstreamMusic/v2/desktop/config/httpapi"
	"github.com/vgong24/VexstreamMusic/v2/desktop/config/picker"
	"github.com/vgong24/VexstreamMusic/v2/desktop/explore"
	explorehttp "github.com/vgong24/VexstreamMusic/v2/desktop/explore/httpapi"
	"github.com/vgong24/VexstreamMusic/v2/desktop/explore/librarymatch"
	exploreprovider "github.com/vgong24/VexstreamMusic/v2/desktop/explore/provider"
	importcommit "github.com/vgong24/VexstreamMusic/v2/desktop/imports/commit"
	importjobs "github.com/vgong24/VexstreamMusic/v2/desktop/imports/jobs"
	importjobshttp "github.com/vgong24/VexstreamMusic/v2/desktop/imports/jobs/httpapi"
	importprovider "github.com/vgong24/VexstreamMusic/v2/desktop/imports/provider"
	importreview "github.com/vgong24/VexstreamMusic/v2/desktop/imports/review"
	importreviewhttp "github.com/vgong24/VexstreamMusic/v2/desktop/imports/review/httpapi"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/workflow"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	libraryadminhttp "github.com/vgong24/VexstreamMusic/v2/desktop/library/admin/httpapi"
	libraryhttp "github.com/vgong24/VexstreamMusic/v2/desktop/library/httpapi"
	libraryservice "github.com/vgong24/VexstreamMusic/v2/desktop/library/service"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library/trash"
	trashhttp "github.com/vgong24/VexstreamMusic/v2/desktop/library/trash/httpapi"
	"github.com/vgong24/VexstreamMusic/v2/desktop/lifecycle"
	"github.com/vgong24/VexstreamMusic/v2/desktop/mobile"
	playersession "github.com/vgong24/VexstreamMusic/v2/desktop/player/session"
	playersessionhttp "github.com/vgong24/VexstreamMusic/v2/desktop/player/session/httpapi"
	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
	playlistshttp "github.com/vgong24/VexstreamMusic/v2/desktop/playlists/httpapi"
	youtubeprovider "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube"
	runtimeenv "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube/runtime"
	runtimehttp "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube/runtime/httpapi"
)

type Options struct {
	Version  string
	BuildRef string
	Layout   appdata.Layout
	Home     string
}

type Application struct {
	version     string
	buildRef    string
	layout      appdata.Layout
	mux         *http.ServeMux
	jobs        *importjobs.Manager
	mobile      *mobile.Service
	sessionPlan func() lifecycle.SessionPlan
	exit        chan struct{}
	exitOnce    sync.Once

	warningsMu  sync.RWMutex
	warnings    []string
}

func Open(options Options) (*Application, error) {
	options.Version = strings.TrimSpace(options.Version)
	options.BuildRef = strings.TrimSpace(options.BuildRef)
	if options.Version == "" || options.BuildRef == "" {
		return nil, errors.New("V4 version and buildRef are required")
	}
	if strings.TrimSpace(options.Layout.Base) == "" {
		return nil, errors.New("V4 app-data layout is required")
	}
	if strings.TrimSpace(options.Home) == "" {
		options.Home, _ = os.UserHomeDir()
	}

	config, err := desktopconfig.Open(
		options.Layout.Config,
		options.Layout.ConfigBackup,
	)
	if err != nil {
		return nil, err
	}

	openedLibrary, err := libraryservice.Open(libraryservice.Paths{
		Cache:           options.Layout.LibraryCache,
		FirstSeen:       options.Layout.FirstSeen,
		Overlays:        options.Layout.Overlays,
		PlaybackWindows: options.Layout.PlaybackWindows,
	})
	if err != nil {
		return nil, err
	}
	libraryOwner := openedLibrary.Library

	application := &Application{
		version:  options.Version,
		buildRef: options.BuildRef,
		layout:   options.Layout,
		exit:     make(chan struct{}),
		warnings: []string{},
	}

	if _, err := libraryOwner.Reconcile(config.Sources()); err != nil {
		application.addWarning("startup Library reconcile: " + err.Error())
	}
	if openedLibrary.DiscardedLegacyCache {
		application.addWarning(
			"legacy Library cache was discarded and rebuilt from current sources",
		)
	}

	librarySnapshot := func() library.Projection {
		return libraryOwner.Snapshot(config.Sources(), false)
	}
	trackSnapshot := func() []library.Track {
		projection := librarySnapshot()
		return append([]library.Track(nil), projection.Tracks...)
	}
	refreshLibrary := func() error {
		_, err := libraryOwner.Reconcile(config.Sources())
		return err
	}

	playlistsOwner, err := playlists.Open(options.Layout.Playlists)
	if err != nil {
		return nil, err
	}

	mobileOwner, err := mobile.New(mobile.Options{
		Root:             options.Layout.MobileBridge,
		Version:          options.Version,
		LibrarySnapshot:  librarySnapshot,
		PlaylistSnapshot: playlistsOwner.Snapshot,
	})
	if err != nil {
		return nil, err
	}
	application.mobile = mobileOwner

	trashOwner, err := trash.Open(
		trash.Paths{
			TrashState:     options.Layout.TrashState,
			DeletedHistory: options.Layout.DeletedHistory,
			FilesRoot:      options.Layout.TrashFiles,
			PurgeJournal:   options.Layout.TrashPurgeJournal,
		},
		trash.LibraryCleaner{State: libraryOwner},
	)
	if err != nil {
		return nil, err
	}
	if trashOwner.Snapshot().PendingPurge != nil {
		if _, err := trashOwner.RecoverPurge(); err != nil {
			application.addWarning("pending Trash purge recovery: " + err.Error())
		}
	}

	bridgeBytes, err := v4app.MediaBridgeBytes()
	if err != nil {
		return nil, err
	}
	if _, err := runtimeenv.MaterializeBridge(
		options.Layout.ProviderBridge,
		bridgeBytes,
	); err != nil {
		return nil, err
	}
	runtimeOwner := runtimeenv.NewService(options.Layout.SharedRuntime)
	providerBridge := youtubeprovider.NewDynamic(
		options.Layout.ProviderBridge,
		runtimeenv.BridgeEnv(),
		runtimeOwner.Python,
	)

	exploreMatcher := librarymatch.New(trackSnapshot)
	exploreSeedResolver := librarymatch.NewSeedResolver(librarySnapshot)
	exploreService := explore.Service{
		Provider:    exploreprovider.YouTube{Searcher: providerBridge},
		Matcher:     exploreMatcher,
		ProviderRef: "youtube",
	}

	importInspector := importprovider.YouTube{Inspector: providerBridge}
	importReviewService := importreview.Service{
		Inspector: importInspector,
		Library:   trackSnapshot,
	}
	importPreparer := importprovider.YouTubePreparation{Bridge: providerBridge}
	importWorkflow := workflow.Service{
		StagingRoot:        options.Layout.ImportStaging,
		Preparer:           importPreparer,
		Commit:             importcommit.CommitPrepared,
		DestinationAllowed: config.IsSource,
		AfterCommit: func(importcommit.Result) error {
			return refreshLibrary()
		},
	}
	importManager, err := importjobs.New(importWorkflow, 2)
	if err != nil {
		return nil, err
	}
	application.jobs = importManager

	web, err := fs.Sub(v4app.Assets, "web")
	if err != nil {
		importManager.Close()
		return nil, err
	}
	mux := http.NewServeMux()
	application.mux = mux

	libraryHTTP := libraryhttp.New(librarySnapshot)
	libraryAdminHTTP := libraryadminhttp.New(
		librarySnapshot,
		libraryOwner,
	)
	playlistsHTTP := playlistshttp.New(playlistsOwner)
	configHTTP := confighttp.NewWithPicker(
		config,
		func(sources []string) error {
			_, err := libraryOwner.Reconcile(sources)
			return err
		},
		picker.Choose,
	)
	discoveryHTTP := discoveryhttp.New(config.Snapshot, func() string {
		return options.Home
	})
	trashHTTP := trashhttp.New(
		trashOwner,
		librarySnapshot,
		refreshLibrary,
	)
	exploreHTTP := explorehttp.New(exploreService, exploreSeedResolver)
	importReviewHTTP := importreviewhttp.New(importReviewService)
	importJobsHTTP := importjobshttp.NewWithDestinations(
		importManager,
		config.Sources,
	)
	playerSessionOwner := playersession.New(options.Layout.PlayerSession)
	playerSessionHTTP := playersessionhttp.New(playerSessionOwner)
	application.sessionPlan = func() lifecycle.SessionPlan {
		projection := librarySnapshot()
		refs := make([]string, 0, len(projection.Tracks))
		for _, track := range projection.Tracks {
			if strings.TrimSpace(track.ID) != "" {
				refs = append(refs, track.ID)
			}
		}
		return lifecycle.ReviewSessions(os.Getpid(), refs)
	}

	runtimeHTTP := runtimehttp.NewWith(
		options.Layout.SharedRuntime,
		func(string) runtimeenv.Status {
			return runtimeOwner.Status()
		},
		func(ctx context.Context, _ string) (string, error) {
			return runtimeOwner.Setup(ctx)
		},
	)

	register(mux, libraryHTTP,
		"/api/library",
		"/media/",
	)
	register(mux, libraryAdminHTTP,
		"/api/library/overlay",
		"/api/library/playback-window",
	)
	register(mux, playlistsHTTP,
		"/api/playlists",
		"/api/playlists/create",
		"/api/playlists/add",
		"/api/playlists/order",
		"/api/playlists/remove",
		"/api/playlists/rename",
		"/api/playlists/delete",
	)
	register(mux, configHTTP,
		"/api/source/config",
		"/api/source/choose",
		"/api/source/add",
		"/api/source/remove",
		"/api/candidate/ignore",
		"/api/candidate/unignore",
		"/api/ignored/clear",
		"/api/source/config/restore-backup",
	)
	register(mux, discoveryHTTP, "/api/discover")
	register(mux, trashHTTP,
		"/api/library/trash",
		"/api/library/trash/restore",
		"/api/library/trash/purge",
		"/api/library/trash/purge/recover",
	)
	register(mux, exploreHTTP, "/api/discovery/radio")
	register(mux, importReviewHTTP, "/api/import/inspect")
	register(mux, importJobsHTTP,
		"/api/import/jobs",
		"/api/import/destinations",
		"/api/import/start",
		"/api/import/cancel",
		"/api/import/commit",
		"/api/import/discard",
		"/api/import/recover-commit",
	)
	register(mux, runtimeHTTP,
		"/api/import/status",
		"/api/import/setup",
	)
	register(mux, playerSessionHTTP, "/api/player/session")
	mux.Handle("/api/mobile/", mobileOwner.AdminHandler())

	mux.HandleFunc("/api/library/rescan", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			jsonOut(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "POST required",
			})
			return
		}
		projection, err := libraryOwner.Reconcile(config.Sources())
		if err != nil {
			jsonOut(w, http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}
		jsonOut(w, http.StatusOK, map[string]any{
			"ok":     true,
			"tracks": len(projection.Tracks),
		})
	})

	mux.HandleFunc("/health", application.health)
	mux.HandleFunc("/api/system/sessions/retire-plan", application.sessionRetirePlanHandler)
	mux.HandleFunc("/api/system/sessions/retire", application.sessionRetireHandler)
	mux.HandleFunc("/api/system/exit", application.exitHandler)
	mux.Handle("/", http.FileServer(http.FS(web)))

	if err := mobileOwner.ResumeRemembered(); err != nil {
		application.addWarning("remembered VexLife phone bridge: " + err.Error())
	}
	return application, nil
}

func (a *Application) Handler() http.Handler {
	return a.mux
}

func (a *Application) ExitRequested() <-chan struct{} {
	return a.exit
}

func (a *Application) Close() {
	if a.mobile != nil {
		a.mobile.Close()
	}
	if a.jobs != nil {
		a.jobs.Close()
	}
}

func (a *Application) StartupWarnings() []string {
	a.warningsMu.RLock()
	defer a.warningsMu.RUnlock()
	return append([]string(nil), a.warnings...)
}

func (a *Application) addWarning(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	a.warningsMu.Lock()
	a.warnings = append(a.warnings, message)
	a.warningsMu.Unlock()
}

func (a *Application) health(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "GET required",
		})
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{
		"ok":       true,
		"version":  a.version,
		"buildRef": a.buildRef,
		"pid":      os.Getpid(),
		"warnings": a.StartupWarnings(),
	})
}

func (a *Application) sessionRetirePlanHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "GET required",
		})
		return
	}
	if a.sessionPlan == nil {
		jsonOut(w, http.StatusServiceUnavailable, map[string]string{
			"error": "VexStream session review is unavailable",
		})
		return
	}
	jsonOut(w, http.StatusOK, a.sessionPlan())
}

func (a *Application) sessionRetireHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "POST required",
		})
		return
	}
	var request struct {
		Confirm string `json:"confirm"`
		PIDs    []int  `json:"pids"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil ||
		request.Confirm != "RETIRE_OLD_IDLE_SESSIONS" {
		jsonOut(w, http.StatusBadRequest, map[string]string{
			"error": "explicit old-session retirement confirmation is missing",
		})
		return
	}
	if a.sessionPlan == nil {
		jsonOut(w, http.StatusServiceUnavailable, map[string]string{
			"error": "VexStream session review is unavailable",
		})
		return
	}

	plan := a.sessionPlan()
	retired, failed := lifecycle.RetireEligibleSessions(
		os.Getpid(),
		request.PIDs,
		plan,
	)
	jsonOut(w, http.StatusOK, map[string]any{
		"retired": retired,
		"failed":  failed,
		"message": fmt.Sprintf("Retired %d older VexStream session(s).", len(retired)),
	})
}

func (a *Application) exitHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "POST required",
		})
		return
	}
	var request struct {
		Confirm string `json:"confirm"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil ||
		request.Confirm != "EXIT_VEXSTREAM" {
		jsonOut(w, http.StatusBadRequest, map[string]string{
			"error": "explicit exit confirmation required",
		})
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{
		"ok":    true,
		"state": "EXITING",
	})
	a.exitOnce.Do(func() {
		close(a.exit)
	})
}

func register(
	mux *http.ServeMux,
	handler http.Handler,
	patterns ...string,
) {
	for _, pattern := range patterns {
		mux.Handle(pattern, handler)
	}
}

func jsonOut(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
