package main

import (
	"net"
	"testing"
	"time"
)

func TestMobileDirectEnvelopeFreshness(t *testing.T) {
	now := time.Date(2026, 10, 11, 18, 30, 0, 0, time.UTC)
	envelope := mobileRemoteRequestEnvelope{
		SchemaVersion:      mobileRemoteSchema,
		HomeRef:            "vexstream.home.fixture",
		DeviceRef:          "vexlife.android.fixture",
		RequestRef:         "remote.fixture.001",
		IssuedAt:           now.Format(time.RFC3339),
		EphemeralPublicKey: "fixture-public-key",
		Nonce:              "fixture-nonce",
		Ciphertext:         "fixture-ciphertext",
		Signature:          "fixture-signature",
	}
	if err := validateMobileDirectEnvelope(envelope, now); err != nil {
		t.Fatalf("fresh direct envelope rejected: %v", err)
	}

	envelope.IssuedAt = now.Add(-mobileDirectClockSkew - time.Second).Format(time.RFC3339)
	if err := validateMobileDirectEnvelope(envelope, now); err == nil {
		t.Fatal("stale direct envelope was accepted")
	}

	envelope.IssuedAt = now.Add(mobileDirectClockSkew + time.Second).Format(time.RFC3339)
	if err := validateMobileDirectEnvelope(envelope, now); err == nil {
		t.Fatal("future direct envelope outside clock skew was accepted")
	}
}

func TestMobileDirectReplayGuardRejectsDuplicate(t *testing.T) {
	mobileDirectGateway.mu.Lock()
	mobileDirectGateway.replay = map[string]time.Time{}
	mobileDirectGateway.mu.Unlock()

	now := time.Date(2026, 10, 11, 18, 30, 0, 0, time.UTC)
	if !rememberMobileDirectRequest("device.fixture", "request.fixture", now) {
		t.Fatal("first direct request was rejected")
	}
	if rememberMobileDirectRequest("device.fixture", "request.fixture", now.Add(time.Second)) {
		t.Fatal("duplicate direct request was accepted")
	}
	if !rememberMobileDirectRequest("device.fixture", "request.fixture", now.Add(mobileDirectReplayTTL+time.Second)) {
		t.Fatal("expired replay entry did not clear")
	}
}

func TestMobileDirectPublicAddressFiltering(t *testing.T) {
	for _, value := range []string{"10.0.0.4", "192.168.1.20", "100.64.0.8", "127.0.0.1"} {
		if mobileDirectPublicIPv4(net.ParseIP(value)) {
			t.Fatalf("non-public IPv4 route admitted: %s", value)
		}
	}
	if !mobileDirectPublicIPv4(net.ParseIP("203.0.113.8")) {
		t.Fatal("public-shaped IPv4 route rejected")
	}
	if mobileDirectPublicIPv6(net.ParseIP("fd00::1")) {
		t.Fatal("ULA IPv6 route admitted")
	}
	if !mobileDirectPublicIPv6(net.ParseIP("2001:db8::1")) {
		t.Fatal("global-shaped IPv6 route rejected")
	}
}

func TestMobileDirectRateBudgetIsBounded(t *testing.T) {
	mobileDirectGateway.mu.Lock()
	mobileDirectGateway.rates = map[string]mobileDirectRateState{}
	mobileDirectGateway.mu.Unlock()

	now := time.Date(2026, 10, 11, 18, 30, 0, 0, time.UTC)
	for i := 0; i < mobileDirectRateLimit; i++ {
		if !mobileDirectAllowSource("203.0.113.9:443", now) {
			t.Fatalf("request %d was rejected before the direct rate limit", i+1)
		}
	}
	if mobileDirectAllowSource("203.0.113.9:443", now) {
		t.Fatal("request beyond the direct rate limit was accepted")
	}
	if !mobileDirectAllowSource("203.0.113.9:443", now.Add(mobileDirectRateWindow)) {
		t.Fatal("direct rate window did not reset")
	}
}

// [VXG RealForever]
