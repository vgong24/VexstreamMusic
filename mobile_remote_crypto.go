package main

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"os"
	"strings"
)

const mobileRemoteCryptoSchema = "vexstream.remote-crypto/v1"

// ensureMobileRemoteGatewayKey keeps one stable ECDH identity beside the existing VexStream mobile
// certificate. The private key never leaves app-private desktop storage; Android receives only the
// public key during an explicitly approved local pairing.
func ensureMobileRemoteGatewayKey(path string) (string, error) {
	curve := ecdh.P256()

	if data, err := os.ReadFile(path); err == nil {
		raw, decodeErr := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(data)))
		if decodeErr == nil {
			privateKey, keyErr := curve.NewPrivateKey(raw)
			if keyErr == nil {
				return marshalRemoteGatewayPublicKey(privateKey.PublicKey())
			}
		}
	}

	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}

	encoded := base64.RawStdEncoding.EncodeToString(privateKey.Bytes()) + "\n"
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return "", err
	}

	return marshalRemoteGatewayPublicKey(privateKey.PublicKey())
}

func loadMobileRemoteGatewayPrivateKey(path string) (*ecdh.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, err
	}
	return ecdh.P256().NewPrivateKey(raw)
}

func marshalRemoteGatewayPublicKey(publicKey *ecdh.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(der), nil
}

func validateMobileDevicePublicKey(encoded string) error {
	if strings.TrimSpace(encoded) == "" {
		return errors.New("device public key missing")
	}
	der, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return err
	}
	publicKey, ok := parsed.(*ecdsa.PublicKey)
	if !ok || publicKey.Curve == nil || publicKey.Curve.Params().Name != "P-256" {
		return errors.New("device public key must be ECDSA P-256")
	}
	return nil
}

func (b *mobileLANBridgeState) remoteHomeRefLocked() string {
	if b.certificateSHA256 == "" {
		return ""
	}
	return "vexstream.home." + b.certificateSHA256
}

// [VXG RealForever]
