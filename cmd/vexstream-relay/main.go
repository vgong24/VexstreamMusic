package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	relaySchema          = "vexstream.remote-relay/v1"
	hostSessionTTL       = 90 * time.Second
	longPollDuration     = 25 * time.Second
	envelopeMaxBytes     = 2 * 1024 * 1024
	maxClockSkew         = 2 * time.Minute
	maxOutstandingPerHost = 64
)

type relayDevice struct {
	DeviceRef       string `json:"deviceRef"`
	DevicePublicKey string `json:"devicePublicKey"`
}

type hostRegisterRequest struct {
	SchemaVersion  string        `json:"schemaVersion"`
	HomeRef        string        `json:"homeRef"`
	CertificateDER string        `json:"certificateDer"`
	Nonce          string        `json:"nonce"`
	IssuedAt       string        `json:"issuedAt"`
	Devices        []relayDevice `json:"devices"`
	Signature      string        `json:"signature"`
}

type remoteRequestEnvelope struct {
	SchemaVersion      string `json:"schemaVersion"`
	HomeRef            string `json:"homeRef"`
	DeviceRef          string `json:"deviceRef"`
	RequestRef         string `json:"requestRef"`
	IssuedAt           string `json:"issuedAt"`
	EphemeralPublicKey string `json:"ephemeralPublicKey"`
	Nonce              string `json:"nonce"`
	Ciphertext         string `json:"ciphertext"`
	Signature          string `json:"signature"`
}

type remotePollRequest struct {
	SchemaVersion string `json:"schemaVersion"`
	HomeRef       string `json:"homeRef"`
	DeviceRef     string `json:"deviceRef"`
	RequestRef    string `json:"requestRef"`
	IssuedAt      string `json:"issuedAt"`
	Signature     string `json:"signature"`
}

type remoteResponseEnvelope struct {
	SchemaVersion string `json:"schemaVersion"`
	HomeRef       string `json:"homeRef"`
	RequestRef    string `json:"requestRef"`
	Nonce         string `json:"nonce"`
	Ciphertext    string `json:"ciphertext"`
}

type outstandingRequest struct {
	deviceRef string
	expiresAt time.Time
}

type hostSession struct {
	homeRef     string
	token       string
	expiresAt   time.Time
	devices     map[string]*ecdsa.PublicKey
	requests    chan remoteRequestEnvelope
	outstanding map[string]outstandingRequest
	responses   map[string]remoteResponseEnvelope
	responseWaiters map[string]chan struct{}
}

type relayState struct {
	mu    sync.Mutex
	hosts map[string]*hostSession
	now   func() time.Time
}

func newRelayState() *relayState {
	return &relayState{
		hosts: make(map[string]*hostSession),
		now:   time.Now,
	}
}

func main() {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	state := newRelayState()
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           state.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("VexStream remote relay listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}

func (s *relayState) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", s.health)
	mux.HandleFunc("/v1/host/register", s.hostRegister)
	mux.HandleFunc("/v1/host/heartbeat", s.hostHeartbeat)
	mux.HandleFunc("/v1/host/requests", s.hostRequests)
	mux.HandleFunc("/v1/host/respond", s.hostRespond)
	mux.HandleFunc("/v1/remote/request", s.remoteRequest)
	mux.HandleFunc("/v1/remote/response", s.remoteResponse)
	return http.MaxBytesHandler(mux, envelopeMaxBytes)
}

func (s *relayState) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	jsonOut(w, http.StatusOK, map[string]any{
		"ok":     true,
		"schema": relaySchema,
		"storesMedia": false,
		"storesCredentials": false,
	})
}

func (s *relayState) hostRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var request hostRegisterRequest
	if err := decodeJSON(r, &request); err != nil {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid registration"})
		return
	}
	if request.SchemaVersion != relaySchema || !freshTimestamp(request.IssuedAt, s.now(), maxClockSkew) {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "stale or unsupported registration"})
		return
	}

	certDER, err := decodeRawBase64(request.CertificateDER)
	if err != nil {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid certificate"})
		return
	}
	certificate, err := x509.ParseCertificate(certDER)
	if err != nil {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid certificate"})
		return
	}
	fingerprint := sha256.Sum256(certificate.Raw)
	expectedHomeRef := "vexstream.home." + hex.EncodeToString(fingerprint[:])
	if request.HomeRef != expectedHomeRef {
		jsonOut(w, http.StatusForbidden, map[string]string{"error": "home identity mismatch"})
		return
	}
	certKey, ok := certificate.PublicKey.(*ecdsa.PublicKey)
	if !ok || certKey.Curve.Params().Name != "P-256" {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "unsupported home key"})
		return
	}

	devices, digest, err := parseDevices(request.Devices)
	if err != nil {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid device set"})
		return
	}
	if !verifySignature(
		certKey,
		registrationCanonical(
			request.HomeRef,
			request.Nonce,
			request.IssuedAt,
			digest,
		),
		request.Signature,
	) {
		jsonOut(w, http.StatusForbidden, map[string]string{"error": "registration signature rejected"})
		return
	}

	sessionToken, err := randomToken(32)
	if err != nil {
		jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "session unavailable"})
		return
	}

	s.mu.Lock()
	old := s.hosts[request.HomeRef]
	session := &hostSession{
		homeRef:         request.HomeRef,
		token:           sessionToken,
		expiresAt:       s.now().Add(hostSessionTTL),
		devices:         devices,
		requests:        make(chan remoteRequestEnvelope, maxOutstandingPerHost),
		outstanding:     make(map[string]outstandingRequest),
		responses:       make(map[string]remoteResponseEnvelope),
		responseWaiters: make(map[string]chan struct{}),
	}
	if old != nil && s.now().Before(old.expiresAt) {
		// A legitimate re-registration from the same certificate refreshes routing without silently
		// dropping work already admitted by that live Home session.
		session.requests = old.requests
		session.outstanding = old.outstanding
		session.responses = old.responses
		session.responseWaiters = old.responseWaiters
	}
	s.hosts[request.HomeRef] = session
	s.cleanupLocked()
	s.mu.Unlock()

	jsonOut(w, http.StatusOK, map[string]any{
		"schemaVersion": relaySchema,
		"sessionToken":  sessionToken,
		"expiresInSeconds": int(hostSessionTTL.Seconds()),
	})
}

func (s *relayState) hostHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	homeRef := strings.TrimSpace(r.URL.Query().Get("homeRef"))

	s.mu.Lock()
	session := s.authorizedHostLocked(homeRef, bearerToken(r))
	if session != nil {
		session.expiresAt = s.now().Add(hostSessionTTL)
	}
	s.mu.Unlock()

	if session == nil {
		jsonOut(w, http.StatusUnauthorized, map[string]string{"error": "host session unavailable"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *relayState) hostRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	homeRef := strings.TrimSpace(r.URL.Query().Get("homeRef"))

	s.mu.Lock()
	session := s.authorizedHostLocked(homeRef, bearerToken(r))
	if session != nil {
		session.expiresAt = s.now().Add(hostSessionTTL)
	}
	var ch <-chan remoteRequestEnvelope
	if session != nil {
		ch = session.requests
	}
	s.mu.Unlock()

	if session == nil {
		jsonOut(w, http.StatusUnauthorized, map[string]string{"error": "host session unavailable"})
		return
	}

	select {
	case request := <-ch:
		jsonOut(w, http.StatusOK, request)
	case <-time.After(longPollDuration):
		w.WriteHeader(http.StatusNoContent)
	case <-r.Context().Done():
		return
	}
}

func (s *relayState) hostRespond(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var response remoteResponseEnvelope
	if err := decodeJSON(r, &response); err != nil || response.SchemaVersion != relaySchema {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid response envelope"})
		return
	}
	if len(response.Ciphertext) > envelopeMaxBytes || response.RequestRef == "" || response.Nonce == "" {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid response envelope"})
		return
	}

	s.mu.Lock()
	session := s.authorizedHostLocked(response.HomeRef, bearerToken(r))
	if session == nil {
		s.mu.Unlock()
		jsonOut(w, http.StatusUnauthorized, map[string]string{"error": "host session unavailable"})
		return
	}
	outstanding, ok := session.outstanding[response.RequestRef]
	if !ok || s.now().After(outstanding.expiresAt) {
		delete(session.outstanding, response.RequestRef)
		s.mu.Unlock()
		jsonOut(w, http.StatusGone, map[string]string{"error": "request is no longer active"})
		return
	}
	session.responses[response.RequestRef] = response
	if waiter := session.responseWaiters[response.RequestRef]; waiter != nil {
		close(waiter)
		delete(session.responseWaiters, response.RequestRef)
	}
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *relayState) remoteRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request remoteRequestEnvelope
	if err := decodeJSON(r, &request); err != nil {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid request envelope"})
		return
	}
	if request.SchemaVersion != relaySchema || !freshTimestamp(request.IssuedAt, s.now(), maxClockSkew) {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "stale or unsupported request"})
		return
	}
	if request.HomeRef == "" || request.DeviceRef == "" || request.RequestRef == "" ||
		request.EphemeralPublicKey == "" || request.Nonce == "" || request.Ciphertext == "" {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "incomplete request envelope"})
		return
	}

	s.mu.Lock()
	s.cleanupLocked()
	session := s.hosts[request.HomeRef]
	if session == nil || !s.now().Before(session.expiresAt) {
		s.mu.Unlock()
		jsonOut(w, http.StatusServiceUnavailable, map[string]string{"error": "home is offline"})
		return
	}
	deviceKey := session.devices[request.DeviceRef]
	if deviceKey == nil {
		s.mu.Unlock()
		jsonOut(w, http.StatusForbidden, map[string]string{"error": "device is not enrolled"})
		return
	}
	if _, exists := session.outstanding[request.RequestRef]; exists ||
		len(session.outstanding) >= maxOutstandingPerHost {
		s.mu.Unlock()
		jsonOut(w, http.StatusTooManyRequests, map[string]string{"error": "request capacity unavailable"})
		return
	}
	s.mu.Unlock()

	if !verifySignature(
		deviceKey,
		remoteRequestCanonical(request),
		request.Signature,
	) {
		jsonOut(w, http.StatusForbidden, map[string]string{"error": "device signature rejected"})
		return
	}

	s.mu.Lock()
	session = s.hosts[request.HomeRef]
	if session == nil || !s.now().Before(session.expiresAt) {
		s.mu.Unlock()
		jsonOut(w, http.StatusServiceUnavailable, map[string]string{"error": "home is offline"})
		return
	}
	session.outstanding[request.RequestRef] = outstandingRequest{
		deviceRef: request.DeviceRef,
		expiresAt: s.now().Add(2 * time.Minute),
	}
	requestQueue := session.requests
	s.mu.Unlock()

	select {
	case requestQueue <- request:
		jsonOut(w, http.StatusAccepted, map[string]any{
			"schemaVersion": relaySchema,
			"requestRef":    request.RequestRef,
			"state":         "QUEUED",
		})
	default:
		s.mu.Lock()
		if current := s.hosts[request.HomeRef]; current != nil {
			delete(current.outstanding, request.RequestRef)
		}
		s.mu.Unlock()
		jsonOut(w, http.StatusTooManyRequests, map[string]string{"error": "home request queue is full"})
	}
}

func (s *relayState) remoteResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var poll remotePollRequest
	if err := decodeJSON(r, &poll); err != nil || poll.SchemaVersion != relaySchema ||
		!freshTimestamp(poll.IssuedAt, s.now(), maxClockSkew) {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "invalid response poll"})
		return
	}

	s.mu.Lock()
	s.cleanupLocked()
	session := s.hosts[poll.HomeRef]
	if session == nil || !s.now().Before(session.expiresAt) {
		s.mu.Unlock()
		jsonOut(w, http.StatusServiceUnavailable, map[string]string{"error": "home is offline"})
		return
	}
	deviceKey := session.devices[poll.DeviceRef]
	outstanding, exists := session.outstanding[poll.RequestRef]
	s.mu.Unlock()

	if deviceKey == nil || !exists || outstanding.deviceRef != poll.DeviceRef {
		jsonOut(w, http.StatusForbidden, map[string]string{"error": "response is not available to this device"})
		return
	}
	if !verifySignature(deviceKey, remotePollCanonical(poll), poll.Signature) {
		jsonOut(w, http.StatusForbidden, map[string]string{"error": "device signature rejected"})
		return
	}

	for {
		s.mu.Lock()
		session = s.hosts[poll.HomeRef]
		if session == nil || !s.now().Before(session.expiresAt) {
			s.mu.Unlock()
			jsonOut(w, http.StatusServiceUnavailable, map[string]string{"error": "home is offline"})
			return
		}
		if response, ok := session.responses[poll.RequestRef]; ok {
			delete(session.responses, poll.RequestRef)
			delete(session.outstanding, poll.RequestRef)
			delete(session.responseWaiters, poll.RequestRef)
			s.mu.Unlock()
			jsonOut(w, http.StatusOK, response)
			return
		}
		waiter := session.responseWaiters[poll.RequestRef]
		if waiter == nil {
			waiter = make(chan struct{})
			session.responseWaiters[poll.RequestRef] = waiter
		}
		s.mu.Unlock()

		select {
		case <-waiter:
			continue
		case <-time.After(longPollDuration):
			jsonOut(w, http.StatusAccepted, map[string]any{
				"schemaVersion": relaySchema,
				"requestRef":    poll.RequestRef,
				"state":         "PENDING",
			})
			return
		case <-r.Context().Done():
			return
		}
	}
}

func (s *relayState) authorizedHostLocked(homeRef, token string) *hostSession {
	session := s.hosts[homeRef]
	if session == nil || !s.now().Before(session.expiresAt) || token == "" {
		return nil
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(session.token)) != 1 {
		return nil
	}
	return session
}

func (s *relayState) cleanupLocked() {
	now := s.now()
	for homeRef, session := range s.hosts {
		for requestRef, outstanding := range session.outstanding {
			if !now.Before(outstanding.expiresAt) {
				delete(session.outstanding, requestRef)
				delete(session.responses, requestRef)
				if waiter := session.responseWaiters[requestRef]; waiter != nil {
					close(waiter)
					delete(session.responseWaiters, requestRef)
				}
			}
		}
		if !now.Before(session.expiresAt) && len(session.outstanding) == 0 {
			delete(s.hosts, homeRef)
		}
	}
}

func parseDevices(input []relayDevice) (map[string]*ecdsa.PublicKey, string, error) {
	if len(input) == 0 || len(input) > 64 {
		return nil, "", errors.New("invalid device count")
	}
	devices := make(map[string]*ecdsa.PublicKey, len(input))
	canonical := append([]relayDevice(nil), input...)
	sort.Slice(canonical, func(i, j int) bool {
		return canonical[i].DeviceRef < canonical[j].DeviceRef
	})
	var digestInput strings.Builder
	for _, device := range canonical {
		if device.DeviceRef == "" || device.DevicePublicKey == "" {
			return nil, "", errors.New("incomplete device")
		}
		if _, duplicate := devices[device.DeviceRef]; duplicate {
			return nil, "", errors.New("duplicate device")
		}
		der, err := decodeRawBase64(device.DevicePublicKey)
		if err != nil {
			return nil, "", err
		}
		parsed, err := x509.ParsePKIXPublicKey(der)
		if err != nil {
			return nil, "", err
		}
		key, ok := parsed.(*ecdsa.PublicKey)
		if !ok || key.Curve.Params().Name != "P-256" {
			return nil, "", errors.New("device key must be ECDSA P-256")
		}
		devices[device.DeviceRef] = key
		digestInput.WriteString(device.DeviceRef)
		digestInput.WriteByte('\n')
		digestInput.WriteString(device.DevicePublicKey)
		digestInput.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(digestInput.String()))
	return devices, hex.EncodeToString(sum[:]), nil
}

func registrationCanonical(homeRef, nonce, issuedAt, deviceDigest string) []byte {
	return []byte(strings.Join([]string{
		"VEXSTREAM_REMOTE_REGISTER_V1",
		homeRef,
		nonce,
		issuedAt,
		deviceDigest,
	}, "\n"))
}

func remoteRequestCanonical(request remoteRequestEnvelope) []byte {
	sum := sha256.Sum256([]byte(request.Ciphertext))
	return []byte(strings.Join([]string{
		"VEXSTREAM_REMOTE_REQUEST_V1",
		request.HomeRef,
		request.DeviceRef,
		request.RequestRef,
		request.IssuedAt,
		request.EphemeralPublicKey,
		request.Nonce,
		hex.EncodeToString(sum[:]),
	}, "\n"))
}

func remotePollCanonical(request remotePollRequest) []byte {
	return []byte(strings.Join([]string{
		"VEXSTREAM_REMOTE_POLL_V1",
		request.HomeRef,
		request.DeviceRef,
		request.RequestRef,
		request.IssuedAt,
	}, "\n"))
}

func verifySignature(key *ecdsa.PublicKey, payload []byte, encoded string) bool {
	signature, err := decodeRawBase64(encoded)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(payload)
	return ecdsa.VerifyASN1(key, sum[:], signature)
}

func freshTimestamp(value string, now time.Time, skew time.Duration) bool {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return false
	}
	delta := now.Sub(parsed)
	if delta < 0 {
		delta = -delta
	}
	return delta <= skew
}

func bearerToken(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(value, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
}

func randomToken(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeRawBase64(value string) ([]byte, error) {
	return base64.RawStdEncoding.DecodeString(strings.TrimSpace(value))
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("trailing JSON")
	}
	return nil
}

func methodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": method + " required"})
}

func jsonOut(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if status == http.StatusNoContent {
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}

var _ = fmt.Sprintf
