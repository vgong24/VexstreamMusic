// Package mobile preserves VexStream's proven same-Wi-Fi phone producer contract.
//
// This is the V4 composition of the already-lived V3 contract:
//   _vexstream._tcp -> pinned TLS -> desktop approval -> bearer
//   -> sanitized /v1/catalog + range-capable /v1/media/<trackRef>
//
// It does not expose desktop administration, filesystem paths, import/trash/repair,
// Home authority, or remote/off-home authority.
package mobile

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/playlists"
)

const (
	DefaultLANPort       = 18766
	serviceType          = "_vexstream._tcp"
	pairingWindow        = 5 * time.Minute
	pairingRequestLife   = 10 * time.Minute
	pairingSchemaV1      = "vexstream.mobile.local-pairing/v1"
	pairingSchemaCurrent = "vexstream.mobile.local-pairing/v2"
)

type Options struct {
	Root             string
	Version          string
	LibrarySnapshot  func() library.Projection
	PlaylistSnapshot func() []playlists.Playlist
}

type pairedDevice struct {
	DeviceRef       string `json:"deviceRef"`
	DisplayName     string `json:"displayName"`
	TokenHash       string `json:"tokenHash"`
	DevicePublicKey string `json:"devicePublicKey,omitempty"`
	RemoteEnabled   bool   `json:"remoteEnabled,omitempty"`
	ApprovedAt      string `json:"approvedAt"`
}
type pairingFile struct {
	SchemaVersion string         `json:"schemaVersion"`
	Devices       []pairedDevice `json:"devices"`
}
type pendingPairing struct {
	RequestRef string
	DeviceRef string
	DisplayName string
	RequestedAt time.Time
	ExpiresAt time.Time
	State string
	Token string
}
type fingerprintCacheEntry struct{ Size int64; ModTime int64; SHA256 string }

type Service struct {
	mu sync.Mutex
	root string
	version string
	librarySnapshot func() library.Projection
	playlistSnapshot func() []playlists.Playlist
	running bool
	pairingWindowUntil time.Time
	pending map[string]*pendingPairing
	devices map[string]pairedDevice
	server *http.Server
	listener net.Listener
	mdns *zeroconf.Server
	certPath string
	keyPath string
	certificateSHA256 string
	pairingCode string
	displayName string
	fingerprintMu sync.Mutex
	fingerprintCache map[string]fingerprintCacheEntry
}

func New(options Options) (*Service,error) {
	options.Root=strings.TrimSpace(options.Root)
	options.Version=strings.TrimSpace(options.Version)
	if options.Root=="" { return nil,errors.New("mobile bridge root is required") }
	if options.Version=="" { return nil,errors.New("mobile bridge source version is required") }
	if options.LibrarySnapshot==nil||options.PlaylistSnapshot==nil { return nil,errors.New("mobile bridge requires Library and Playlist snapshots") }
	if err:=os.MkdirAll(options.Root,0o700);err!=nil{return nil,err}
	s:=&Service{root:options.Root,version:options.Version,librarySnapshot:options.LibrarySnapshot,playlistSnapshot:options.PlaylistSnapshot,pending:map[string]*pendingPairing{},devices:map[string]pairedDevice{},fingerprintCache:map[string]fingerprintCacheEntry{}}
	s.certPath=filepath.Join(options.Root,"server-cert.pem"); s.keyPath=filepath.Join(options.Root,"server-key.pem")
	fingerprint,code,err:=ensureCertificate(s.certPath,s.keyPath); if err!=nil{return nil,err}
	s.certificateSHA256=fingerprint; s.pairingCode=code
	host,_:=os.Hostname(); host=strings.TrimSpace(host); if host==""{host="Windows PC"}; s.displayName="VexStream on "+host
	data,err:=os.ReadFile(filepath.Join(options.Root,"paired-devices.json")); if err!=nil&&!os.IsNotExist(err){return nil,err}
	if len(data)>0{ devices,err:=decodeRememberedDevices(data); if err!=nil{return nil,err}; for _,device:=range devices{if device.DeviceRef!=""&&device.TokenHash!=""{s.devices[device.DeviceRef]=device}}}
	return s,nil
}

func (s *Service) ResumeRemembered() error { s.mu.Lock(); remembered:=len(s.devices)>0; s.mu.Unlock(); if !remembered{return nil}; return s.EnsureStarted() }

func (s *Service) EnsureStarted() error {
	s.mu.Lock()
	if s.running{s.mu.Unlock();return nil}
	listener,err:=net.Listen("tcp",fmt.Sprintf("0.0.0.0:%d",DefaultLANPort)); if err!=nil{s.mu.Unlock();return err}
	mux:=http.NewServeMux()
	mux.HandleFunc("/v1/health",s.lanHealth)
	mux.HandleFunc("/v1/pairing/info",s.pairingInfo)
	mux.HandleFunc("/v1/pairing/request",s.pairingRequest)
	mux.HandleFunc("/v1/pairing/status",s.pairingRequestStatus)
	mux.HandleFunc("/v1/pairing/revoke-self",s.authorized(s.pairingRevokeSelf))
	mux.HandleFunc("/v1/catalog",s.authorized(s.mobileCatalog))
	mux.HandleFunc("/v1/media/",s.authorized(s.mobileMedia))
	server:=&http.Server{Handler:mux,ReadHeaderTimeout:5*time.Second,IdleTimeout:60*time.Second}
	s.listener=listener; s.server=server; s.running=true
	displayName:=s.displayName; certPath:=s.certPath; keyPath:=s.keyPath; fingerprint:=s.certificateSHA256
	s.mu.Unlock()
	mdns,mdnsErr:=zeroconf.Register(displayName,serviceType,"local.",DefaultLANPort,[]string{"bridge=vexstream-mobile/v1","transport=https","pairing=desktop-approval","certificateSha256="+fingerprint},nil)
	if mdnsErr!=nil{_ = listener.Close();s.mu.Lock();s.listener=nil;s.server=nil;s.running=false;s.mu.Unlock();return fmt.Errorf("VexStream nearby discovery unavailable: %w",mdnsErr)}
	s.mu.Lock();s.mdns=mdns;s.mu.Unlock()
	go func(){if err:=server.ServeTLS(listener,certPath,keyPath);err!=nil&&!errors.Is(err,http.ErrServerClosed){}}()
	return nil
}

func (s *Service) Close(){s.mu.Lock();server:=s.server;mdns:=s.mdns;s.server=nil;s.listener=nil;s.mdns=nil;s.running=false;s.mu.Unlock();if mdns!=nil{mdns.Shutdown()};if server!=nil{_ = server.Close()}}
func (s *Service) AdminHandler() http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){switch r.URL.Path{case "/api/mobile/pairing":s.mobilePairingStatus(w,r);case "/api/mobile/pairing/start":s.mobilePairingStart(w,r);case "/api/mobile/pairing/decision":s.mobilePairingDecision(w,r);case "/api/mobile/pairing/revoke":s.mobilePairingRevoke(w,r);default:http.NotFound(w,r)}})}

type trackProjection struct{TrackRef string `json:"trackRef"`;MediaSHA256 string `json:"mediaSha256"`;MediaByteLength int64 `json:"mediaByteLength"`;Title string `json:"title"`;Artist string `json:"artist"`;Album string `json:"album,omitempty"`;Genre string `json:"genre,omitempty"`;Year string `json:"year,omitempty"`;DurationSeconds float64 `json:"durationSeconds,omitempty"`;PlaybackStartSeconds float64 `json:"playbackStartSeconds,omitempty"`;PlaybackEndSeconds *float64 `json:"playbackEndSeconds,omitempty"`}
type playlistProjection struct{PlaylistRef string `json:"playlistRef"`;Name string `json:"name"`;TrackRefs []string `json:"trackRefs"`}
type catalogProjection struct{HubRef string `json:"hubRef"`;SourceVersion string `json:"sourceVersion"`;CatalogRevision string `json:"catalogRevision"`;Tracks []trackProjection `json:"tracks"`;Playlists []playlistProjection `json:"playlists"`}

func (s *Service) mobileCatalog(w http.ResponseWriter,r *http.Request){
	if r.Method!=http.MethodGet{w.Header().Set("Allow",http.MethodGet);writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"GET required"});return}
	snapshot:=s.librarySnapshot(); projected:=make([]trackProjection,0,len(snapshot.Tracks))
	for _,track:=range snapshot.Tracks{path:=snapshot.Paths[track.ID];if path==""{writeJSON(w,http.StatusServiceUnavailable,map[string]string{"error":"library media path unavailable for "+track.ID});return};sha,size,err:=s.mediaFingerprint(path);if err!=nil{writeJSON(w,http.StatusServiceUnavailable,map[string]string{"error":"library media unavailable for "+track.ID});return};end:=track.PlaybackEnd;if end!=nil{copied:=*end;end=&copied};projected=append(projected,trackProjection{TrackRef:track.ID,MediaSHA256:sha,MediaByteLength:size,Title:track.Title,Artist:track.Artist,Album:track.Album,Genre:track.Genre,Year:track.Year,DurationSeconds:track.Duration,PlaybackStartSeconds:track.PlaybackStart,PlaybackEndSeconds:end})}
	sort.Slice(projected,func(i,j int)bool{return projected[i].TrackRef<projected[j].TrackRef})
	available:=make(map[string]bool,len(projected));for _,track:=range projected{available[track.TrackRef]=true}
	projectedPlaylists:=projectPlaylists(s.playlistSnapshot(),available)
	revisionBytes,err:=json.Marshal(struct{Tracks []trackProjection `json:"tracks"`;Playlists []playlistProjection `json:"playlists"`}{Tracks:projected,Playlists:projectedPlaylists});if err!=nil{writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"catalog projection failed"});return};revision:=sha256.Sum256(revisionBytes)
	w.Header().Set("Cache-Control","no-store");writeJSON(w,http.StatusOK,catalogProjection{HubRef:"vexstream.desktop.local",SourceVersion:s.version,CatalogRevision:hex.EncodeToString(revision[:]),Tracks:projected,Playlists:projectedPlaylists})
}
func (s *Service) mobileMedia(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodGet&&r.Method!=http.MethodHead{w.Header().Set("Allow","GET, HEAD");writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"GET or HEAD required"});return};trackRef:=strings.TrimPrefix(r.URL.Path,"/v1/media/");if trackRef==""||strings.ContainsAny(trackRef,"/\\"){http.NotFound(w,r);return};path:=s.librarySnapshot().Paths[trackRef];if path==""{http.NotFound(w,r);return};file,err:=os.Open(path);if err!=nil{http.NotFound(w,r);return};defer file.Close();stat,err:=file.Stat();if err!=nil||!stat.Mode().IsRegular(){http.NotFound(w,r);return};w.Header().Set("Content-Type","audio/mpeg");w.Header().Set("Accept-Ranges","bytes");w.Header().Set("Cache-Control","private, no-store");http.ServeContent(w,r,filepath.Base(path),stat.ModTime(),file)}
func projectPlaylists(rows []playlists.Playlist,available map[string]bool)[]playlistProjection{projected:=make([]playlistProjection,0,len(rows));for _,row:=range rows{name:=strings.TrimSpace(row.Name);if row.ID==""||name==""{continue};refs:=make([]string,0,len(row.TrackRefs));for _,ref:=range row.TrackRefs{if available[ref]{refs=append(refs,ref)}};projected=append(projected,playlistProjection{PlaylistRef:row.ID,Name:name,TrackRefs:refs})};sort.Slice(projected,func(i,j int)bool{if projected[i].Name==projected[j].Name{return projected[i].PlaylistRef<projected[j].PlaylistRef};return strings.ToLower(projected[i].Name)<strings.ToLower(projected[j].Name)});return projected}
func (s *Service) mediaFingerprint(path string)(string,int64,error){stat,err:=os.Stat(path);if err!=nil{return "",0,err};if !stat.Mode().IsRegular(){return "",0,errors.New("media path is not a regular file")};key:=filepath.Clean(path);modTime:=stat.ModTime().UnixNano();s.fingerprintMu.Lock();cached,ok:=s.fingerprintCache[key];s.fingerprintMu.Unlock();if ok&&cached.Size==stat.Size()&&cached.ModTime==modTime{return cached.SHA256,cached.Size,nil};file,err:=os.Open(path);if err!=nil{return "",0,err};defer file.Close();digest:=sha256.New();if _,err:=io.Copy(digest,file);err!=nil{return "",0,err};value:=hex.EncodeToString(digest.Sum(nil));s.fingerprintMu.Lock();s.fingerprintCache[key]=fingerprintCacheEntry{Size:stat.Size(),ModTime:modTime,SHA256:value};s.fingerprintMu.Unlock();return value,stat.Size(),nil}

func (s *Service) lanHealth(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodGet{w.Header().Set("Allow",http.MethodGet);writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"GET required"});return};s.mu.Lock();fingerprint:=s.certificateSHA256;displayName:=s.displayName;s.mu.Unlock();writeJSON(w,http.StatusOK,map[string]any{"ok":true,"bridge":"vexstream-mobile-lan/v1","sourceVersion":s.version,"displayName":displayName,"certificateSha256":fingerprint,"canonicalAuthorization":false})}
func (s *Service) pairingInfo(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodGet{w.Header().Set("Allow",http.MethodGet);writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"GET required"});return};s.mu.Lock();defer s.mu.Unlock();s.cleanupPendingLocked();writeJSON(w,http.StatusOK,map[string]any{"displayName":s.displayName,"certificateSha256":s.certificateSHA256,"pairingCode":s.pairingCode,"pairingWindowOpen":time.Now().Before(s.pairingWindowUntil),"pairingWindowUntil":s.pairingWindowUntil.UTC().Format(time.RFC3339)})}
func (s *Service) pairingRequest(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodPost{w.Header().Set("Allow",http.MethodPost);writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"POST required"});return};var request struct{DeviceRef string `json:"deviceRef"`;DisplayName string `json:"displayName"`};if err:=decodeJSON(r,&request);err!=nil{writeJSON(w,http.StatusBadRequest,map[string]string{"error":"invalid pairing request"});return};request.DeviceRef=strings.TrimSpace(request.DeviceRef);request.DisplayName=strings.TrimSpace(request.DisplayName);if request.DeviceRef==""||request.DisplayName==""||len(request.DisplayName)>80{writeJSON(w,http.StatusBadRequest,map[string]string{"error":"deviceRef and displayName are required"});return};s.mu.Lock();defer s.mu.Unlock();s.cleanupPendingLocked();if !time.Now().Before(s.pairingWindowUntil){writeJSON(w,http.StatusForbidden,map[string]string{"error":"pairing window is closed"});return};for _,pending:=range s.pending{if pending.DeviceRef==request.DeviceRef&&pending.State=="PENDING"{writeJSON(w,http.StatusOK,map[string]any{"requestRef":pending.RequestRef,"state":pending.State});return}};requestRef,err:=randomRef("pair");if err!=nil{writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"pairing request unavailable"});return};now:=time.Now();expiresAt:=now.Add(pairingRequestLife);s.pending[requestRef]=&pendingPairing{RequestRef:requestRef,DeviceRef:request.DeviceRef,DisplayName:request.DisplayName,RequestedAt:now,ExpiresAt:expiresAt,State:"PENDING"};s.scheduleStopIfUnused(expiresAt);writeJSON(w,http.StatusOK,map[string]any{"requestRef":requestRef,"state":"PENDING"})}
func (s *Service) pairingRequestStatus(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodGet{w.Header().Set("Allow",http.MethodGet);writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"GET required"});return};requestRef:=strings.TrimSpace(r.URL.Query().Get("requestRef"));s.mu.Lock();defer s.mu.Unlock();s.cleanupPendingLocked();pending:=s.pending[requestRef];if pending==nil{writeJSON(w,http.StatusNotFound,map[string]string{"error":"pairing request not found"});return};response:=map[string]any{"requestRef":pending.RequestRef,"state":pending.State};if pending.State=="APPROVED"&&pending.Token!=""{response["token"]=pending.Token};writeJSON(w,http.StatusOK,response)}
func (s *Service) pairingRevokeSelf(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodPost{w.Header().Set("Allow",http.MethodPost);writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"POST required"});return};deviceRef:=s.deviceRefForRequest(r);if deviceRef==""{writeJSON(w,http.StatusUnauthorized,map[string]string{"error":"paired VexLife device required"});return};s.mu.Lock();defer s.mu.Unlock();delete(s.devices,deviceRef);if err:=s.saveLocked();err!=nil{writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"unable to save device removal"});return};writeJSON(w,http.StatusOK,map[string]any{"ok":true});go s.stopIfUnused()}
func (s *Service) authorized(next http.HandlerFunc)http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){if !s.authorizedRequest(r){w.Header().Set("WWW-Authenticate","Bearer");writeJSON(w,http.StatusUnauthorized,map[string]string{"error":"paired VexLife device required"});return};next(w,r)}}
func (s *Service) authorizedRequest(r *http.Request)bool{return s.deviceRefForRequest(r)!=""}
func (s *Service) deviceRefForRequest(r *http.Request)string{raw:=strings.TrimSpace(r.Header.Get("Authorization"));if !strings.HasPrefix(raw,"Bearer "){return ""};token:=strings.TrimSpace(strings.TrimPrefix(raw,"Bearer "));if token==""{return ""};sum:=sha256.Sum256([]byte(token));candidate:=hex.EncodeToString(sum[:]);s.mu.Lock();defer s.mu.Unlock();for deviceRef,device:=range s.devices{if len(candidate)==len(device.TokenHash)&&subtle.ConstantTimeCompare([]byte(candidate),[]byte(device.TokenHash))==1{return deviceRef}};return ""}

func (s *Service) mobilePairingStatus(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodGet{w.Header().Set("Allow",http.MethodGet);writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"GET required"});return};s.mu.Lock();defer s.mu.Unlock();s.cleanupPendingLocked();pending:=make([]map[string]any,0);for _,request:=range s.pending{if request.State!="PENDING"{continue};pending=append(pending,map[string]any{"requestRef":request.RequestRef,"displayName":request.DisplayName,"requestedAt":request.RequestedAt.UTC().Format(time.RFC3339)})};sort.Slice(pending,func(i,j int)bool{return pending[i]["requestedAt"].(string)<pending[j]["requestedAt"].(string)});paired:=make([]map[string]any,0,len(s.devices));for _,device:=range s.devices{paired=append(paired,map[string]any{"deviceRef":device.DeviceRef,"displayName":device.DisplayName,"approvedAt":device.ApprovedAt})};sort.Slice(paired,func(i,j int)bool{return paired[i]["displayName"].(string)<paired[j]["displayName"].(string)});writeJSON(w,http.StatusOK,map[string]any{"running":s.running,"displayName":s.displayName,"pairingCode":s.pairingCode,"pairingWindowOpen":time.Now().Before(s.pairingWindowUntil),"pairingWindowUntil":s.pairingWindowUntil.UTC().Format(time.RFC3339),"pending":pending,"paired":paired,"canonicalAuthority":false,"transportDescription":"Local VexStream pairing candidate; Home/Safety authority is not implied."})}
func (s *Service) mobilePairingStart(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodPost{w.Header().Set("Allow",http.MethodPost);writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"POST required"});return};if err:=s.EnsureStarted();err!=nil{writeJSON(w,http.StatusInternalServerError,map[string]string{"error":err.Error()});return};s.mu.Lock();s.pairingWindowUntil=time.Now().Add(pairingWindow);until:=s.pairingWindowUntil;code:=s.pairingCode;s.mu.Unlock();s.scheduleStopIfUnused(until);writeJSON(w,http.StatusOK,map[string]any{"ok":true,"pairingCode":code,"expiresAt":until.UTC().Format(time.RFC3339)})}
func (s *Service) mobilePairingDecision(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodPost{w.Header().Set("Allow",http.MethodPost);writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"POST required"});return};var request struct{RequestRef string `json:"requestRef"`;Decision string `json:"decision"`};if err:=decodeJSON(r,&request);err!=nil{writeJSON(w,http.StatusBadRequest,map[string]string{"error":"invalid pairing decision"});return};s.mu.Lock();defer s.mu.Unlock();s.cleanupPendingLocked();pending:=s.pending[strings.TrimSpace(request.RequestRef)];if pending==nil||pending.State!="PENDING"{writeJSON(w,http.StatusNotFound,map[string]string{"error":"pending pairing request not found"});return};switch strings.ToUpper(strings.TrimSpace(request.Decision)){case "DENY":pending.State="DENIED";writeJSON(w,http.StatusOK,map[string]any{"ok":true,"state":"DENIED"});go s.stopIfUnused();case "APPROVE":tokenBytes:=make([]byte,32);if _,err:=rand.Read(tokenBytes);err!=nil{writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"unable to form device token"});return};token:=base64.RawURLEncoding.EncodeToString(tokenBytes);tokenHash:=sha256.Sum256([]byte(token));approvedAt:=time.Now().UTC().Format(time.RFC3339);s.devices[pending.DeviceRef]=pairedDevice{DeviceRef:pending.DeviceRef,DisplayName:pending.DisplayName,TokenHash:hex.EncodeToString(tokenHash[:]),ApprovedAt:approvedAt};pending.State="APPROVED";pending.Token=token;if err:=s.saveLocked();err!=nil{delete(s.devices,pending.DeviceRef);pending.State="PENDING";pending.Token="";writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"unable to save paired device"});return};writeJSON(w,http.StatusOK,map[string]any{"ok":true,"state":"APPROVED"});default:writeJSON(w,http.StatusBadRequest,map[string]string{"error":"decision must be APPROVE or DENY"})}}
func (s *Service) mobilePairingRevoke(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodPost{w.Header().Set("Allow",http.MethodPost);writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"POST required"});return};var request struct{DeviceRef string `json:"deviceRef"`};if err:=decodeJSON(r,&request);err!=nil{writeJSON(w,http.StatusBadRequest,map[string]string{"error":"invalid revoke request"});return};s.mu.Lock();defer s.mu.Unlock();deviceRef:=strings.TrimSpace(request.DeviceRef);if _,ok:=s.devices[deviceRef];!ok{writeJSON(w,http.StatusNotFound,map[string]string{"error":"paired device not found"});return};delete(s.devices,deviceRef);if err:=s.saveLocked();err!=nil{writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"unable to save device removal"});return};writeJSON(w,http.StatusOK,map[string]any{"ok":true});go s.stopIfUnused()}

func (s *Service) saveLocked()error{payload,err:=encodeRememberedDevices(s.devices);if err!=nil{return err};path:=filepath.Join(s.root,"paired-devices.json");tmp:=path+".tmp";if err:=os.WriteFile(tmp,payload,0o600);err!=nil{return err};if _,err:=os.Stat(path);err==nil{if err:=os.Remove(path);err!=nil{return err}};return os.Rename(tmp,path)}
func decodeRememberedDevices(data []byte)([]pairedDevice,error){var stored pairingFile;if err:=json.Unmarshal(data,&stored);err!=nil{return nil,err};switch stored.SchemaVersion{case pairingSchemaV1,pairingSchemaCurrent:return stored.Devices,nil;default:return nil,fmt.Errorf("unsupported mobile pairing schema %q",stored.SchemaVersion)}}
func encodeRememberedDevices(devicesByRef map[string]pairedDevice)([]byte,error){devices:=make([]pairedDevice,0,len(devicesByRef));for _,device:=range devicesByRef{devices=append(devices,device)};sort.Slice(devices,func(i,j int)bool{return devices[i].DisplayName<devices[j].DisplayName});payload,err:=json.MarshalIndent(pairingFile{SchemaVersion:pairingSchemaCurrent,Devices:devices},"","  ");if err!=nil{return nil,err};return append(payload,'\n'),nil}
func (s *Service) scheduleStopIfUnused(until time.Time){delay:=time.Until(until);if delay<0{delay=0};go func(){timer:=time.NewTimer(delay+100*time.Millisecond);defer timer.Stop();<-timer.C;s.stopIfUnused()}()}
func (s *Service) stopIfUnused(){s.mu.Lock();s.cleanupPendingLocked();if !s.running||len(s.devices)>0||s.hasActivePendingLocked()||time.Now().Before(s.pairingWindowUntil){s.mu.Unlock();return};server:=s.server;mdns:=s.mdns;s.server=nil;s.listener=nil;s.mdns=nil;s.running=false;s.mu.Unlock();if mdns!=nil{mdns.Shutdown()};if server!=nil{_ = server.Close()}}
func (s *Service) hasActivePendingLocked()bool{now:=time.Now();for _,pending:=range s.pending{if pending.State=="PENDING"&&now.Before(pending.ExpiresAt){return true}};return false}
func (s *Service) cleanupPendingLocked(){now:=time.Now();for ref,pending:=range s.pending{if now.After(pending.ExpiresAt){delete(s.pending,ref)}}}

func ensureCertificate(certPath,keyPath string)(string,string,error){if certPEM,err:=os.ReadFile(certPath);err==nil{block,_:=pem.Decode(certPEM);if block!=nil{cert,parseErr:=x509.ParseCertificate(block.Bytes);if parseErr==nil{sum:=sha256.Sum256(cert.Raw);return hex.EncodeToString(sum[:]),pairingCodeFromDigest(sum),nil}}};privateKey,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader);if err!=nil{return "","",err};serialLimit:=new(big.Int).Lsh(big.NewInt(1),128);serial,err:=rand.Int(rand.Reader,serialLimit);if err!=nil{return "","",err};now:=time.Now();template:=&x509.Certificate{SerialNumber:serial,Subject:pkix.Name{CommonName:"VexStream Local Mobile Bridge",Organization:[]string{"Vextreme"}},NotBefore:now.Add(-5*time.Minute),NotAfter:now.AddDate(10,0,0),KeyUsage:x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment,ExtKeyUsage:[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},BasicConstraintsValid:true,DNSNames:[]string{"vexstream.local"}};der,err:=x509.CreateCertificate(rand.Reader,template,template,&privateKey.PublicKey,privateKey);if err!=nil{return "","",err};keyDER,err:=x509.MarshalPKCS8PrivateKey(privateKey);if err!=nil{return "","",err};if err:=os.WriteFile(certPath,pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE",Bytes:der}),0o600);err!=nil{return "","",err};if err:=os.WriteFile(keyPath,pem.EncodeToMemory(&pem.Block{Type:"PRIVATE KEY",Bytes:keyDER}),0o600);err!=nil{return "","",err};sum:=sha256.Sum256(der);return hex.EncodeToString(sum[:]),pairingCodeFromDigest(sum),nil}
func pairingCodeFromDigest(sum [32]byte)string{value:=binary.BigEndian.Uint32(sum[:4])%1000000;return fmt.Sprintf("%06d",value)}
func randomRef(prefix string)(string,error){raw:=make([]byte,12);if _,err:=rand.Read(raw);err!=nil{return "",err};return prefix+"."+hex.EncodeToString(raw),nil}
func decodeJSON(r *http.Request,value any)error{return json.NewDecoder(r.Body).Decode(value)}
func writeJSON(w http.ResponseWriter,status int,value any){w.Header().Set("Content-Type","application/json");w.Header().Set("X-Content-Type-Options","nosniff");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(value)}
