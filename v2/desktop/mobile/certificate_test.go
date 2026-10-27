package mobile

import (
	"path/filepath"
	"regexp"
	"testing"
)

func TestCertificateIdentityAndPairingCodePersistAcrossRestart(t *testing.T) {
	root := t.TempDir()
	certPath := filepath.Join(root, "server-cert.pem")
	keyPath := filepath.Join(root, "server-key.pem")

	firstFingerprint, firstCode, err := ensureCertificate(
		certPath,
		keyPath,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondFingerprint, secondCode, err := ensureCertificate(
		certPath,
		keyPath,
	)
	if err != nil {
		t.Fatal(err)
	}

	if firstFingerprint != secondFingerprint {
		t.Fatalf(
			"certificate identity changed across restart: %s != %s",
			firstFingerprint,
			secondFingerprint,
		)
	}
	if firstCode != secondCode {
		t.Fatalf(
			"pairing code changed across restart: %s != %s",
			firstCode,
			secondCode,
		)
	}
	if len(firstFingerprint) != 64 {
		t.Fatalf(
			"certificate fingerprint length=%d",
			len(firstFingerprint),
		)
	}
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(firstCode) {
		t.Fatalf("pairing code=%q", firstCode)
	}
}
