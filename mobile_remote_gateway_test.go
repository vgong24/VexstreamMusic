package main

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"path/filepath"
	"testing"
)

func TestMobileRemoteGatewayIdentityPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway-key")

	first, err := ensureMobileRemoteGatewayKey(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ensureMobileRemoteGatewayKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("remote gateway public identity changed across restart")
	}
	der, err := base64.RawStdEncoding.DecodeString(first)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		t.Fatal(err)
	}
	switch parsed.(type) {
	case *ecdh.PublicKey, *ecdsa.PublicKey:
		// id-ecPublicKey has no signing/agreement discriminator in SubjectPublicKeyInfo.
	default:
		t.Fatalf("gateway key type=%T", parsed)
	}
}

func TestMobileRemoteDevicePublicKeyRequiresP256ECDSA(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawStdEncoding.EncodeToString(der)
	if err := validateMobileDevicePublicKey(encoded); err != nil {
		t.Fatalf("valid device key rejected: %v", err)
	}
	if err := validateMobileDevicePublicKey("not-a-key"); err == nil {
		t.Fatal("invalid device key was accepted")
	}
}

func TestMobileRemoteCipherRoundTrip(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "gateway-key")
	if _, err := ensureMobileRemoteGatewayKey(path); err != nil {
		t.Fatal(err)
	}
	gatewayPrivate, err := loadMobileRemoteGatewayPrivateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	phonePrivate, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := phonePrivate.ECDH(gatewayPrivate.PublicKey())
	if err != nil {
		t.Fatal(err)
	}
	requestRef := "remote.fixture"
	phoneKey := remoteHKDF(shared, requestRef)

	phonePublicDER, err := x509.MarshalPKIXPublicKey(phonePrivate.PublicKey())
	if err != nil {
		t.Fatal(err)
	}
	desktopKey, err := remoteRequestKey(
		path,
		base64.RawStdEncoding.EncodeToString(phonePublicDER),
		requestRef,
	)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("private VexStream request")
	aad := remoteRequestAAD(
		"vexstream.home.fixture",
		"vexlife.fixture",
		requestRef,
	)
	nonce, ciphertext, err := sealRemoteCiphertext(phoneKey, plaintext, aad)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := openRemoteCiphertext(desktopKey, nonce, ciphertext, aad)
	if err != nil {
		t.Fatal(err)
	}
	if string(opened) != string(plaintext) {
		t.Fatalf("round trip=%q", opened)
	}
}

func TestMobileRemoteCapabilityIsReadOnlyMusicSurface(t *testing.T) {
	allowed := []mobileRemoteInnerRequest{
		{Method: "GET", Path: "/v1/catalog", Token: "token"},
		{Method: "GET", Path: "/v1/media/track", Token: "token"},
		{Method: "HEAD", Path: "/v1/media/track", Token: "token"},
	}
	for _, request := range allowed {
		if !remoteRequestAllowed(request) {
			t.Fatalf("expected allowed: %+v", request)
		}
	}

	denied := []mobileRemoteInnerRequest{
		{Method: "POST", Path: "/v1/catalog", Token: "token"},
		{Method: "GET", Path: "/api/library", Token: "token"},
		{Method: "GET", Path: "/v1/media/../config", Token: "token"},
		{Method: "GET", Path: "/v1/catalog"},
	}
	for _, request := range denied {
		if remoteRequestAllowed(request) {
			t.Fatalf("expected denied: %+v", request)
		}
	}
}

// [VXG RealForever]
