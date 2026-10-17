// Package mobile owns the bounded Desktop V4 producer used by VexLife on the same Wi-Fi.
//
// Human/product walk:
//
//	Desktop phone affordance
//	-> open a five-minute pairing window
//	-> advertise _vexstream._tcp with the persistent TLS fingerprint
//	-> VexLife selects the producer and compares the six-digit code
//	-> Desktop approves or denies the request
//	-> a remembered bearer authorizes only /v1/catalog and /v1/media/<trackRef>
//
// The ordinary Desktop browser/admin server remains loopback-only. This package does not own
// desktop administration, remote/off-home authority, Home membership, or Android presentation.
package mobile

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
)

const (
	DefaultLANPort = 18766
	serviceType    = "_vexstream._tcp"
)

type Options struct {
	Root             string
	Version          string
	LibrarySnapshot  func() library.Projection
	PlaylistSnapshot func() []playlists.Playlist
}

type Service struct {
	mu sync.Mutex

	root             string
	version          string
	librarySnapshot  func() library.Projection
	playlistSnapshot func() []playlists.Playlist

	running            bool
	pairingWindowUntil time.Time
	pending            map[string]*pendingPairing
	devices            map[string]pairedDevice

	server   *http.Server
	listener net.Listener
	mdns     *zeroconf.Server

	certPath          string
	keyPath           string
	certificateSHA256 string
	pairingCode       string
	displayName       string

	fingerprintMu    sync.Mutex
	fingerprintCache map[string]fingerprintCacheEntry
}

func New(options Options) (*Service, error) {
	options.Root = strings.TrimSpace(options.Root)
	options.Version = strings.TrimSpace(options.Version)

	if options.Root == "" {
		return nil, errors.New("mobile bridge root is required")
	}
	if options.Version == "" {
		return nil, errors.New("mobile bridge source version is required")
	}
	if options.LibrarySnapshot == nil || options.PlaylistSnapshot == nil {
		return nil, errors.New("mobile bridge requires Library and Playlist snapshots")
	}
	if err := os.MkdirAll(options.Root, 0o700); err != nil {
		return nil, err
	}

	service := &Service{
		root:             options.Root,
		version:          options.Version,
		librarySnapshot:  options.LibrarySnapshot,
		playlistSnapshot: options.PlaylistSnapshot,
		pending:          map[string]*pendingPairing{},
		devices:          map[string]pairedDevice{},
		fingerprintCache: map[string]fingerprintCacheEntry{},
	}

	service.certPath = filepath.Join(options.Root, "server-cert.pem")
	service.keyPath = filepath.Join(options.Root, "server-key.pem")
	fingerprint, code, err := ensureCertificate(service.certPath, service.keyPath)
	if err != nil {
		return nil, err
	}
	service.certificateSHA256 = fingerprint
	service.pairingCode = code

	host, _ := os.Hostname()
	host = strings.TrimSpace(host)
	if host == "" {
		host = "Windows PC"
	}
	service.displayName = "VexStream on " + host

	persisted, err := os.ReadFile(filepath.Join(options.Root, "paired-devices.json"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(persisted) > 0 {
		devices, err := decodeRememberedDevices(persisted)
		if err != nil {
			return nil, err
		}
		for _, device := range devices {
			if device.DeviceRef == "" || device.TokenHash == "" {
				continue
			}
			service.devices[device.DeviceRef] = device
		}
	}

	return service, nil
}

// ResumeRemembered keeps an already-paired VexLife phone discoverable after Desktop restart.
// A producer with no remembered phone stays LAN-closed until the user opens pairing.
func (s *Service) ResumeRemembered() error {
	s.mu.Lock()
	remembered := len(s.devices) > 0
	s.mu.Unlock()

	if !remembered {
		return nil
	}
	return s.EnsureStarted()
}

// EnsureStarted realizes only the bounded TLS/mobile listener. It never widens the browser/admin mux.
func (s *Service) EnsureStarted() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", DefaultLANPort))
	if err != nil {
		s.mu.Unlock()
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", s.lanHealth)
	mux.HandleFunc("/v1/pairing/info", s.pairingInfo)
	mux.HandleFunc("/v1/pairing/request", s.pairingRequest)
	mux.HandleFunc("/v1/pairing/status", s.pairingRequestStatus)
	mux.HandleFunc("/v1/pairing/revoke-self", s.authorized(s.pairingRevokeSelf))
	mux.HandleFunc("/v1/catalog", s.authorized(s.mobileCatalog))
	mux.HandleFunc("/v1/media/", s.authorized(s.mobileMedia))

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	s.listener = listener
	s.server = server
	s.running = true

	displayName := s.displayName
	certPath := s.certPath
	keyPath := s.keyPath
	fingerprint := s.certificateSHA256
	s.mu.Unlock()

	mdns, err := zeroconf.Register(
		displayName,
		serviceType,
		"local.",
		DefaultLANPort,
		[]string{
			"bridge=vexstream-mobile/v1",
			"transport=https",
			"pairing=desktop-approval",
			"certificateSha256=" + fingerprint,
		},
		nil,
	)
	if err != nil {
		_ = listener.Close()
		s.mu.Lock()
		s.listener = nil
		s.server = nil
		s.running = false
		s.mu.Unlock()
		return fmt.Errorf("VexStream nearby discovery unavailable: %w", err)
	}

	s.mu.Lock()
	s.mdns = mdns
	s.mu.Unlock()

	go func() {
		if err := server.ServeTLS(listener, certPath, keyPath); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			// The loopback admin surface remains the place to report/retry pairing.
		}
	}()

	return nil
}

func (s *Service) Close() {
	s.mu.Lock()
	server := s.server
	mdns := s.mdns
	s.server = nil
	s.listener = nil
	s.mdns = nil
	s.running = false
	s.mu.Unlock()

	if mdns != nil {
		mdns.Shutdown()
	}
	if server != nil {
		_ = server.Close()
	}
}

// AdminHandler is mounted only on Desktop's loopback application mux.
// The /v1 catalog/media surface is served exclusively by the separate TLS listener.
func (s *Service) AdminHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/mobile/pairing":
			s.mobilePairingStatus(w, r)
		case "/api/mobile/pairing/start":
			s.mobilePairingStart(w, r)
		case "/api/mobile/pairing/decision":
			s.mobilePairingDecision(w, r)
		case "/api/mobile/pairing/revoke":
			s.mobilePairingRevoke(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func decodeJSON(r *http.Request, value any) error {
	return json.NewDecoder(r.Body).Decode(value)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
