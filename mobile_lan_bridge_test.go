package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestSameVersionOlderBuildIsNotReused(t *testing.T) {
	if shouldReuseRunningInstance(version, "") {
		t.Fatal("same marketing version without current build identity must not be reused")
	}
	if shouldReuseRunningInstance(version, "older-build") {
		t.Fatal("different same-version build must not be reused")
	}
	if !shouldReuseRunningInstance(version, buildRef) {
		t.Fatal("exact current version/build identity should be reused")
	}
}

func TestMobileLANCertificateIdentityPersistsAcrossRestart(t *testing.T) {
	root := t.TempDir()
	certPath := filepath.Join(root, "server-cert.pem")
	keyPath := filepath.Join(root, "server-key.pem")

	firstFingerprint, firstCode, err := ensureMobileLANCertificate(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	secondFingerprint, secondCode, err := ensureMobileLANCertificate(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}

	if firstFingerprint != secondFingerprint {
		t.Fatalf("certificate identity changed across restart: %s != %s", firstFingerprint, secondFingerprint)
	}
	if firstCode != secondCode {
		t.Fatalf("pairing confirmation code changed across restart: %s != %s", firstCode, secondCode)
	}
	if len(firstFingerprint) != 64 {
		t.Fatalf("certificate fingerprint length=%d", len(firstFingerprint))
	}
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(firstCode) {
		t.Fatalf("pairing code=%q", firstCode)
	}
}

func TestMobileLANAuthorizationAcceptsOnlyStoredTokenHash(t *testing.T) {
	token := "fixture-secret-token"
	sum := sha256Hex(token)

	bridge := &mobileLANBridgeState{
		devices: map[string]mobilePairedDevice{
			"vexlife.fixture": {
				DeviceRef: "vexlife.fixture",
				TokenHash: sum,
			},
		},
		pending: map[string]*mobilePendingPairing{},
	}

	allowed := httptest.NewRequest("GET", "https://vexstream.local/v1/catalog", nil)
	allowed.Header.Set("Authorization", "Bearer "+token)
	if !bridge.authorizedRequest(allowed) {
		t.Fatal("stored token should authorize")
	}

	wrong := httptest.NewRequest("GET", "https://vexstream.local/v1/catalog", nil)
	wrong.Header.Set("Authorization", "Bearer wrong-token")
	if bridge.authorizedRequest(wrong) {
		t.Fatal("unknown token must not authorize")
	}

	missing := httptest.NewRequest("GET", "https://vexstream.local/v1/catalog", nil)
	if bridge.authorizedRequest(missing) {
		t.Fatal("missing token must not authorize")
	}
}

func TestMobileLANPairingRequestRequiresOpenWindowAndExposesTokenOnlyAfterApproval(t *testing.T) {
	bridge := &mobileLANBridgeState{
		devices: map[string]mobilePairedDevice{},
		pending: map[string]*mobilePendingPairing{},
	}

	devicePublicKey := testMobileDevicePublicKey(t)
	body := []byte(
		`{"deviceRef":"vexlife.fixture","displayName":"Victor's phone","devicePublicKey":"` +
			devicePublicKey +
			`"}`,
	)
	closedRequest := httptest.NewRequest(http.MethodPost, "/v1/pairing/request", bytes.NewReader(body))
	closedResponse := httptest.NewRecorder()
	bridge.pairingRequest(closedResponse, closedRequest)
	if closedResponse.Code != http.StatusForbidden {
		t.Fatalf("closed pairing status=%d body=%s", closedResponse.Code, closedResponse.Body.String())
	}

	bridge.pairingWindowUntil = time.Now().Add(time.Minute)
	openRequest := httptest.NewRequest(http.MethodPost, "/v1/pairing/request", bytes.NewReader(body))
	openResponse := httptest.NewRecorder()
	bridge.pairingRequest(openResponse, openRequest)
	if openResponse.Code != http.StatusOK {
		t.Fatalf("open pairing status=%d body=%s", openResponse.Code, openResponse.Body.String())
	}

	var formed struct {
		RequestRef string `json:"requestRef"`
		State      string `json:"state"`
	}
	if err := json.Unmarshal(openResponse.Body.Bytes(), &formed); err != nil {
		t.Fatal(err)
	}
	if formed.RequestRef == "" || formed.State != "PENDING" {
		t.Fatalf("pairing response=%+v", formed)
	}

	pending := bridge.pending[formed.RequestRef]
	if pending == nil {
		t.Fatal("pending pairing request was not retained")
	}
	pending.State = "APPROVED"
	pending.Token = "approved-fixture-token"

	statusRequest := httptest.NewRequest(
		http.MethodGet,
		"/v1/pairing/status?requestRef="+formed.RequestRef,
		nil,
	)
	statusResponse := httptest.NewRecorder()
	bridge.pairingRequestStatus(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("pairing status HTTP=%d body=%s", statusResponse.Code, statusResponse.Body.String())
	}

	var status map[string]any
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status["state"] != "APPROVED" || status["token"] != "approved-fixture-token" {
		t.Fatalf("approved status=%v", status)
	}
}

func TestMobileLANPairingStatusNeverReturnsTokenWhilePending(t *testing.T) {
	bridge := &mobileLANBridgeState{
		devices: map[string]mobilePairedDevice{},
		pending: map[string]*mobilePendingPairing{
			"pair.fixture": {
				RequestRef:  "pair.fixture",
				DeviceRef:   "vexlife.fixture",
				DisplayName: "Fixture phone",
				RequestedAt: time.Now(),
				ExpiresAt:   time.Now().Add(time.Minute),
				State:       "PENDING",
				Token:       "must-not-leak",
			},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/pairing/status?requestRef=pair.fixture", nil)
	response := httptest.NewRecorder()
	bridge.pairingRequestStatus(response, request)

	var status map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if _, ok := status["token"]; ok {
		t.Fatalf("pending pairing response leaked token: %v", status)
	}
}

func TestMobileLANStopsUnusedSurfaceAfterPairingWindow(t *testing.T) {
	bridge := &mobileLANBridgeState{
		running:            true,
		pairingWindowUntil: time.Now().Add(-time.Second),
		devices:            map[string]mobilePairedDevice{},
		pending:            map[string]*mobilePendingPairing{},
	}

	bridge.stopIfUnused()

	if bridge.running {
		t.Fatal("unused LAN bridge should stop after the pairing window closes")
	}
}

func TestMobileLANStaysAvailableForInFlightPairingAfterWindowCloses(t *testing.T) {
	bridge := &mobileLANBridgeState{
		running:            true,
		pairingWindowUntil: time.Now().Add(-time.Second),
		devices:            map[string]mobilePairedDevice{},
		pending: map[string]*mobilePendingPairing{
			"pair.fixture": {
				RequestRef:  "pair.fixture",
				DeviceRef:   "vexlife.fixture",
				DisplayName: "Fixture phone",
				RequestedAt: time.Now().Add(-time.Minute),
				ExpiresAt:   time.Now().Add(time.Minute),
				State:       "PENDING",
			},
		},
	}

	bridge.stopIfUnused()

	if !bridge.running {
		t.Fatal("in-flight pairing should keep the bridge alive until the request expires")
	}
}

func TestMobileLANStaysAvailableForRememberedDevice(t *testing.T) {
	bridge := &mobileLANBridgeState{
		running:            true,
		pairingWindowUntil: time.Now().Add(-time.Second),
		devices: map[string]mobilePairedDevice{
			"vexlife.fixture": {
				DeviceRef: "vexlife.fixture",
				TokenHash: sha256Hex("fixture-token"),
			},
		},
		pending: map[string]*mobilePendingPairing{},
	}

	bridge.stopIfUnused()

	if !bridge.running {
		t.Fatal("remembered phone should keep the secure LAN bridge available")
	}
}


func testMobileDevicePublicKey(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawStdEncoding.EncodeToString(der)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
