package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/huin/goupnp/dcps/internetgateway2"
)

const (
	mobileDirectPort              = 18767
	mobileDirectPath              = "/v1/direct/request"
	mobileDirectBodyMax           = 768 * 1024
	mobileDirectClockSkew         = 2 * time.Minute
	mobileDirectReplayTTL         = 5 * time.Minute
	mobileDirectRateWindow        = time.Minute
	mobileDirectRateLimit         = 120
	mobileDirectReplayLimit       = 4096
	mobileDirectRateEntryLimit    = 1024
	mobileDirectMaxConcurrent     = 8
	mobileDirectUPnPLeaseSeconds  = 3600
	mobileDirectUPnPRenewInterval = 25 * time.Minute
	mobileDirectUPnPDescription   = "VexStream paired-device direct ingress"
)

type mobileDirectUPnPClient interface {
	AddPortMappingCtx(
		ctx context.Context,
		remoteHost string,
		externalPort uint16,
		protocol string,
		internalPort uint16,
		internalClient string,
		enabled bool,
		portMappingDescription string,
		leaseDurationSec uint32,
	) error
	DeletePortMappingCtx(ctx context.Context, remoteHost string, externalPort uint16, protocol string) error
	GetExternalIPAddressCtx(ctx context.Context) (string, error)
}

type mobileDirectUPnPMapping struct {
	client         mobileDirectUPnPClient
	externalPort   uint16
	internalClient string
	route          string
}

type mobileDirectRateState struct {
	windowStart time.Time
	count       int
}

type mobileDirectGatewayRuntime struct {
	mu       sync.Mutex
	running  bool
	server   *http.Server
	listener net.Listener
	routes   []string
	mapping  *mobileDirectUPnPMapping
	replay   map[string]time.Time
	rates    map[string]mobileDirectRateState
	slots    chan struct{}
}

var mobileDirectGateway = &mobileDirectGatewayRuntime{
	replay: map[string]time.Time{},
	rates:  map[string]mobileDirectRateState{},
	slots:  make(chan struct{}, mobileDirectMaxConcurrent),
}

func startMobileDirectGateway(a *App) {
	mobileLANBridge.mu.Lock()
	remoteDevices := 0
	for _, device := range mobileLANBridge.devices {
		if device.RemoteEnabled && device.DevicePublicKey != "" {
			remoteDevices++
		}
	}
	certPath := mobileLANBridge.certPath
	keyPath := mobileLANBridge.keyPath
	mobileLANBridge.mu.Unlock()

	if remoteDevices == 0 || certPath == "" || keyPath == "" {
		return
	}

	mobileDirectGateway.mu.Lock()
	if mobileDirectGateway.running {
		mobileDirectGateway.mu.Unlock()
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", mobileDirectPort))
	if err != nil {
		mobileDirectGateway.mu.Unlock()
		fmt.Fprintln(os.Stderr, "VexStream direct mobile ingress unavailable:", err)
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc(mobileDirectPath, mobileDirectRequestHandler(a))
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 4 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	mobileDirectGateway.listener = listener
	mobileDirectGateway.server = server
	mobileDirectGateway.running = true
	mobileDirectGateway.routes = mobileDirectBaseRoutes()
	mobileDirectGateway.mu.Unlock()

	go func() {
		if err := server.ServeTLS(listener, certPath, keyPath); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "VexStream direct mobile ingress stopped:", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	refreshMobileDirectUPnP(ctx)
	cancel()

	go mobileDirectUPnPRenewalLoop()
}

func stopMobileDirectGatewayIfUnused() {
	mobileLANBridge.mu.Lock()
	remoteDevices := 0
	for _, device := range mobileLANBridge.devices {
		if device.RemoteEnabled && device.DevicePublicKey != "" {
			remoteDevices++
		}
	}
	mobileLANBridge.mu.Unlock()
	if remoteDevices > 0 {
		return
	}

	mobileDirectGateway.mu.Lock()
	server := mobileDirectGateway.server
	mapping := mobileDirectGateway.mapping
	mobileDirectGateway.server = nil
	mobileDirectGateway.listener = nil
	mobileDirectGateway.mapping = nil
	mobileDirectGateway.routes = nil
	mobileDirectGateway.running = false
	mobileDirectGateway.replay = map[string]time.Time{}
	mobileDirectGateway.rates = map[string]mobileDirectRateState{}
	mobileDirectGateway.mu.Unlock()

	if mapping != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = mapping.client.DeletePortMappingCtx(ctx, "", mapping.externalPort, "TCP")
		cancel()
	}
	if server != nil {
		_ = server.Close()
	}
}

func mobileDirectGatewayRoutes() []string {
	mobileDirectGateway.mu.Lock()
	defer mobileDirectGateway.mu.Unlock()
	if !mobileDirectGateway.running {
		return nil
	}
	return append([]string(nil), mobileDirectGateway.routes...)
}

func mobileDirectRequestHandler(a *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			jsonOut(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		if !mobileDirectAllowSource(r.RemoteAddr, time.Now()) {
			jsonOut(w, http.StatusTooManyRequests, map[string]string{"error": "remote VexStream request rejected"})
			return
		}
		select {
		case mobileDirectGateway.slots <- struct{}{}:
			defer func() { <-mobileDirectGateway.slots }()
		default:
			jsonOut(w, http.StatusServiceUnavailable, map[string]string{"error": "remote VexStream ingress busy"})
			return
		}

		if r.ContentLength > mobileDirectBodyMax {
			jsonOut(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "remote VexStream request rejected"})
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, mobileDirectBodyMax+1))
		if err != nil || len(body) > mobileDirectBodyMax {
			jsonOut(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "remote VexStream request rejected"})
			return
		}

		var envelope mobileRemoteRequestEnvelope
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&envelope); err != nil {
			jsonOut(w, http.StatusBadRequest, map[string]string{"error": "remote VexStream request rejected"})
			return
		}
		if err := validateMobileDirectEnvelope(envelope, time.Now()); err != nil {
			jsonOut(w, http.StatusForbidden, map[string]string{"error": "remote VexStream request rejected"})
			return
		}
		if !verifyMobileDirectEnvelopeIdentity(envelope) {
			jsonOut(w, http.StatusForbidden, map[string]string{"error": "remote VexStream request rejected"})
			return
		}
		if !rememberMobileDirectRequest(envelope.DeviceRef, envelope.RequestRef, time.Now()) {
			jsonOut(w, http.StatusConflict, map[string]string{"error": "remote VexStream request rejected"})
			return
		}

		response, err := handleMobileRemoteRequest(a, envelope)
		if err != nil || response == nil {
			jsonOut(w, http.StatusBadRequest, map[string]string{"error": "remote VexStream request rejected"})
			return
		}
		jsonOut(w, http.StatusOK, response)
	}
}

func validateMobileDirectEnvelope(envelope mobileRemoteRequestEnvelope, now time.Time) error {
	if envelope.SchemaVersion != mobileRemoteSchema {
		return errors.New("unsupported direct VexStream schema")
	}
	if len(envelope.HomeRef) == 0 || len(envelope.HomeRef) > 256 ||
		len(envelope.DeviceRef) == 0 || len(envelope.DeviceRef) > 256 ||
		len(envelope.RequestRef) == 0 || len(envelope.RequestRef) > 256 ||
		len(envelope.IssuedAt) == 0 || len(envelope.IssuedAt) > 64 ||
		len(envelope.EphemeralPublicKey) == 0 || len(envelope.EphemeralPublicKey) > 1024 ||
		len(envelope.Nonce) == 0 || len(envelope.Nonce) > 128 ||
		len(envelope.Ciphertext) == 0 || len(envelope.Ciphertext) > mobileDirectBodyMax ||
		len(envelope.Signature) == 0 || len(envelope.Signature) > 1024 {
		return errors.New("direct VexStream envelope bounds rejected")
	}
	issuedAt, err := time.Parse(time.RFC3339, envelope.IssuedAt)
	if err != nil {
		return err
	}
	if issuedAt.Before(now.Add(-mobileDirectClockSkew)) || issuedAt.After(now.Add(mobileDirectClockSkew)) {
		return errors.New("direct VexStream envelope is stale")
	}
	return nil
}

func verifyMobileDirectEnvelopeIdentity(envelope mobileRemoteRequestEnvelope) bool {
	mobileLANBridge.mu.Lock()
	device := mobileLANBridge.devices[envelope.DeviceRef]
	homeRef := mobileLANBridge.remoteHomeRefLocked()
	mobileLANBridge.mu.Unlock()
	if homeRef == "" || envelope.HomeRef != homeRef || !device.RemoteEnabled || device.DevicePublicKey == "" {
		return false
	}
	return verifyMobileDeviceSignature(
		device.DevicePublicKey,
		remoteRequestCanonical(envelope),
		envelope.Signature,
	)
}

func rememberMobileDirectRequest(deviceRef, requestRef string, now time.Time) bool {
	key := deviceRef + "\n" + requestRef
	mobileDirectGateway.mu.Lock()
	defer mobileDirectGateway.mu.Unlock()

	for candidate, observedAt := range mobileDirectGateway.replay {
		if now.Sub(observedAt) > mobileDirectReplayTTL {
			delete(mobileDirectGateway.replay, candidate)
		}
	}
	if _, exists := mobileDirectGateway.replay[key]; exists {
		return false
	}
	if len(mobileDirectGateway.replay) >= mobileDirectReplayLimit {
		oldestKey := ""
		var oldest time.Time
		for candidate, observedAt := range mobileDirectGateway.replay {
			if oldestKey == "" || observedAt.Before(oldest) {
				oldestKey = candidate
				oldest = observedAt
			}
		}
		delete(mobileDirectGateway.replay, oldestKey)
	}
	mobileDirectGateway.replay[key] = now
	return true
}

func mobileDirectAllowSource(remoteAddr string, now time.Time) bool {
	host := remoteAddr
	if parsed, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = parsed
	}
	mobileDirectGateway.mu.Lock()
	defer mobileDirectGateway.mu.Unlock()

	state, exists := mobileDirectGateway.rates[host]
	if !exists && len(mobileDirectGateway.rates) >= mobileDirectRateEntryLimit {
		for candidate, candidateState := range mobileDirectGateway.rates {
			if now.Sub(candidateState.windowStart) >= 2*mobileDirectRateWindow {
				delete(mobileDirectGateway.rates, candidate)
			}
		}
		if len(mobileDirectGateway.rates) >= mobileDirectRateEntryLimit {
			return false
		}
	}
	if state.windowStart.IsZero() || now.Sub(state.windowStart) >= mobileDirectRateWindow {
		state = mobileDirectRateState{windowStart: now}
	}
	state.count++
	mobileDirectGateway.rates[host] = state
	return state.count <= mobileDirectRateLimit
}

func mobileDirectBaseRoutes() []string {
	routes := mobileDirectGlobalIPv6Routes()
	if configured := strings.TrimSpace(os.Getenv("VEXSTREAM_DIRECT_PUBLIC_URL")); configured != "" {
		if parsed, err := url.Parse(configured); err == nil && parsed.Scheme == "https" && parsed.Host != "" {
			routes = append(routes, strings.TrimRight(configured, "/"))
		}
	}
	return normalizeMobileDirectRoutes(routes)
}

func mobileDirectGlobalIPv6Routes() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var routes []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			var ip net.IP
			switch value := address.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if !mobileDirectPublicIPv6(ip) {
				continue
			}
			routes = append(routes, fmt.Sprintf("https://[%s]:%d", ip.String(), mobileDirectPort))
		}
	}
	return normalizeMobileDirectRoutes(routes)
}

func mobileDirectPublicIPv6(ip net.IP) bool {
	if ip == nil || ip.To4() != nil {
		return false
	}
	value := ip.To16()
	return value != nil && value[0]&0xe0 == 0x20
}

func mobileDirectPublicIPv4(ip net.IP) bool {
	value := ip.To4()
	if value == nil || value.IsUnspecified() || value.IsLoopback() || value.IsPrivate() {
		return false
	}
	if value[0] == 100 && value[1]&0xc0 == 64 {
		return false
	}
	if value[0] >= 224 {
		return false
	}
	return true
}

func normalizeMobileDirectRoutes(routes []string) []string {
	seen := map[string]bool{}
	clean := make([]string, 0, len(routes))
	for _, route := range routes {
		route = strings.TrimRight(strings.TrimSpace(route), "/")
		if route == "" || seen[route] {
			continue
		}
		seen[route] = true
		clean = append(clean, route)
	}
	sort.Strings(clean)
	return clean
}

func mobileDirectUPnPRenewalLoop() {
	ticker := time.NewTicker(mobileDirectUPnPRenewInterval)
	defer ticker.Stop()
	for range ticker.C {
		mobileDirectGateway.mu.Lock()
		running := mobileDirectGateway.running
		mobileDirectGateway.mu.Unlock()
		if !running {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		refreshMobileDirectUPnP(ctx)
		cancel()
	}
}

func refreshMobileDirectUPnP(ctx context.Context) {
	refreshMobileDirectBaseRoutes()

	internalIP, err := mobileDirectDefaultIPv4()
	if err != nil {
		return
	}

	mobileDirectGateway.mu.Lock()
	existing := mobileDirectGateway.mapping
	mobileDirectGateway.mu.Unlock()
	if existing != nil {
		externalText, externalErr := existing.client.GetExternalIPAddressCtx(ctx)
		externalIP := net.ParseIP(strings.TrimSpace(externalText))
		if externalErr == nil && mobileDirectPublicIPv4(externalIP) {
			if err := existing.client.AddPortMappingCtx(
				ctx,
				"",
				existing.externalPort,
				"TCP",
				mobileDirectPort,
				existing.internalClient,
				true,
				mobileDirectUPnPDescription,
				mobileDirectUPnPLeaseSeconds,
			); err == nil {
				refreshed := &mobileDirectUPnPMapping{
					client:         existing.client,
					externalPort:   existing.externalPort,
					internalClient: existing.internalClient,
					route:          fmt.Sprintf("https://%s:%d", externalIP.String(), existing.externalPort),
				}
				setMobileDirectUPnPMapping(refreshed)
				return
			}
		}
	}

	clients := mobileDirectUPnPClients(ctx)
	for _, client := range clients {
		externalText, err := client.GetExternalIPAddressCtx(ctx)
		if err != nil {
			continue
		}
		externalIP := net.ParseIP(strings.TrimSpace(externalText))
		if !mobileDirectPublicIPv4(externalIP) {
			continue
		}
		for candidate := mobileDirectPort; candidate < mobileDirectPort+10; candidate++ {
			externalPort := uint16(candidate)
			if err := client.AddPortMappingCtx(
				ctx,
				"",
				externalPort,
				"TCP",
				mobileDirectPort,
				internalIP.String(),
				true,
				mobileDirectUPnPDescription,
				mobileDirectUPnPLeaseSeconds,
			); err != nil {
				continue
			}
			mapping := &mobileDirectUPnPMapping{
				client:         client,
				externalPort:   externalPort,
				internalClient: internalIP.String(),
				route:          fmt.Sprintf("https://%s:%d", externalIP.String(), externalPort),
			}
			setMobileDirectUPnPMapping(mapping)
			return
		}
	}
}

func refreshMobileDirectBaseRoutes() {
	mobileDirectGateway.mu.Lock()
	mapping := mobileDirectGateway.mapping
	routes := mobileDirectBaseRoutes()
	if mapping != nil {
		routes = append(routes, mapping.route)
	}
	mobileDirectGateway.routes = normalizeMobileDirectRoutes(routes)
	mobileDirectGateway.mu.Unlock()
}

func setMobileDirectUPnPMapping(mapping *mobileDirectUPnPMapping) {
	mobileDirectGateway.mu.Lock()
	previous := mobileDirectGateway.mapping
	mobileDirectGateway.mapping = mapping
	routes := mobileDirectBaseRoutes()
	if mapping != nil {
		routes = append(routes, mapping.route)
	}
	mobileDirectGateway.routes = normalizeMobileDirectRoutes(routes)
	mobileDirectGateway.mu.Unlock()

	if previous != nil && previous != mapping &&
		(previous.externalPort != mapping.externalPort || previous.route != mapping.route) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = previous.client.DeletePortMappingCtx(ctx, "", previous.externalPort, "TCP")
		cancel()
	}
}

func mobileDirectUPnPClients(ctx context.Context) []mobileDirectUPnPClient {
	var clients []mobileDirectUPnPClient
	if discovered, _, err := internetgateway2.NewWANIPConnection2ClientsCtx(ctx); err == nil {
		for _, client := range discovered {
			clients = append(clients, client)
		}
	}
	if discovered, _, err := internetgateway2.NewWANIPConnection1ClientsCtx(ctx); err == nil {
		for _, client := range discovered {
			clients = append(clients, client)
		}
	}
	if discovered, _, err := internetgateway2.NewWANPPPConnection1ClientsCtx(ctx); err == nil {
		for _, client := range discovered {
			clients = append(clients, client)
		}
	}
	return clients
}

func mobileDirectDefaultIPv4() (net.IP, error) {
	connection, err := net.Dial("udp4", "1.1.1.1:53")
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	address, ok := connection.LocalAddr().(*net.UDPAddr)
	if !ok || address.IP == nil || address.IP.To4() == nil {
		return nil, errors.New("VexStream direct IPv4 route unavailable")
	}
	return address.IP.To4(), nil
}

// [VXG RealForever]
