package mobile

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	pairingWindow        = 5 * time.Minute
	pairingRequestLife   = 10 * time.Minute
	pairingSchemaV1      = "vexstream.mobile.local-pairing/v1"
	pairingSchemaCurrent = "vexstream.mobile.local-pairing/v2"
)

type pairedDevice struct {
	DeviceRef       string `json:"deviceRef"`
	DisplayName     string `json:"displayName"`
	TokenHash       string `json:"tokenHash"`
	DevicePublicKey string `json:"devicePublicKey,omitempty"`
	RemoteEnabled   bool   `json:"remoteEnabled,omitempty"`
	ApprovedAt      string `json:"approvedAt"`
}

type pairingFile struct {
	SchemaVersion string         `json:"schemaVersion"`
	Devices       []pairedDevice `json:"devices"`
}

type pendingPairing struct {
	RequestRef  string
	DeviceRef   string
	DisplayName string
	RequestedAt time.Time
	ExpiresAt   time.Time
	State       string
	Token       string
}

func (s *Service) lanHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	s.mu.Lock()
	fingerprint := s.certificateSHA256
	displayName := s.displayName
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                     true,
		"bridge":                 "vexstream-mobile-lan/v1",
		"sourceVersion":          s.version,
		"displayName":            displayName,
		"certificateSha256":      fingerprint,
		"canonicalAuthorization": false,
	})
}

func (s *Service) pairingInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupPendingLocked()

	writeJSON(w, http.StatusOK, map[string]any{
		"displayName":        s.displayName,
		"certificateSha256":  s.certificateSHA256,
		"pairingCode":        s.pairingCode,
		"pairingWindowOpen":  time.Now().Before(s.pairingWindowUntil),
		"pairingWindowUntil": s.pairingWindowUntil.UTC().Format(time.RFC3339),
	})
}

func (s *Service) pairingRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var request struct {
		DeviceRef   string `json:"deviceRef"`
		DisplayName string `json:"displayName"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid pairing request",
		})
		return
	}

	request.DeviceRef = strings.TrimSpace(request.DeviceRef)
	request.DisplayName = strings.TrimSpace(request.DisplayName)
	if request.DeviceRef == "" ||
		request.DisplayName == "" ||
		len(request.DisplayName) > 80 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "deviceRef and displayName are required",
		})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupPendingLocked()

	if !time.Now().Before(s.pairingWindowUntil) {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "pairing window is closed",
		})
		return
	}

	for _, pending := range s.pending {
		if pending.DeviceRef == request.DeviceRef &&
			pending.State == "PENDING" {
			writeJSON(w, http.StatusOK, map[string]any{
				"requestRef": pending.RequestRef,
				"state":      pending.State,
			})
			return
		}
	}

	requestRef, err := randomRef("pair")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "pairing request unavailable",
		})
		return
	}

	now := time.Now()
	expiresAt := now.Add(pairingRequestLife)
	s.pending[requestRef] = &pendingPairing{
		RequestRef:  requestRef,
		DeviceRef:   request.DeviceRef,
		DisplayName: request.DisplayName,
		RequestedAt: now,
		ExpiresAt:   expiresAt,
		State:       "PENDING",
	}
	s.scheduleStopIfUnused(expiresAt)

	writeJSON(w, http.StatusOK, map[string]any{
		"requestRef": requestRef,
		"state":      "PENDING",
	})
}

func (s *Service) pairingRequestStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	requestRef := strings.TrimSpace(r.URL.Query().Get("requestRef"))

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupPendingLocked()

	pending := s.pending[requestRef]
	if pending == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "pairing request not found",
		})
		return
	}

	response := map[string]any{
		"requestRef": pending.RequestRef,
		"state":      pending.State,
	}
	if pending.State == "APPROVED" && pending.Token != "" {
		response["token"] = pending.Token
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Service) pairingRevokeSelf(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	deviceRef := s.deviceRefForRequest(r)
	if deviceRef == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "paired VexLife device required",
		})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.devices, deviceRef)
	if err := s.saveLocked(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "unable to save device removal",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	go s.stopIfUnused()
}

func (s *Service) authorized(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authorizedRequest(r) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "paired VexLife device required",
			})
			return
		}
		next(w, r)
	}
}

func (s *Service) authorizedRequest(r *http.Request) bool {
	return s.deviceRefForRequest(r) != ""
}

func (s *Service) deviceRefForRequest(r *http.Request) string {
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

	s.mu.Lock()
	defer s.mu.Unlock()
	for deviceRef, device := range s.devices {
		if len(candidate) == len(device.TokenHash) &&
			subtle.ConstantTimeCompare(
				[]byte(candidate),
				[]byte(device.TokenHash),
			) == 1 {
			return deviceRef
		}
	}
	return ""
}

func (s *Service) mobilePairingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupPendingLocked()

	pending := make([]map[string]any, 0)
	for _, request := range s.pending {
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
		return pending[i]["requestedAt"].(string) <
			pending[j]["requestedAt"].(string)
	})

	paired := make([]map[string]any, 0, len(s.devices))
	for _, device := range s.devices {
		paired = append(paired, map[string]any{
			"deviceRef":   device.DeviceRef,
			"displayName": device.DisplayName,
			"approvedAt":  device.ApprovedAt,
		})
	}
	sort.Slice(paired, func(i, j int) bool {
		return paired[i]["displayName"].(string) <
			paired[j]["displayName"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"running":              s.running,
		"displayName":          s.displayName,
		"pairingCode":          s.pairingCode,
		"pairingWindowOpen":    time.Now().Before(s.pairingWindowUntil),
		"pairingWindowUntil":   s.pairingWindowUntil.UTC().Format(time.RFC3339),
		"pending":              pending,
		"paired":               paired,
		"canonicalAuthority":   false,
		"transportDescription": "Local VexStream pairing candidate; Home/Safety authority is not implied.",
	})
}

func (s *Service) mobilePairingStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	if err := s.EnsureStarted(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	s.mu.Lock()
	s.pairingWindowUntil = time.Now().Add(pairingWindow)
	until := s.pairingWindowUntil
	code := s.pairingCode
	s.mu.Unlock()

	s.scheduleStopIfUnused(until)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"pairingCode": code,
		"expiresAt":   until.UTC().Format(time.RFC3339),
	})
}

func (s *Service) mobilePairingDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var request struct {
		RequestRef string `json:"requestRef"`
		Decision   string `json:"decision"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid pairing decision",
		})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupPendingLocked()

	pending := s.pending[strings.TrimSpace(request.RequestRef)]
	if pending == nil || pending.State != "PENDING" {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "pending pairing request not found",
		})
		return
	}

	switch strings.ToUpper(strings.TrimSpace(request.Decision)) {
	case "DENY":
		pending.State = "DENIED"
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    true,
			"state": "DENIED",
		})
		go s.stopIfUnused()
		return

	case "APPROVE":
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "unable to form device token",
			})
			return
		}

		token := base64.RawURLEncoding.EncodeToString(tokenBytes)
		tokenHash := sha256.Sum256([]byte(token))
		approvedAt := time.Now().UTC().Format(time.RFC3339)

		s.devices[pending.DeviceRef] = pairedDevice{
			DeviceRef:   pending.DeviceRef,
			DisplayName: pending.DisplayName,
			TokenHash:   hex.EncodeToString(tokenHash[:]),
			ApprovedAt:  approvedAt,
		}
		pending.State = "APPROVED"
		pending.Token = token

		if err := s.saveLocked(); err != nil {
			delete(s.devices, pending.DeviceRef)
			pending.State = "PENDING"
			pending.Token = ""
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "unable to save paired device",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    true,
			"state": "APPROVED",
		})
		return

	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "decision must be APPROVE or DENY",
		})
	}
}

func (s *Service) mobilePairingRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var request struct {
		DeviceRef string `json:"deviceRef"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid revoke request",
		})
		return
	}

	deviceRef := strings.TrimSpace(request.DeviceRef)

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.devices[deviceRef]; !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "paired device not found",
		})
		return
	}

	delete(s.devices, deviceRef)
	if err := s.saveLocked(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "unable to save device removal",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	go s.stopIfUnused()
}

func (s *Service) saveLocked() error {
	payload, err := encodeRememberedDevices(s.devices)
	if err != nil {
		return err
	}

	path := filepath.Join(s.root, "paired-devices.json")
	temp := path + ".tmp"
	if err := os.WriteFile(temp, payload, 0o600); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return os.Rename(temp, path)
}

func decodeRememberedDevices(data []byte) ([]pairedDevice, error) {
	var stored pairingFile
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}

	switch stored.SchemaVersion {
	case pairingSchemaV1, pairingSchemaCurrent:
		return stored.Devices, nil
	default:
		return nil, fmt.Errorf(
			"unsupported mobile pairing schema %q",
			stored.SchemaVersion,
		)
	}
}

func encodeRememberedDevices(
	devicesByRef map[string]pairedDevice,
) ([]byte, error) {
	devices := make([]pairedDevice, 0, len(devicesByRef))
	for _, device := range devicesByRef {
		devices = append(devices, device)
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].DisplayName < devices[j].DisplayName
	})

	payload, err := json.MarshalIndent(pairingFile{
		SchemaVersion: pairingSchemaCurrent,
		Devices:       devices,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func (s *Service) scheduleStopIfUnused(until time.Time) {
	delay := time.Until(until)
	if delay < 0 {
		delay = 0
	}

	go func() {
		timer := time.NewTimer(delay + 100*time.Millisecond)
		defer timer.Stop()
		<-timer.C
		s.stopIfUnused()
	}()
}

func (s *Service) stopIfUnused() {
	s.mu.Lock()
	s.cleanupPendingLocked()

	if !s.running ||
		len(s.devices) > 0 ||
		s.hasActivePendingLocked() ||
		time.Now().Before(s.pairingWindowUntil) {
		s.mu.Unlock()
		return
	}

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

func (s *Service) hasActivePendingLocked() bool {
	now := time.Now()
	for _, pending := range s.pending {
		if pending.State == "PENDING" &&
			now.Before(pending.ExpiresAt) {
			return true
		}
	}
	return false
}

func (s *Service) cleanupPendingLocked() {
	now := time.Now()
	for ref, pending := range s.pending {
		if now.After(pending.ExpiresAt) {
			delete(s.pending, ref)
		}
	}
}

func randomRef(prefix string) (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return prefix + "." + hex.EncodeToString(raw), nil
}
