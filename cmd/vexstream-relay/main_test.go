package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRelayRoutesOnlySignedEnrolledDeviceEnvelopes(t *testing.T) {
	state := newRelayState()
	server := httptest.NewServer(state.handler())
	defer server.Close()

	certificateDER, homeKey, homeRef := relayTestHomeIdentity(t)
	deviceKey := relayTestDeviceKey(t)
	devicePublic := relayPublicKey(t, &deviceKey.PublicKey)

	devices := []relayDevice{
		{
			DeviceRef:       "vexlife.fixture",
			DevicePublicKey: devicePublic,
		},
	}
	_, digest, err := parseDevices(devices)
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Now().UTC().Format(time.RFC3339)
	nonce := "register.fixture"
	register := hostRegisterRequest{
		SchemaVersion:  relaySchema,
		HomeRef:        homeRef,
		CertificateDER: base64.RawStdEncoding.EncodeToString(certificateDER),
		Nonce:          nonce,
		IssuedAt:       issuedAt,
		Devices:        devices,
	}
	register.Signature = relaySign(
		t,
		homeKey,
		registrationCanonical(
			homeRef,
			nonce,
			issuedAt,
			digest,
		),
	)

	var registration struct {
		SessionToken string `json:"sessionToken"`
	}
	if status := relayTestJSON(
		t,
		http.MethodPost,
		server.URL+"/v1/host/register",
		"",
		register,
		&registration,
	); status != http.StatusOK {
		t.Fatalf("register HTTP=%d", status)
	}
	if registration.SessionToken == "" {
		t.Fatal("relay did not return host session token")
	}

	request := remoteRequestEnvelope{
		SchemaVersion:      relaySchema,
		HomeRef:            homeRef,
		DeviceRef:          "vexlife.fixture",
		RequestRef:         "request.fixture",
		IssuedAt:           time.Now().UTC().Format(time.RFC3339),
		EphemeralPublicKey: "ephemeral.fixture",
		Nonce:              "nonce.fixture",
		Ciphertext:         "ciphertext.fixture",
	}
	request.Signature = relaySign(
		t,
		deviceKey,
		remoteRequestCanonical(request),
	)

	if status := relayTestJSON(
		t,
		http.MethodPost,
		server.URL+"/v1/remote/request",
		"",
		request,
		nil,
	); status != http.StatusAccepted {
		t.Fatalf("remote request HTTP=%d", status)
	}

	var delivered remoteRequestEnvelope
	if status := relayTestJSON(
		t,
		http.MethodGet,
		server.URL+"/v1/host/requests?homeRef="+homeRef,
		registration.SessionToken,
		nil,
		&delivered,
	); status != http.StatusOK {
		t.Fatalf("host poll HTTP=%d", status)
	}
	if delivered.RequestRef != request.RequestRef ||
		delivered.Ciphertext != request.Ciphertext {
		t.Fatalf("delivered=%+v", delivered)
	}

	response := remoteResponseEnvelope{
		SchemaVersion: relaySchema,
		HomeRef:       homeRef,
		RequestRef:    request.RequestRef,
		Nonce:         "response-nonce.fixture",
		Ciphertext:    "response-ciphertext.fixture",
	}
	if status := relayTestJSON(
		t,
		http.MethodPost,
		server.URL+"/v1/host/respond",
		registration.SessionToken,
		response,
		nil,
	); status != http.StatusNoContent {
		t.Fatalf("host response HTTP=%d", status)
	}

	poll := remotePollRequest{
		SchemaVersion: relaySchema,
		HomeRef:       homeRef,
		DeviceRef:     request.DeviceRef,
		RequestRef:    request.RequestRef,
		IssuedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	poll.Signature = relaySign(
		t,
		deviceKey,
		remotePollCanonical(poll),
	)

	var returned remoteResponseEnvelope
	if status := relayTestJSON(
		t,
		http.MethodPost,
		server.URL+"/v1/remote/response",
		"",
		poll,
		&returned,
	); status != http.StatusOK {
		t.Fatalf("phone response HTTP=%d", status)
	}
	if returned.Ciphertext != response.Ciphertext {
		t.Fatalf("returned=%+v", returned)
	}
}

func TestRelayRejectsUnregisteredDeviceBeforeQueueing(t *testing.T) {
	state := newRelayState()
	server := httptest.NewServer(state.handler())
	defer server.Close()

	certificateDER, homeKey, homeRef := relayTestHomeIdentity(t)
	registeredKey := relayTestDeviceKey(t)
	registeredPublic := relayPublicKey(t, &registeredKey.PublicKey)
	devices := []relayDevice{{
		DeviceRef:       "vexlife.registered",
		DevicePublicKey: registeredPublic,
	}}
	_, digest, err := parseDevices(devices)
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Now().UTC().Format(time.RFC3339)
	register := hostRegisterRequest{
		SchemaVersion:  relaySchema,
		HomeRef:        homeRef,
		CertificateDER: base64.RawStdEncoding.EncodeToString(certificateDER),
		Nonce:          "nonce",
		IssuedAt:       issuedAt,
		Devices:        devices,
	}
	register.Signature = relaySign(
		t,
		homeKey,
		registrationCanonical(
			homeRef,
			register.Nonce,
			issuedAt,
			digest,
		),
	)
	if status := relayTestJSON(
		t,
		http.MethodPost,
		server.URL+"/v1/host/register",
		"",
		register,
		nil,
	); status != http.StatusOK {
		t.Fatalf("register HTTP=%d", status)
	}

	attacker := relayTestDeviceKey(t)
	request := remoteRequestEnvelope{
		SchemaVersion:      relaySchema,
		HomeRef:            homeRef,
		DeviceRef:          "vexlife.attacker",
		RequestRef:         "request.denied",
		IssuedAt:           time.Now().UTC().Format(time.RFC3339),
		EphemeralPublicKey: "opaque",
		Nonce:              "opaque",
		Ciphertext:         "opaque",
	}
	request.Signature = relaySign(
		t,
		attacker,
		remoteRequestCanonical(request),
	)
	if status := relayTestJSON(
		t,
		http.MethodPost,
		server.URL+"/v1/remote/request",
		"",
		request,
		nil,
	); status != http.StatusForbidden {
		t.Fatalf("unregistered device HTTP=%d", status)
	}
}

func relayTestHomeIdentity(
	t *testing.T,
) ([]byte, *ecdsa.PrivateKey, string) {
	t.Helper()
	key := relayTestDeviceKey(t)
	serial, err := rand.Int(
		rand.Reader,
		new(big.Int).Lsh(big.NewInt(1), 128),
	)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "VexStream relay test",
		},
		NotBefore:   now.Add(-time.Minute),
		NotAfter:    now.Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&key.PublicKey,
		key,
	)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(der)
	return der, key,
		"vexstream.home." +
			hex.EncodeToString(sum[:])
}

func relayTestDeviceKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(
		elliptic.P256(),
		rand.Reader,
	)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func relayPublicKey(
	t *testing.T,
	key *ecdsa.PublicKey,
) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawStdEncoding.EncodeToString(der)
}

func relaySign(
	t *testing.T,
	key *ecdsa.PrivateKey,
	payload []byte,
) string {
	t.Helper()
	sum := sha256.Sum256(payload)
	signature, err := ecdsa.SignASN1(
		rand.Reader,
		key,
		sum[:],
	)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawStdEncoding.EncodeToString(signature)
}

func relayTestJSON(
	t *testing.T,
	method string,
	url string,
	bearer string,
	input any,
	output any,
) int {
	t.Helper()
	var body []byte
	if input != nil {
		var err error
		body, err = json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
	}
	request, err := http.NewRequest(
		method,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}
	if input != nil {
		request.Header.Set(
			"Content-Type",
			"application/json",
		)
	}
	if bearer != "" {
		request.Header.Set(
			"Authorization",
			"Bearer "+bearer,
		)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if output != nil && response.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(response.Body).Decode(output); err != nil {
			t.Fatal(err)
		}
	}
	return response.StatusCode
}

// [VXG RealForever]
