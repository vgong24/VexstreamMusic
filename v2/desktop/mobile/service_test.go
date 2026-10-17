package mobile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
)

func testService(t *testing.T, projection library.Projection, rows []playlists.Playlist) *Service {
	t.Helper()
	service, err := New(Options{
		Root: t.TempDir(),
		Version: "4.0.0",
		LibrarySnapshot: func() library.Projection { return projection },
		PlaylistSnapshot: func() []playlists.Playlist { return rows },
	})
	if err != nil { t.Fatal(err) }
	return service
}

func TestCatalogPreservesProvenMobileProjectionWithoutDesktopPaths(t *testing.T) {
	root:=t.TempDir(); mediaPath:=filepath.Join(root,"private-library","song.mp3")
	if err:=os.MkdirAll(filepath.Dir(mediaPath),0o755);err!=nil{t.Fatal(err)}
	mediaBytes:=[]byte("vexstream-test-media"); if err:=os.WriteFile(mediaPath,mediaBytes,0o644);err!=nil{t.Fatal(err)}
	end:=18.0
	service:=testService(t,library.Projection{Tracks:[]library.Track{{ID:"path-derived-track-ref",Path:mediaPath,Filename:"song.mp3",Title:"On The Go",Artist:"Vex",Album:"Mobile",Genre:"Test",Year:"2026",Duration:20,Source:filepath.Dir(mediaPath),PlaybackStart:2,PlaybackEnd:&end}},Paths:map[string]string{"path-derived-track-ref":mediaPath}},[]playlists.Playlist{{ID:"playlist.one",Name:"Road Trip",TrackRefs:[]string{"path-derived-track-ref","missing-track"}}})
	response:=httptest.NewRecorder();service.mobileCatalog(response,httptest.NewRequest(http.MethodGet,"/v1/catalog",nil))
	if response.Code!=http.StatusOK{t.Fatalf("catalog status=%d body=%s",response.Code,response.Body.String())}
	var catalog catalogProjection;if err:=json.Unmarshal(response.Body.Bytes(),&catalog);err!=nil{t.Fatal(err)}
	if catalog.SourceVersion!="4.0.0"||len(catalog.Tracks)!=1||len(catalog.Playlists)!=1{t.Fatalf("catalog drift: %+v",catalog)}
	digest:=sha256.Sum256(mediaBytes);if catalog.Tracks[0].MediaSHA256!=hex.EncodeToString(digest[:])||catalog.Tracks[0].MediaByteLength!=int64(len(mediaBytes))||catalog.Tracks[0].PlaybackStartSeconds!=2||catalog.Tracks[0].PlaybackEndSeconds==nil||*catalog.Tracks[0].PlaybackEndSeconds!=18{t.Fatalf("track projection drift: %+v",catalog.Tracks[0])}
	if strings.Join(catalog.Playlists[0].TrackRefs,",")!="path-derived-track-ref"{t.Fatalf("playlist projection drift: %+v",catalog.Playlists[0])}
	body:=response.Body.String();for _,forbidden:=range []string{mediaPath,filepath.Dir(mediaPath),`"path"`,`"filename"`,`"source"`}{if strings.Contains(body,forbidden){t.Fatalf("catalog leaked %q: %s",forbidden,body)}}
}

func TestMediaPreservesRangeReads(t *testing.T){root:=t.TempDir();mediaPath:=filepath.Join(root,"song.mp3");if err:=os.WriteFile(mediaPath,[]byte("0123456789"),0o644);err!=nil{t.Fatal(err)};service:=testService(t,library.Projection{Paths:map[string]string{"track-ref":mediaPath}},nil);request:=httptest.NewRequest(http.MethodGet,"/v1/media/track-ref",nil);request.Header.Set("Range","bytes=2-5");response:=httptest.NewRecorder();service.mobileMedia(response,request);if response.Code!=http.StatusPartialContent||!bytes.Equal(response.Body.Bytes(),[]byte("2345")){t.Fatalf("range drift: code=%d body=%q",response.Code,response.Body.Bytes())};if response.Header().Get("Accept-Ranges")!="bytes"{t.Fatalf("Accept-Ranges=%q",response.Header().Get("Accept-Ranges"))}}

func TestCertificateIdentityAndCodePersistAcrossRestart(t *testing.T){root:=t.TempDir();certPath:=filepath.Join(root,"server-cert.pem");keyPath:=filepath.Join(root,"server-key.pem");firstFingerprint,firstCode,err:=ensureCertificate(certPath,keyPath);if err!=nil{t.Fatal(err)};secondFingerprint,secondCode,err:=ensureCertificate(certPath,keyPath);if err!=nil{t.Fatal(err)};if firstFingerprint!=secondFingerprint||firstCode!=secondCode{t.Fatalf("identity changed across restart: %s/%s -> %s/%s",firstFingerprint,firstCode,secondFingerprint,secondCode)};if len(firstFingerprint)!=64||!regexp.MustCompile(`^[0-9]{6}$`).MatchString(firstCode){t.Fatalf("certificate identity invalid: %s %s",firstFingerprint,firstCode)}}

func TestPairingRequiresWindowAndTokenAppearsOnlyAfterApproval(t *testing.T){service:=testService(t,library.Projection{},nil);body:=[]byte(`{"deviceRef":"vexlife.fixture","displayName":"Victor's phone","devicePublicKey":"compatible-extension"}`);closed:=httptest.NewRecorder();service.pairingRequest(closed,httptest.NewRequest(http.MethodPost,"/v1/pairing/request",bytes.NewReader(body)));if closed.Code!=http.StatusForbidden{t.Fatalf("closed status=%d body=%s",closed.Code,closed.Body.String())};service.pairingWindowUntil=time.Now().Add(time.Minute);open:=httptest.NewRecorder();service.pairingRequest(open,httptest.NewRequest(http.MethodPost,"/v1/pairing/request",bytes.NewReader(body)));if open.Code!=http.StatusOK{t.Fatalf("open status=%d body=%s",open.Code,open.Body.String())};var formed struct{RequestRef string `json:"requestRef"`;State string `json:"state"`};if err:=json.Unmarshal(open.Body.Bytes(),&formed);err!=nil{t.Fatal(err)};pending:=service.pending[formed.RequestRef];if pending==nil{t.Fatal("pending request missing")};pending.Token="must-not-leak";status:=httptest.NewRecorder();service.pairingRequestStatus(status,httptest.NewRequest(http.MethodGet,"/v1/pairing/status?requestRef="+formed.RequestRef,nil));if strings.Contains(status.Body.String(),"must-not-leak"){t.Fatalf("pending status leaked token: %s",status.Body.String())};pending.State="APPROVED";status=httptest.NewRecorder();service.pairingRequestStatus(status,httptest.NewRequest(http.MethodGet,"/v1/pairing/status?requestRef="+formed.RequestRef,nil));if !strings.Contains(status.Body.String(),"must-not-leak"){t.Fatalf("approved status did not return token: %s",status.Body.String())}}

func TestAuthorizationAcceptsOnlyRememberedTokenHash(t *testing.T){service:=testService(t,library.Projection{},nil);token:="fixture-secret-token";sum:=sha256.Sum256([]byte(token));service.devices["vexlife.fixture"]=pairedDevice{DeviceRef:"vexlife.fixture",TokenHash:hex.EncodeToString(sum[:])};allowed:=httptest.NewRequest(http.MethodGet,"/v1/catalog",nil);allowed.Header.Set("Authorization","Bearer "+token);if !service.authorizedRequest(allowed){t.Fatal("remembered token should authorize")};wrong:=httptest.NewRequest(http.MethodGet,"/v1/catalog",nil);wrong.Header.Set("Authorization","Bearer wrong-token");if service.authorizedRequest(wrong){t.Fatal("unknown token must not authorize")}}

func TestRememberedDevicesAcceptV1AndPreserveCurrentExtensions(t *testing.T){devices,err:=decodeRememberedDevices([]byte(`{"schemaVersion":"vexstream.mobile.local-pairing/v1","devices":[{"deviceRef":"vexlife.fixture","displayName":"Victor's phone","tokenHash":"fixture","approvedAt":"2026-10-11T12:00:00Z"}]}`));if err!=nil||len(devices)!=1{t.Fatalf("v1 decode: devices=%+v err=%v",devices,err)};devices[0].DevicePublicKey="fixture-p256-public-key";devices[0].RemoteEnabled=true;payload,err:=encodeRememberedDevices(map[string]pairedDevice{devices[0].DeviceRef:devices[0]});if err!=nil{t.Fatal(err)};var stored pairingFile;if err:=json.Unmarshal(payload,&stored);err!=nil{t.Fatal(err)};if stored.SchemaVersion!=pairingSchemaCurrent||stored.Devices[0].DevicePublicKey!="fixture-p256-public-key"||!stored.Devices[0].RemoteEnabled{t.Fatalf("current pairing extensions drift: %+v",stored)};if _,err:=decodeRememberedDevices([]byte(`{"schemaVersion":"vexstream.mobile.local-pairing/v99","devices":[]}`));err==nil{t.Fatal("unknown pairing schema must fail closed")}}
