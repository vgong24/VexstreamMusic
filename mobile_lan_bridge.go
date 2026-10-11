package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	mobileLANPort            = 18766
	mobileLANServiceType     = "_vexstream._tcp"
	mobilePairingWindow      = 5 * time.Minute
	mobilePairingRequestLife = 10 * time.Minute
	mobilePairingSchemaV1    = "vexstream.mobile.local-pairing/v1"
	mobilePairingSchema      = "vexstream.mobile.local-pairing/v2"
)

type mobilePairedDevice struct {
	DeviceRef        string `json:"deviceRef"`
	DisplayName      string `json:"displayName"`
	TokenHash        string `json:"tokenHash"`
	DevicePublicKey  string `json:"devicePublicKey,omitempty"`
	RemoteEnabled    bool   `json:"remoteEnabled,omitempty"`
	ApprovedAt       string `json:"approvedAt"`
}

type mobilePairingFile struct {
	SchemaVersion string               `json:"schemaVersion"`
	Devices       []mobilePairedDevice `json:"devices"`
}

type mobilePendingPairing struct {
	RequestRef       string
	DeviceRef        string
	DisplayName      string
	DevicePublicKey  string
	RequestedAt      time.Time
	ExpiresAt        time.Time
	State            string
	Token            string
}

type mobileLANBridgeState struct {
	mu                 sync.Mutex
	loaded             bool
	running            bool
	pairingWindowUntil time.Time
	pending            map[string]*mobilePendingPairing
	devices            map[string]mobilePairedDevice
	server             *http.Server
	listener           net.Listener
	mdns               *zeroconf.Server
	certPath           string
	keyPath                  string
	remoteGatewayKeyPath     string
	remoteGatewayPublicKey   string
	certificateSHA256        string
	pairingCode              string
	displayName              string
}

var mobileLANBridge = &mobileLANBridgeState{
	pending: map[string]*mobilePendingPairing{},
	devices: map[string]mobilePairedDevice{},
}

func initMobileLANBridge(a *App) error {
	if err := mobileLANBridge.load(); err != nil {
		return err
	}

	mobileLANBridge.mu.Lock()
	paired := len(mobileLANBridge.devices) > 0
	mobileLANBridge.mu.Unlock()

	if paired {
		if err := mobileLANBridge.ensureStarted(a); err != nil {
			return err
		}
		startMobileRemoteGateway(a)
	}
	return nil
}

func (b *mobileLANBridgeState) load() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.loaded {
		return nil
	}

	root := filepath.Join(appDataBase(), "mobile-bridge")
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}

	b.certPath = filepath.Join(root, "server-cert.pem")
	b.keyPath = filepath.Join(root, "server-key.pem")
	b.remoteGatewayKeyPath = filepath.Join(root, "remote-gateway-key")
	fingerprint, code, err := ensureMobileLANCertificate(b.certPath, b.keyPath)
	if err != nil {
		return err
	}
	remoteGatewayPublicKey, err := ensureMobileRemoteGatewayKey(b.remoteGatewayKeyPath)
	if err != nil {
		return err
	}
	b.certificateSHA256 = fingerprint
	b.pairingCode = code
	b.remoteGatewayPublicKey = remoteGatewayPublicKey

	host, _ := os.Hostname()
	host = strings.TrimSpace(host)
	if host == "" {
		host = "Windows PC"
	}
	b.displayName = "VexStream on " + host

	statePath := filepath.Join(root, "paired-devices.json")
	data, err := os.ReadFile(statePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(data) > 0 {
		var stored mobilePairingFile
		if err := json.Unmarshal(data, &stored); err != nil {
			return err
		}
		if stored.SchemaVersion != mobilePairingSchema && stored.SchemaVersion != mobilePairingSchemaV1 {
			return fmt.Errorf("unsupported mobile pairing schema %q", stored.SchemaVersion)
		}
		for _, device := range stored.Devices {
			if device.DeviceRef == "" || device.TokenHash == "" {
				continue
			}
			b.devices[device.DeviceRef] = device
		}
	}

	b.loaded = true
	return nil
}

func (b *mobileLANBridgeState) saveLocked() error {
	devices := make([]mobilePairedDevice, 0, len(b.devices))
	for _, device := range b.devices {
		devices = append(devices, device)
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].DisplayName < devices[j].DisplayName
	})

	payload, err := json.MarshalIndent(mobilePairingFile{
		SchemaVersion: mobilePairingSchema,
		Devices:       devices,
	}, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	path := filepath.Join(appDataBase(), "mobile-bridge", "paired-devices.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0600); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return os.Rename(tmp, path)
}

func (b *mobileLANBridgeState) ensureStarted(a *App) error {
	if err := b.load(); err != nil {
		return err
	}

	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return nil
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", mobileLANPort))
	if err != nil {
		b.mu.Unlock()
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", b.lanHealth)
	mux.HandleFunc("/v1/pairing/info", b.pairingInfo)
	mux.HandleFunc("/v1/pairing/request", b.pairingRequest)
	mux.HandleFunc("/v1/pairing/status", b.pairingRequestStatus)
	mux.HandleFunc("/v1/pairing/remote-upgrade", b.authorized(b.pairingRemoteUpgrade(a)))
	mux.HandleFunc("/v1/pairing/revoke-self", b.authorized(b.pairingRevokeSelf))
	mux.HandleFunc("/v1/catalog", b.authorized(a.mobileCatalog))
	mux.HandleFunc("/v1/media/", b.authorized(a.mobileMedia))

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	b.listener = ln
	b.server = server
	b.running = true
	displayName := b.displayName
	certPath := b.certPath
	keyPath := b.keyPath
	fingerprint := b.certificateSHA256
	b.mu.Unlock()

	mdns, mdnsErr := zeroconf.Register(
		displayName,
		mobileLANServiceType,
		"local.",
		mobileLANPort,
		[]string{
			"bridge=vexstream-mobile/v1",
			"transport=https",
			"pairing=desktop-approval",
			"certificateSha256=" + fingerprint,
		},
		nil,
	)
	if mdnsErr != nil {
		_ = ln.Close()
		b.mu.Lock()
		b.listener = nil
		b.server = nil
		b.running = false
		b.mu.Unlock()
		return fmt.Errorf("VexStream nearby discovery unavailable: %w", mdnsErr)
	}

	b.mu.Lock()
	b.mdns = mdns
	b.mu.Unlock()

	go func() {
		if err := server.ServeTLS(ln, certPath, keyPath); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "VexStream LAN mobile bridge stopped:", err)
		}
	}()

	return nil
}

func (b *mobileLANBridgeState) scheduleStopIfUnused(
	pairingWindowUntil time.Time,
) {
	delay := time.Until(pairingWindowUntil)
	if delay < 0 {
		delay = 0
	}

	go func() {
		timer := time.NewTimer(delay + 100*time.Millisecond)
		defer timer.Stop()
		<-timer.C
		b.stopIfUnused()
	}()
}

func (b *mobileLANBridgeState) stopIfUnused() {
	b.mu.Lock()
	b.cleanupPendingLocked()
	if !b.running ||
		len(b.devices) > 0 ||
		b.hasActivePendingLocked() ||
		time.Now().Before(b.pairingWindowUntil) {
		b.mu.Unlock()
		return
	}

	server := b.server
	mdns := b.mdns
	b.server = nil
	b.listener = nil
	b.mdns = nil
	b.running = false
	b.mu.Unlock()

	if mdns != nil {
		mdns.Shutdown()
	}
	if server != nil {
		_ = server.Close()
	}
}

func (b *mobileLANBridgeState) lanHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	b.mu.Lock()
	fingerprint := b.certificateSHA256
	displayName := b.displayName
	b.mu.Unlock()

	jsonOut(w, http.StatusOK, map[string]any{
		"ok":                     true,
		"bridge":                 "vexstream-mobile-lan/v1",
		"sourceVersion":          version,
		"displayName":            displayName,
		"certificateSha256":      fingerprint,
		"canonicalAuthorization": false,
	})
}

func (b *mobileLANBridgeState) pairingInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.cleanupPendingLocked()
	jsonOut(w, http.StatusOK, map[string]any{
		"displayName":        b.displayName,
		"certificateSha256":  b.certificateSHA256,
		"pairingCode":        b.pairingCode,
		"pairingWindowOpen":  time.Now().Before(b.pairingWindowUntil),
		"pairingWindowUntil": b.pairingWindowUntil.UTC().Format(time.RFC3339),
	})
}

func (b *mobileLANBridgeState) pairingRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var request struct {
		DeviceRef       string `json:"deviceRef"`
		DisplayName     string `json:"displayName"`
		DevicePublicKey string `json:"devicePublicKey"`
	}
	if err := decode(r, &request); err != nil {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid pairing request"})
		return
	}
	request.DeviceRef = strings.TrimSpace(request.DeviceRef)
	request.DisplayName = strings.TrimSpace(request.DisplayName)
	request.DevicePublicKey = strings.TrimSpace(request.DevicePublicKey)
	if request.DeviceRef == "" || request.DisplayName == "" || len(request.DisplayName) > 80 {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "deviceRef and displayName are required"})
		return
	}
	if err := validateMobileDevicePublicKey(request.DevicePublicKey); err != nil {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "valid devicePublicKey is required"})
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.cleanupPendingLocked()
	if !time.Now().Before(b.pairingWindowUntil) {
		jsonOut(w, http.StatusForbidden, map[string]string{"error": "pairing window is closed"})
		return
	}

	for _, pending := range b.pending {
		if pending.DeviceRef == request.DeviceRef && pending.State == "PENDING" {
			if pending.DevicePublicKey != request.DevicePublicKey {
				jsonOut(w, http.StatusConflict, map[string]string{"error": "device signing identity changed during pairing"})
				return
			}
			jsonOut(w, http.StatusOK, map[string]any{
				"requestRef": pending.RequestRef,
				"state":      pending.State,
			})
			return
		}
	}

	requestRef, err := randomRef("pair")
	if err != nil {
		jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "pairing request unavailable"})
		return
	}
	now := time.Now()
	expiresAt := now.Add(mobilePairingRequestLife)
	b.pending[requestRef] = &mobilePendingPairing{
		RequestRef:      requestRef,
		DeviceRef:       request.DeviceRef,
		DisplayName:     request.DisplayName,
		DevicePublicKey: request.DevicePublicKey,
		RequestedAt:     now,
		ExpiresAt:   expiresAt,
		State:       "PENDING",
	}
	b.scheduleStopIfUnused(expiresAt)

	jsonOut(w, http.StatusOK, map[string]any{
		"requestRef": requestRef,
		"state":      "PENDING",
	})
}

func (b *mobileLANBridgeState) pairingRequestStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	requestRef := strings.TrimSpace(r.URL.Query().Get("requestRef"))
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cleanupPendingLocked()
	pending := b.pending[requestRef]
	if pending == nil {
		jsonOut(w, http.StatusNotFound, map[string]string{"error": "pairing request not found"})
		return
	}

	response := map[string]any{
		"requestRef": pending.RequestRef,
		"state":      pending.State,
	}
	if pending.State == "APPROVED" && pending.Token != "" {
		response["token"] = pending.Token
		response["remoteHomeRef"] = b.remoteHomeRefLocked()
		response["remoteGatewayPublicKey"] = b.remoteGatewayPublicKey
		if relayURL := mobileRemoteRelayURL(); relayURL != "" {
			response["remoteRelayBaseUrl"] = relayURL
		}
		response["remoteAccessReady"] =
			pending.DevicePublicKey != "" &&
				mobileRemoteRelayURL() != ""
	}
	jsonOut(w, http.StatusOK, response)
}

func (b *mobileLANBridgeState) pairingRemoteUpgrade(a *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}

		deviceRef := b.deviceRefForRequest(r)
		if deviceRef == "" {
			jsonOut(w, http.StatusUnauthorized, map[string]string{"error": "paired VexLife device required"})
			return
		}

		var request struct {
			DevicePublicKey string `json:"devicePublicKey"`
		}
		if err := decode(r, &request); err != nil {
			jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid remote upgrade request"})
			return
		}
		request.DevicePublicKey = strings.TrimSpace(request.DevicePublicKey)
		if err := validateMobileDevicePublicKey(request.DevicePublicKey); err != nil {
			jsonOut(w, http.StatusBadRequest, map[string]string{"error": "valid devicePublicKey is required"})
			return
		}

		b.mu.Lock()
		device, ok := b.devices[deviceRef]
		if !ok {
			b.mu.Unlock()
			jsonOut(w, http.StatusUnauthorized, map[string]string{"error": "paired VexLife device required"})
			return
		}
		if device.DevicePublicKey != "" && device.DevicePublicKey != request.DevicePublicKey {
			b.mu.Unlock()
			jsonOut(w, http.StatusConflict, map[string]string{"error": "device signing identity changed; re-pair explicitly"})
			return
		}
		device.DevicePublicKey = request.DevicePublicKey
		device.RemoteEnabled = true
		b.devices[deviceRef] = device
		if err := b.saveLocked(); err != nil {
			b.mu.Unlock()
			jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "unable to save remote device identity"})
			return
		}
		homeRef := b.remoteHomeRefLocked()
		gatewayPublicKey := b.remoteGatewayPublicKey
		b.mu.Unlock()

		relayURL := mobileRemoteRelayURL()
		jsonOut(w, http.StatusOK, map[string]any{
			"remoteHomeRef":          homeRef,
			"remoteGatewayPublicKey": gatewayPublicKey,
			"remoteRelayBaseUrl":     relayURL,
			"remoteAccessReady":      relayURL != "",
		})
		go startMobileRemoteGateway(a)
	}
}

func (b *mobileLANBridgeState) pairingRevokeSelf(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	deviceRef := b.deviceRefForRequest(r)
	if deviceRef == "" {
		jsonOut(w, http.StatusUnauthorized, map[string]string{"error": "paired VexLife device required"})
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.devices, deviceRef)
	if err := b.saveLocked(); err != nil {
		jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "unable to save device removal"})
		return
	}

	jsonOut(w, http.StatusOK, map[string]any{"ok": true})
	go b.stopIfUnused()
}

func (b *mobileLANBridgeState) authorized(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !b.authorizedRequest(r) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			jsonOut(w, http.StatusUnauthorized, map[string]string{"error": "paired VexLife device required"})
			return
		}
		next(w, r)
	}
}

func (b *mobileLANBridgeState) authorizedRequest(r *http.Request) bool {
	return b.deviceRefForRequest(r) != ""
}

func (b *mobileLANBridgeState) deviceRefForRequest(r *http.Request) string {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(raw, "Bearer ") {
		return ""
	}
	token := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	candidate := hex.EncodeToString(sum[:])

	b.mu.Lock()
	defer b.mu.Unlock()
	for deviceRef, device := range b.devices {
		if len(candidate) == len(device.TokenHash) && subtle.ConstantTimeCompare([]byte(candidate), []byte(device.TokenHash)) == 1 {
			return deviceRef
		}
	}
	return ""
}

func (a *App) mobilePairingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	if err := mobileLANBridge.load(); err != nil {
		jsonOut(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	mobileLANBridge.mu.Lock()
	defer mobileLANBridge.mu.Unlock()
	mobileLANBridge.cleanupPendingLocked()

	pending := make([]map[string]any, 0)
	for _, request := range mobileLANBridge.pending {
		if request.State != "PENDING" {
			continue
		}
		pending = append(pending, map[string]any{
			"requestRef":  request.RequestRef,
			"displayName": request.DisplayName,
			"requestedAt": request.RequestedAt.UTC().Format(time.RFC3339),
		})
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i]["requestedAt"].(string) < pending[j]["requestedAt"].(string)
	})

	paired := make([]map[string]any, 0, len(mobileLANBridge.devices))
	for _, device := range mobileLANBridge.devices {
		paired = append(paired, map[string]any{
			"deviceRef":     device.DeviceRef,
			"displayName":   device.DisplayName,
			"remoteEnabled": device.RemoteEnabled && device.DevicePublicKey != "",
			"approvedAt":    device.ApprovedAt,
		})
	}
	sort.Slice(paired, func(i, j int) bool {
		return paired[i]["displayName"].(string) < paired[j]["displayName"].(string)
	})

	jsonOut(w, http.StatusOK, map[string]any{
		"running":              mobileLANBridge.running,
		"displayName":          mobileLANBridge.displayName,
		"pairingCode":          mobileLANBridge.pairingCode,
		"pairingWindowOpen":    time.Now().Before(mobileLANBridge.pairingWindowUntil),
		"pairingWindowUntil":   mobileLANBridge.pairingWindowUntil.UTC().Format(time.RFC3339),
		"pending":              pending,
		"paired":               paired,
		"canonicalAuthority":   false,
		"remoteRelayConfigured": mobileRemoteRelayURL() != "",
		"transportDescription": "Local pairing plus bounded remote VexStream gateway candidate; canonical Home/Safety authority is not implied.",
	})
}

func (a *App) mobilePairingStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if err := mobileLANBridge.ensureStarted(a); err != nil {
		jsonOut(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	mobileLANBridge.mu.Lock()
	mobileLANBridge.pairingWindowUntil = time.Now().Add(mobilePairingWindow)
	until := mobileLANBridge.pairingWindowUntil
	code := mobileLANBridge.pairingCode
	mobileLANBridge.mu.Unlock()
	mobileLANBridge.scheduleStopIfUnused(until)
	jsonOut(w, http.StatusOK, map[string]any{
		"ok":          true,
		"pairingCode": code,
		"expiresAt":   until.UTC().Format(time.RFC3339),
	})
}

func (a *App) mobilePairingDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	var request struct {
		RequestRef string `json:"requestRef"`
		Decision   string `json:"decision"`
	}
	if err := decode(r, &request); err != nil {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid pairing decision"})
		return
	}

	mobileLANBridge.mu.Lock()
	defer mobileLANBridge.mu.Unlock()
	mobileLANBridge.cleanupPendingLocked()
	pending := mobileLANBridge.pending[strings.TrimSpace(request.RequestRef)]
	if pending == nil || pending.State != "PENDING" {
		jsonOut(w, http.StatusNotFound, map[string]string{"error": "pending pairing request not found"})
		return
	}

	switch strings.ToUpper(strings.TrimSpace(request.Decision)) {
	case "DENY":
		pending.State = "DENIED"
		jsonOut(w, http.StatusOK, map[string]any{"ok": true, "state": "DENIED"})
		go mobileLANBridge.stopIfUnused()
		return
	case "APPROVE":
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "unable to form device token"})
			return
		}
		token := base64.RawURLEncoding.EncodeToString(tokenBytes)
		tokenHash := sha256.Sum256([]byte(token))
		approvedAt := time.Now().UTC().Format(time.RFC3339)
		mobileLANBridge.devices[pending.DeviceRef] = mobilePairedDevice{
			DeviceRef:       pending.DeviceRef,
			DisplayName:     pending.DisplayName,
			TokenHash:       hex.EncodeToString(tokenHash[:]),
			DevicePublicKey: pending.DevicePublicKey,
			RemoteEnabled:   pending.DevicePublicKey != "",
			ApprovedAt:      approvedAt,
		}
		pending.State = "APPROVED"
		pending.Token = token
		if err := mobileLANBridge.saveLocked(); err != nil {
			delete(mobileLANBridge.devices, pending.DeviceRef)
			pending.State = "PENDING"
			pending.Token = ""
			jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "unable to save paired device"})
			return
		}
		jsonOut(w, http.StatusOK, map[string]any{"ok": true, "state": "APPROVED"})
		go startMobileRemoteGateway(a)
		return
	default:
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "decision must be APPROVE or DENY"})
	}
}

func (a *App) mobilePairingRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	var request struct {
		DeviceRef string `json:"deviceRef"`
	}
	if err := decode(r, &request); err != nil {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid revoke request"})
		return
	}

	mobileLANBridge.mu.Lock()
	defer mobileLANBridge.mu.Unlock()
	deviceRef := strings.TrimSpace(request.DeviceRef)
	if _, ok := mobileLANBridge.devices[deviceRef]; !ok {
		jsonOut(w, http.StatusNotFound, map[string]string{"error": "paired device not found"})
		return
	}
	delete(mobileLANBridge.devices, deviceRef)
	if err := mobileLANBridge.saveLocked(); err != nil {
		jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "unable to save device removal"})
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{"ok": true})
	go mobileLANBridge.stopIfUnused()
}

func (b *mobileLANBridgeState) hasActivePendingLocked() bool {
	now := time.Now()
	for _, pending := range b.pending {
		if pending.State == "PENDING" && now.Before(pending.ExpiresAt) {
			return true
		}
	}
	return false
}

func (b *mobileLANBridgeState) cleanupPendingLocked() {
	now := time.Now()
	for ref, pending := range b.pending {
		if now.After(pending.ExpiresAt) {
			delete(b.pending, ref)
		}
	}
}

func ensureMobileLANCertificate(certPath, keyPath string) (string, string, error) {
	if certPEM, err := os.ReadFile(certPath); err == nil {
		block, _ := pem.Decode(certPEM)
		if block != nil {
			cert, parseErr := x509.ParseCertificate(block.Bytes)
			if parseErr == nil {
				sum := sha256.Sum256(cert.Raw)
				return hex.EncodeToString(sum[:]), pairingCodeFromDigest(sum), nil
			}
		}
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return "", "", err
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "VexStream Local Mobile Bridge",
			Organization: []string{"Vextreme"},
		},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"vexstream.local"},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:]), pairingCodeFromDigest(sum), nil
}

func pairingCodeFromDigest(sum [32]byte) string {
	value := binary.BigEndian.Uint32(sum[:4]) % 1000000
	return fmt.Sprintf("%06d", value)
}

func randomRef(prefix string) (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return prefix + "." + hex.EncodeToString(raw), nil
}
