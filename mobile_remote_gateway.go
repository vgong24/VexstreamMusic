package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
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
)

const (
	mobileRemoteSchema           = "vexstream.remote-relay/v1"
	mobileRemoteBodyMax          = 1024 * 1024
	mobileRemotePollTimeout      = 35 * time.Second
	mobileRemoteRegistrationRefresh = 55 * time.Second
	defaultMobileRemoteRelayURL      = ""
)

type mobileRemoteRequestEnvelope struct {
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

type mobileRemoteResponseEnvelope struct {
	SchemaVersion string `json:"schemaVersion"`
	HomeRef       string `json:"homeRef"`
	RequestRef    string `json:"requestRef"`
	Nonce         string `json:"nonce"`
	Ciphertext    string `json:"ciphertext"`
}

type mobileRemoteInnerRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Token   string            `json:"token"`
	Headers map[string]string `json:"headers,omitempty"`
}

type mobileRemoteInnerResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type mobileRemoteGatewayRuntime struct {
	mu      sync.Mutex
	running bool
}

var mobileRemoteGateway = &mobileRemoteGatewayRuntime{}

func mobileRemoteRelayURL() string {
	value := strings.TrimSpace(os.Getenv("VEXSTREAM_REMOTE_RELAY_URL"))
	if value == "" {
		value = defaultMobileRemoteRelayURL
	}
	value = strings.TrimRight(value, "/")
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return ""
	}
	return value
}

func startMobileRemoteGateway(a *App) {
	relayURL := mobileRemoteRelayURL()
	if relayURL == "" {
		return
	}

	mobileLANBridge.mu.Lock()
	remoteDevices := 0
	for _, device := range mobileLANBridge.devices {
		if device.RemoteEnabled && device.DevicePublicKey != "" {
			remoteDevices++
		}
	}
	mobileLANBridge.mu.Unlock()
	if remoteDevices == 0 {
		return
	}

	mobileRemoteGateway.mu.Lock()
	if mobileRemoteGateway.running {
		mobileRemoteGateway.mu.Unlock()
		return
	}
	mobileRemoteGateway.running = true
	mobileRemoteGateway.mu.Unlock()

	go func() {
		defer func() {
			mobileRemoteGateway.mu.Lock()
			mobileRemoteGateway.running = false
			mobileRemoteGateway.mu.Unlock()
		}()
		mobileRemoteGatewayLoop(a, relayURL)
	}()
}

func mobileRemoteGatewayLoop(a *App, relayURL string) {
	backoff := time.Second
	for {
		sessionToken, err := registerMobileRemoteHost(relayURL)
		if err != nil {
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		registeredAt := time.Now()

		for {
			// Re-register periodically instead of keeping one immortal relay session. This refreshes
			// the relay's public device-key allowlist after pair/revoke without granting the relay
			// authority over the underlying VexStream bearer or media.
			if time.Since(registeredAt) >= mobileRemoteRegistrationRefresh {
				break
			}

			request, status, err := pollMobileRemoteRequest(relayURL, sessionToken)
			if err != nil || status == http.StatusUnauthorized {
				break
			}
			if request == nil {
				continue
			}

			go func(envelope mobileRemoteRequestEnvelope, token string) {
				response, err := handleMobileRemoteRequest(a, envelope)
				if err != nil {
					response = mobileRemoteErrorResponse(envelope, err)
				}
				if response != nil {
					_ = postMobileRemoteResponse(relayURL, token, *response)
				}
			}(*request, sessionToken)
		}
	}
}

func registerMobileRemoteHost(relayURL string) (string, error) {
	mobileLANBridge.mu.Lock()
	homeRef := mobileLANBridge.remoteHomeRefLocked()
	certPath := mobileLANBridge.certPath
	keyPath := mobileLANBridge.keyPath
	devices := make([]relayDevice, 0)
	for _, device := range mobileLANBridge.devices {
		if device.RemoteEnabled && device.DevicePublicKey != "" {
			devices = append(devices, relayDevice{
				DeviceRef:       device.DeviceRef,
				DevicePublicKey: device.DevicePublicKey,
			})
		}
	}
	mobileLANBridge.mu.Unlock()

	if homeRef == "" || len(devices) == 0 {
		return "", errors.New("no remote-capable VexStream devices")
	}

	certificateDER, privateKey, err := loadMobileRemoteCertificateIdentity(certPath, keyPath)
	if err != nil {
		return "", err
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].DeviceRef < devices[j].DeviceRef
	})
	deviceDigest := relayDeviceDigest(devices)
	nonce, err := randomRef("remote-register")
	if err != nil {
		return "", err
	}
	issuedAt := time.Now().UTC().Format(time.RFC3339)
	signature, err := signECDSAPayload(
		privateKey,
		registrationCanonical(
			homeRef,
			nonce,
			issuedAt,
			deviceDigest,
		),
	)
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"schemaVersion":  mobileRemoteSchema,
		"homeRef":        homeRef,
		"certificateDer": base64.RawStdEncoding.EncodeToString(certificateDER),
		"nonce":          nonce,
		"issuedAt":       issuedAt,
		"devices":        devices,
		"signature":      signature,
	}

	var response struct {
		SessionToken string `json:"sessionToken"`
	}
	status, err := mobileRemoteJSONRequest(
		http.MethodPost,
		relayURL+"/v1/host/register",
		"",
		payload,
		&response,
	)
	if err != nil || status != http.StatusOK || response.SessionToken == "" {
		return "", fmt.Errorf("remote relay registration failed: status=%d err=%w", status, err)
	}
	return response.SessionToken, nil
}

func heartbeatMobileRemoteHost(relayURL, sessionToken string) error {
	mobileLANBridge.mu.Lock()
	homeRef := mobileLANBridge.remoteHomeRefLocked()
	mobileLANBridge.mu.Unlock()
	status, err := mobileRemoteJSONRequest(
		http.MethodPost,
		relayURL+"/v1/host/heartbeat?homeRef="+url.QueryEscape(homeRef),
		sessionToken,
		nil,
		nil,
	)
	if err != nil || status != http.StatusNoContent {
		return fmt.Errorf("remote relay heartbeat failed: status=%d err=%w", status, err)
	}
	return nil
}

func pollMobileRemoteRequest(
	relayURL string,
	sessionToken string,
) (*mobileRemoteRequestEnvelope, int, error) {
	mobileLANBridge.mu.Lock()
	homeRef := mobileLANBridge.remoteHomeRefLocked()
	mobileLANBridge.mu.Unlock()

	var response mobileRemoteRequestEnvelope
	status, err := mobileRemoteJSONRequest(
		http.MethodGet,
		relayURL+"/v1/host/requests?homeRef="+url.QueryEscape(homeRef),
		sessionToken,
		nil,
		&response,
	)
	if err != nil {
		return nil, status, err
	}
	if status == http.StatusNoContent {
		return nil, status, nil
	}
	if status != http.StatusOK {
		return nil, status, fmt.Errorf("relay request poll HTTP %d", status)
	}
	return &response, status, nil
}

func handleMobileRemoteRequest(
	a *App,
	envelope mobileRemoteRequestEnvelope,
) (*mobileRemoteResponseEnvelope, error) {
	if envelope.SchemaVersion != mobileRemoteSchema {
		return nil, errors.New("unsupported remote request schema")
	}

	mobileLANBridge.mu.Lock()
	device := mobileLANBridge.devices[envelope.DeviceRef]
	homeRef := mobileLANBridge.remoteHomeRefLocked()
	gatewayKeyPath := mobileLANBridge.remoteGatewayKeyPath
	mobileLANBridge.mu.Unlock()

	if homeRef == "" || envelope.HomeRef != homeRef ||
		!device.RemoteEnabled || device.DevicePublicKey == "" {
		return nil, errors.New("remote device is not authorized")
	}
	if !verifyMobileDeviceSignature(
		device.DevicePublicKey,
		remoteRequestCanonical(envelope),
		envelope.Signature,
	) {
		return nil, errors.New("remote device signature rejected")
	}

	key, err := remoteRequestKey(
		gatewayKeyPath,
		envelope.EphemeralPublicKey,
		envelope.RequestRef,
	)
	if err != nil {
		return nil, err
	}
	plaintext, err := openRemoteCiphertext(
		key,
		envelope.Nonce,
		envelope.Ciphertext,
		remoteRequestAAD(
			envelope.HomeRef,
			envelope.DeviceRef,
			envelope.RequestRef,
		),
	)
	if err != nil {
		return nil, err
	}

	var inner mobileRemoteInnerRequest
	if err := json.Unmarshal(plaintext, &inner); err != nil {
		return nil, err
	}
	if !remoteRequestAllowed(inner) {
		return nil, errors.New("remote request is outside the VexStream mobile capability")
	}

	response, err := forwardRemoteRequestToLocalBridge(a, inner)
	if err != nil {
		return nil, err
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	nonce, ciphertext, err := sealRemoteCiphertext(
		key,
		responseBytes,
		remoteResponseAAD(
			envelope.HomeRef,
			envelope.RequestRef,
		),
	)
	if err != nil {
		return nil, err
	}

	return &mobileRemoteResponseEnvelope{
		SchemaVersion: mobileRemoteSchema,
		HomeRef:       envelope.HomeRef,
		RequestRef:    envelope.RequestRef,
		Nonce:         nonce,
		Ciphertext:    ciphertext,
	}, nil
}

func mobileRemoteErrorResponse(
	envelope mobileRemoteRequestEnvelope,
	cause error,
) *mobileRemoteResponseEnvelope {
	mobileLANBridge.mu.Lock()
	gatewayKeyPath := mobileLANBridge.remoteGatewayKeyPath
	mobileLANBridge.mu.Unlock()
	key, err := remoteRequestKey(
		gatewayKeyPath,
		envelope.EphemeralPublicKey,
		envelope.RequestRef,
	)
	if err != nil {
		return nil
	}
	payload, _ := json.Marshal(mobileRemoteInnerResponse{
		Status: http.StatusBadGateway,
		Body: base64.RawStdEncoding.EncodeToString(
			[]byte("remote VexStream request failed"),
		),
	})
	nonce, ciphertext, err := sealRemoteCiphertext(
		key,
		payload,
		remoteResponseAAD(
			envelope.HomeRef,
			envelope.RequestRef,
		),
	)
	if err != nil {
		return nil
	}
	_ = cause
	return &mobileRemoteResponseEnvelope{
		SchemaVersion: mobileRemoteSchema,
		HomeRef:       envelope.HomeRef,
		RequestRef:    envelope.RequestRef,
		Nonce:         nonce,
		Ciphertext:    ciphertext,
	}
}

func postMobileRemoteResponse(
	relayURL, sessionToken string,
	response mobileRemoteResponseEnvelope,
) error {
	status, err := mobileRemoteJSONRequest(
		http.MethodPost,
		relayURL+"/v1/host/respond",
		sessionToken,
		response,
		nil,
	)
	if err != nil || status != http.StatusNoContent {
		return fmt.Errorf("remote response publish failed: status=%d err=%w", status, err)
	}
	return nil
}

func remoteRequestAllowed(request mobileRemoteInnerRequest) bool {
	method := strings.ToUpper(strings.TrimSpace(request.Method))
	path := "/" + strings.TrimLeft(strings.TrimSpace(request.Path), "/")
	if request.Token == "" {
		return false
	}
	switch {
	case method == http.MethodGet && path == "/v1/catalog":
		return true
	case (method == http.MethodGet || method == http.MethodHead) &&
		strings.HasPrefix(path, "/v1/media/"):
		return !strings.Contains(path, "..")
	default:
		return false
	}
}

func forwardRemoteRequestToLocalBridge(
	a *App,
	request mobileRemoteInnerRequest,
) (mobileRemoteInnerResponse, error) {
	_ = a
	client, err := mobileLoopbackClient()
	if err != nil {
		return mobileRemoteInnerResponse{}, err
	}
	path := "/" + strings.TrimLeft(request.Path, "/")
	req, err := http.NewRequest(
		strings.ToUpper(request.Method),
		"https://vexstream.local:"+fmt.Sprint(mobileLANPort)+path,
		nil,
	)
	if err != nil {
		return mobileRemoteInnerResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+request.Token)
	for name, value := range request.Headers {
		switch strings.ToLower(name) {
		case "range", "accept":
			req.Header.Set(name, value)
		}
	}

	response, err := client.Do(req)
	if err != nil {
		return mobileRemoteInnerResponse{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, mobileRemoteBodyMax+1))
	if err != nil {
		return mobileRemoteInnerResponse{}, err
	}
	if len(body) > mobileRemoteBodyMax {
		return mobileRemoteInnerResponse{}, errors.New("remote response exceeds bounded payload")
	}

	headers := make(map[string]string)
	for _, name := range []string{
		"Content-Type",
		"Content-Length",
		"Content-Range",
		"Accept-Ranges",
	} {
		if value := response.Header.Get(name); value != "" {
			headers[name] = value
		}
	}

	return mobileRemoteInnerResponse{
		Status:  response.StatusCode,
		Headers: headers,
		Body:    base64.RawStdEncoding.EncodeToString(body),
	}, nil
}

func mobileLoopbackClient() (*http.Client, error) {
	mobileLANBridge.mu.Lock()
	certPath := mobileLANBridge.certPath
	mobileLANBridge.mu.Unlock()

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certPEM) {
		return nil, errors.New("unable to load VexStream mobile certificate")
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    pool,
			ServerName: "vexstream.local",
			MinVersion: tls.VersionTLS12,
		},
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(
				ctx,
				network,
				fmt.Sprintf("127.0.0.1:%d", mobileLANPort),
			)
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
	}, nil
}

func remoteRequestKey(
	gatewayKeyPath string,
	ephemeralPublicKey string,
	requestRef string,
) ([]byte, error) {
	privateKey, err := loadMobileRemoteGatewayPrivateKey(gatewayKeyPath)
	if err != nil {
		return nil, err
	}
	der, err := base64.RawStdEncoding.DecodeString(ephemeralPublicKey)
	if err != nil {
		return nil, err
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	var publicKey *ecdh.PublicKey
	switch key := parsed.(type) {
	case *ecdh.PublicKey:
		publicKey = key
	case *ecdsa.PublicKey:
		publicKey, err = key.ECDH()
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("remote ephemeral key must be P-256 EC")
	}
	shared, err := privateKey.ECDH(publicKey)
	if err != nil {
		return nil, err
	}
	return remoteHKDF(shared, requestRef), nil
}

func remoteHKDF(shared []byte, requestRef string) []byte {
	salt := sha256.Sum256([]byte("vexstream.remote/v1"))
	extract := hmac.New(sha256.New, salt[:])
	_, _ = extract.Write(shared)
	prk := extract.Sum(nil)

	expand := hmac.New(sha256.New, prk)
	_, _ = expand.Write([]byte(requestRef))
	_, _ = expand.Write([]byte{1})
	return expand.Sum(nil)
}

func openRemoteCiphertext(
	key []byte,
	nonceEncoded string,
	ciphertextEncoded string,
	aad []byte,
) ([]byte, error) {
	nonce, err := base64.RawStdEncoding.DecodeString(nonceEncoded)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(ciphertextEncoded)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, errors.New("invalid remote nonce")
	}
	return aead.Open(nil, nonce, ciphertext, aad)
}

func sealRemoteCiphertext(
	key []byte,
	plaintext []byte,
	aad []byte,
) (string, string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", "", err
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, aad)
	return base64.RawStdEncoding.EncodeToString(nonce),
		base64.RawStdEncoding.EncodeToString(ciphertext),
		nil
}

func verifyMobileDeviceSignature(
	encodedKey string,
	payload []byte,
	encodedSignature string,
) bool {
	der, err := base64.RawStdEncoding.DecodeString(encodedKey)
	if err != nil {
		return false
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return false
	}
	key, ok := parsed.(*ecdsa.PublicKey)
	if !ok || key.Curve.Params().Name != "P-256" {
		return false
	}
	signature, err := base64.RawStdEncoding.DecodeString(encodedSignature)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(payload)
	return ecdsa.VerifyASN1(key, sum[:], signature)
}

func loadMobileRemoteCertificateIdentity(
	certPath, keyPath string,
) ([]byte, *ecdsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, errors.New("VexStream mobile certificate missing")
	}
	certificate, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, errors.New("VexStream mobile private key missing")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	privateKey, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, nil, errors.New("VexStream mobile private key is not ECDSA")
	}
	return certificate.Raw, privateKey, nil
}

func signECDSAPayload(
	key *ecdsa.PrivateKey,
	payload []byte,
) (string, error) {
	sum := sha256.Sum256(payload)
	signature, err := ecdsa.SignASN1(rand.Reader, key, sum[:])
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(signature), nil
}

type relayDevice struct {
	DeviceRef       string `json:"deviceRef"`
	DevicePublicKey string `json:"devicePublicKey"`
}

func relayDeviceDigest(devices []relayDevice) string {
	canonical := append([]relayDevice(nil), devices...)
	sort.Slice(canonical, func(i, j int) bool {
		return canonical[i].DeviceRef < canonical[j].DeviceRef
	})
	var input strings.Builder
	for _, device := range canonical {
		input.WriteString(device.DeviceRef)
		input.WriteByte('\n')
		input.WriteString(device.DevicePublicKey)
		input.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(input.String()))
	return hex.EncodeToString(sum[:])
}

func registrationCanonical(
	homeRef, nonce, issuedAt, deviceDigest string,
) []byte {
	return []byte(strings.Join([]string{
		"VEXSTREAM_REMOTE_REGISTER_V1",
		homeRef,
		nonce,
		issuedAt,
		deviceDigest,
	}, "\n"))
}

func remoteRequestCanonical(
	request mobileRemoteRequestEnvelope,
) []byte {
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

func remoteRequestAAD(
	homeRef, deviceRef, requestRef string,
) []byte {
	return []byte(strings.Join([]string{
		"VEXSTREAM_REMOTE_REQUEST_AAD_V1",
		homeRef,
		deviceRef,
		requestRef,
	}, "\n"))
}

func remoteResponseAAD(
	homeRef, requestRef string,
) []byte {
	return []byte(strings.Join([]string{
		"VEXSTREAM_REMOTE_RESPONSE_AAD_V1",
		homeRef,
		requestRef,
	}, "\n"))
}

func mobileRemoteJSONRequest(
	method, requestURL, bearer string,
	input any,
	output any,
) (int, error) {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(encoded)
	}

	ctx, cancel := context.WithTimeout(context.Background(), mobileRemotePollTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/json")
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if output != nil && response.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(io.LimitReader(response.Body, mobileRemoteBodyMax)).Decode(output); err != nil {
			return response.StatusCode, err
		}
	}
	return response.StatusCode, nil
}

// [VXG RealForever]
