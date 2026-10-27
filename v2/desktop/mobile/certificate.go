package mobile

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// ensureCertificate gives one Desktop installation a stable local pairing identity.
// The six-digit code is derived from the exact certificate digest that Android pins.
func ensureCertificate(
	certPath string,
	keyPath string,
) (string, string, error) {
	if certPEM, err := os.ReadFile(certPath); err == nil {
		block, _ := pem.Decode(certPEM)
		if block != nil {
			cert, parseErr := x509.ParseCertificate(block.Bytes)
			if parseErr == nil {
				sum := sha256.Sum256(cert.Raw)
				return hex.EncodeToString(sum[:]),
					pairingCodeFromDigest(sum),
					nil
			}
		}
	}

	privateKey, err := ecdsa.GenerateKey(
		elliptic.P256(),
		rand.Reader,
	)
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

	der, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return "", "", err
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", err
	}

	if err := os.WriteFile(
		certPath,
		pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: der,
		}),
		0o600,
	); err != nil {
		return "", "", err
	}

	if err := os.WriteFile(
		keyPath,
		pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: keyDER,
		}),
		0o600,
	); err != nil {
		return "", "", err
	}

	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:]),
		pairingCodeFromDigest(sum),
		nil
}

func pairingCodeFromDigest(sum [32]byte) string {
	value := binary.BigEndian.Uint32(sum[:4]) % 1_000_000
	return fmt.Sprintf("%06d", value)
}
