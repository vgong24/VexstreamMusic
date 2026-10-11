package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"testing"
)

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

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
