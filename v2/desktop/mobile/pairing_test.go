package mobile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

func TestAuthorizationAcceptsOnlyStoredTokenHash(t *testing.T) {
	service := testService(t, library.Projection{}, nil)

	token := "fixture-secret-token"
	sum := sha256.Sum256([]byte(token))
	service.devices["vexlife.fixture"] = pairedDevice{
		DeviceRef: "vexlife.fixture",
		TokenHash: hex.EncodeToString(sum[:]),
	}

	allowed := httptest.NewRequest(
		http.MethodGet,
		"https://vexstream.local/v1/catalog",
		nil,
	)
	allowed.Header.Set("Authorization", "Bearer "+token)
	if !service.authorizedRequest(allowed) {
		t.Fatal("stored token should authorize")
	}

	wrong := httptest.NewRequest(
		http.MethodGet,
		"https://vexstream.local/v1/catalog",
		nil,
	)
	wrong.Header.Set("Authorization", "Bearer wrong-token")
	if service.authorizedRequest(wrong) {
		t.Fatal("unknown token must not authorize")
	}

	missing := httptest.NewRequest(
		http.MethodGet,
		"https://vexstream.local/v1/catalog",
		nil,
	)
	if service.authorizedRequest(missing) {
		t.Fatal("missing token must not authorize")
	}
}

func TestArtworkRouteUsesStoredBearerAuthorization(t *testing.T) {
	service :=
		testService(
			t,
			library.Projection{},
			nil,
		)

	token :=
		"fixture-artwork-token"
	sum :=
		sha256.Sum256(
			[]byte(token),
		)
	service.devices["vexlife.fixture"] =
		pairedDevice{
			DeviceRef: "vexlife.fixture",
			TokenHash: hex.EncodeToString(
				sum[:],
			),
		}

	handler :=
		service.authorized(
			service.mobileArtwork,
		)

	missing :=
		httptest.NewRecorder()
	handler(
		missing,
		httptest.NewRequest(
			http.MethodGet,
			"/v1/artwork/track",
			nil,
		),
	)
	if missing.Code !=
		http.StatusUnauthorized {
		t.Fatalf(
			"missing token status=%d",
			missing.Code,
		)
	}

	wrongRequest :=
		httptest.NewRequest(
			http.MethodGet,
			"/v1/artwork/track",
			nil,
		)
	wrongRequest.Header.Set(
		"Authorization",
		"Bearer wrong-token",
	)
	wrong :=
		httptest.NewRecorder()
	handler(
		wrong,
		wrongRequest,
	)
	if wrong.Code !=
		http.StatusUnauthorized {
		t.Fatalf(
			"wrong token status=%d",
			wrong.Code,
		)
	}

	allowedRequest :=
		httptest.NewRequest(
			http.MethodGet,
			"/v1/artwork/track",
			nil,
		)
	allowedRequest.Header.Set(
		"Authorization",
		"Bearer "+token,
	)
	allowed :=
		httptest.NewRecorder()
	handler(
		allowed,
		allowedRequest,
	)
	if allowed.Code !=
		http.StatusNotFound {
		t.Fatalf(
			"stored token did not reach artwork handler: status=%d body=%s",
			allowed.Code,
			allowed.Body.String(),
		)
	}
}

func TestPairingRequestRequiresOpenWindowAndTokenOnlyAfterApproval(t *testing.T) {
	service := testService(t, library.Projection{}, nil)

	body := []byte(
		`{"deviceRef":"vexlife.fixture","displayName":"Victor's phone"}`,
	)

	closedRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/pairing/request",
		bytes.NewReader(body),
	)
	closedResponse := httptest.NewRecorder()
	service.pairingRequest(closedResponse, closedRequest)
	if closedResponse.Code != http.StatusForbidden {
		t.Fatalf(
			"closed pairing status=%d body=%s",
			closedResponse.Code,
			closedResponse.Body.String(),
		)
	}

	service.pairingWindowUntil = time.Now().Add(time.Minute)
	openRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/pairing/request",
		bytes.NewReader(body),
	)
	openResponse := httptest.NewRecorder()
	service.pairingRequest(openResponse, openRequest)
	if openResponse.Code != http.StatusOK {
		t.Fatalf(
			"open pairing status=%d body=%s",
			openResponse.Code,
			openResponse.Body.String(),
		)
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

	pending := service.pending[formed.RequestRef]
	if pending == nil {
		t.Fatal("pending pairing request was not retained")
	}
	pending.Token = "must-not-leak"

	pendingStatus := httptest.NewRecorder()
	service.pairingRequestStatus(
		pendingStatus,
		httptest.NewRequest(
			http.MethodGet,
			"/v1/pairing/status?requestRef="+formed.RequestRef,
			nil,
		),
	)
	if strings.Contains(
		pendingStatus.Body.String(),
		"must-not-leak",
	) {
		t.Fatalf(
			"pending pairing response leaked token: %s",
			pendingStatus.Body.String(),
		)
	}

	pending.State = "APPROVED"
	approvedStatus := httptest.NewRecorder()
	service.pairingRequestStatus(
		approvedStatus,
		httptest.NewRequest(
			http.MethodGet,
			"/v1/pairing/status?requestRef="+formed.RequestRef,
			nil,
		),
	)
	if !strings.Contains(
		approvedStatus.Body.String(),
		"must-not-leak",
	) {
		t.Fatalf(
			"approved pairing response omitted token: %s",
			approvedStatus.Body.String(),
		)
	}
}

func TestRememberedDevicesAcceptV1V2AndRejectUnknownSchema(t *testing.T) {
	devices, err := decodeRememberedDevices([]byte(`{
	  "schemaVersion":"vexstream.mobile.local-pairing/v1",
	  "devices":[{
	    "deviceRef":"vexlife.fixture",
	    "displayName":"Victor's phone",
	    "tokenHash":"fixture-token-hash",
	    "approvedAt":"2026-10-11T12:00:00Z"
	  }]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 ||
		devices[0].DeviceRef != "vexlife.fixture" {
		t.Fatalf("decoded v1 devices=%+v", devices)
	}

	devices[0].DevicePublicKey = "fixture-p256-public-key"
	devices[0].RemoteEnabled = true
	payload, err := encodeRememberedDevices(
		map[string]pairedDevice{
			devices[0].DeviceRef: devices[0],
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	var stored pairingFile
	if err := json.Unmarshal(payload, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.SchemaVersion != pairingSchemaCurrent {
		t.Fatalf(
			"saved schema=%q want=%q",
			stored.SchemaVersion,
			pairingSchemaCurrent,
		)
	}
	if stored.Devices[0].DevicePublicKey !=
		"fixture-p256-public-key" ||
		!stored.Devices[0].RemoteEnabled {
		t.Fatalf(
			"v2 extensions were not preserved: %+v",
			stored.Devices[0],
		)
	}

	_, err = decodeRememberedDevices([]byte(`{
	  "schemaVersion":"vexstream.mobile.local-pairing/v99",
	  "devices":[]
	}`))
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"vexstream.mobile.local-pairing/v99",
		) {
		t.Fatalf("unknown schema error=%v", err)
	}
}

func TestUnusedPairingSurfaceStopsAfterWindowCloses(t *testing.T) {
	service := testService(t, library.Projection{}, nil)
	service.running = true
	service.pairingWindowUntil = time.Now().Add(-time.Second)

	service.stopIfUnused()

	if service.running {
		t.Fatal("unused LAN bridge should stop after pairing window closes")
	}
}

func TestInFlightPairingKeepsSurfaceAliveAfterWindowCloses(t *testing.T) {
	service := testService(t, library.Projection{}, nil)
	service.running = true
	service.pairingWindowUntil = time.Now().Add(-time.Second)
	service.pending["pair.fixture"] = &pendingPairing{
		RequestRef:  "pair.fixture",
		DeviceRef:   "vexlife.fixture",
		DisplayName: "Fixture phone",
		RequestedAt: time.Now().Add(-time.Minute),
		ExpiresAt:   time.Now().Add(time.Minute),
		State:       "PENDING",
	}

	service.stopIfUnused()

	if !service.running {
		t.Fatal(
			"in-flight pairing should keep the bridge alive until request expiry",
		)
	}
}

func TestRememberedPhoneKeepsSecureSurfaceAvailable(t *testing.T) {
	service := testService(t, library.Projection{}, nil)
	service.running = true
	service.pairingWindowUntil = time.Now().Add(-time.Second)
	service.devices["vexlife.fixture"] = pairedDevice{
		DeviceRef: "vexlife.fixture",
		TokenHash: sha256Hex("fixture-token"),
	}

	service.stopIfUnused()

	if !service.running {
		t.Fatal("remembered phone should keep secure LAN bridge available")
	}
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
