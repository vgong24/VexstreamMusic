# VexStream Mobile Bridge v1 — producer contract

`[VXG RealForever]`

## Why this exists

VexStream desktop is the canonical music-library/media producer. VexLife Android is a separate
consumer that needs to stream those media bytes to the phone and retain selected tracks for verified
offline playback.

This document is the source-addressable join between the two repositories. It prevents a future
worker from reconstructing the bridge from chat history or from exposing VexStream's existing
localhost administrative API merely because Android needs network access.

## Repository relationship

```text
producer
  repository = vgong24/VexstreamMusic
  source family = mobile_bridge.go
  formation branch = VXG-101126-vexstream-mobile-bridge

consumer
  repository = vgong24/VexLife-Android
  source family = app/src/main/kotlin/com/vextreme/vexlife/android/data/vexstream/
  formation branch = VXG-101126-vexstream-mobile-stream-offline
```

Formation branches are current work coordinates, not permanent semantic ownership. After normal
review/merge, each repository's accepted source becomes authoritative.

The VexStream 2.2 Explore candidate remains a separate product concern. Mobile bridge work was
branched from its exact 2.2 source and does not change the Explore acceptance question.

## Existing desktop boundary

The ordinary VexStream server remains loopback-only and owns the full local desktop product:

```text
127.0.0.1:<ordinary dynamic port>
  UI
  library administration
  imports
  playlists
  diagnostics
  repair/session operations
  /media/<desktop track id>
```

Do not widen that listener to the LAN merely to connect Android.

## Mobile projection

The mobile bridge exposes only:

```text
GET  /v1/health
GET  /v1/catalog
GET/HEAD /v1/media/<trackRef>
```

No mutating endpoint is admitted by v1.

The catalog contains only:

```text
hubRef
sourceVersion
catalogRevision

tracks[]
  trackRef
  mediaSha256
  mediaByteLength
  title
  artist
  album
  genre
  year
  durationSeconds
  playbackStartSeconds
  playbackEndSeconds
```

It deliberately excludes desktop paths, filenames, source-folder locations, provider/private
configuration, credentials, raw Home/model endpoints, import controls, diagnostics, repair controls,
and arbitrary administrative actions.

## Identity

Current VexStream desktop track IDs are derived from normalized file paths. Therefore:

```text
trackRef
  = current Hub catalog address
  != portable media identity

mediaSha256 + mediaByteLength
  = exact media-byte identity/integrity binding
```

Android retains verified offline copies by media digest rather than assuming a desktop path-derived
track ref is permanent.

## Development transport

The current first-lived-proof transport is intentionally not the production Home gateway.

VexStream listens on:

```text
127.0.0.1:18765
```

for this mobile-development surface (overridable through `VEXSTREAM_MOBILE_DEV_PORT`).

A physical Android device may reach that loopback listener through an explicit developer transport
such as ADB reverse:

```text
phone localhost:18765
  -> adb reverse
  -> desktop 127.0.0.1:18765
```

This proves the product data/playback relationship without granting LAN exposure or inventing final
pairing credentials.

```text
ADB_REVERSE_REACHABILITY
!= PAIRING
!= AUTHENTICATION
!= AUTHORIZATION
!= PRODUCT_REMOTE_GATEWAY
```

## Product transport

The eventual real phone/Home connection must consume the accepted Home/Safety relationship:

```text
reachability
!= authentication
!= authorization
paired membership
!= active capability lease
```

The consumer currently identifies three required capability classes:

```text
CATALOG_READ
MEDIA_STREAM
MEDIA_COPY
```

Those names are Android consumer intentions, not accepted canonical security refs yet. The rightful
Home/Safety owner must admit/mint the real capability identities before a product bridge claims
authorization.

For Android target SDK 37, prefer the narrow system-mediated local-service selection route where it
fits rather than asking for blanket LAN access by default. Remote/off-home streaming must not widen
or port-forward the ordinary desktop application/admin surface. A dedicated remote ingress may expose
only the already-sanitized mobile capability while preserving the distinction between reachability and
canonical Home/Safety authority.

## Offline invariant

A phone copy becomes downloaded only after Android independently verifies:

```text
expected byte length
+ expected SHA-256
+ atomic local completion
```

The desktop catalog's declaration is the expected binding; Android's local verification establishes
that the received bytes match it.

Hub loss or catalog reorganization must not erase a valid local copy.

## First end-to-end proof

```text
desktop VexStream running
-> mobile bridge health readable through dev transport
-> Android reads sanitized catalog
-> Android streams one track to phone playback
-> Android downloads that track
-> downloaded bytes verify exact length + SHA-256
-> desktop transport disappears
-> same track remains playable from phone-local storage
-> reconnect does not duplicate the identical media object
```

Only after this proof should the transport itself be promoted toward LAN/Home product realization.

<!-- [VXG RealForever][VEXSTREAM][MOBILE-BRIDGE][V1] -->

## Nearby pairing successor to the development transport

The ADB-reverse path has now produced a lived physical-device proof and remains a recovery route.
The next local product candidate does not widen the ordinary localhost administration server.

A separate TLS listener is admitted only for the mobile projection:

```text
_vexstream._tcp
TCP 18766
HTTPS
```

Pairing is explicitly opened from the localhost VexStream UI for five minutes. mDNS advertises the
full SHA-256 fingerprint of a persistent self-signed P-256 certificate. The selected Android consumer
pins that exact certificate before it sends pairing data.

Human flow:

```text
desktop: Connect a phone
phone:   Find VexStream
both:    compare six-digit certificate-derived confirmation code
phone:   Request access
desktop: Allow / Deny
```

On Allow, VexStream forms a random per-device bearer token. The desktop persists only its SHA-256
hash; Android retains the token under Android Keystore encryption. The bearer protects only the
sanitized catalog/media surface.

Remembered route identity is the pinned desktop certificate, not the current LAN IP address. A future
DHCP route change can therefore rediscover the same desktop and update the route without re-pairing.

This remains a bounded local VexStream pairing candidate:

```text
LOCAL_VEXSTREAM_PAIRING
!= VEXHOME_MEMBERSHIP
!= CANONICAL_HOME_AUTHN
!= CANONICAL_HOME_AUTHZ
!= CAPABILITY_LEASE
!= REMOTE_ACCESS_AUTHORITY
```

The rightful Home/Safety owner remains required before this mechanism can claim remote/off-home
authority.

## Playlist projection

VexStream 3.0 adds desktop-owned playlists to the same sanitized catalog response:

```text
playlists[]
  playlistRef
  name
  trackRefs[]
```

Only track refs present in the current projected mobile catalog are emitted. Desktop timestamps,
filesystem state, source paths, and playlist mutation APIs are not exposed.

Current ownership is deliberately one-way:

```text
DESKTOP_PLAYLIST = CANONICAL_WRITER
ANDROID_PLAYLIST_PROJECTION = READ_ONLY_BROWSE_AND_PLAY
```

Phone-local playlist authoring and explicit sync may be added later, but this contract does not
invent automatic two-way conflict resolution.


## Remote/off-home successor — bounded content-blind relay

The same-Wi-Fi bridge remains the preferred route when the paired desktop is directly reachable.
The off-home successor adds reachability without exposing the ordinary VexStream UI/admin listener
or sending the mobile bearer/media plaintext to a public relay.

Source placement:

```text
mobile_lan_bridge.go
  local approval + bearer authorization
  existing paired-device record
  device signing public key

mobile_remote_crypto.go
  stable desktop remote-gateway ECDH identity

mobile_remote_gateway.go
  outbound relay registration/polling
  verifies paired phone signature again
  decrypts the inner request
  enforces read-only VexStream mobile path allowlist
  forwards only to the existing local TLS mobile listener
  encrypts the response back to the phone

cmd/vexstream-relay/
  public in-memory rendezvous/relay
  stores no music, bearer, Home payload, or private key
```

Local pairing now carries the Android device's P-256 signing public key. The private signing key stays
in Android Keystore. Already-paired v1 phones may upgrade while directly reachable through the
existing bearer-authorized `/v1/pairing/remote-upgrade` endpoint; changing an already-bound signing
key requires an explicit re-pair.

The desktop keeps a stable P-256 ECDH gateway private key in the same app-private mobile-bridge state
area as its existing TLS private key. Android receives only the public key.

For a remote request:

```text
phone
  signs routing envelope with paired device key
  forms ephemeral P-256 ECDH key
  derives request key with desktop gateway public key
  AES-GCM encrypts:
    VexStream bearer
    method/path
    bounded headers
  -> relay

relay
  verifies the phone signature against the currently registered public key
  queues only ciphertext
  -> desktop outbound gateway

desktop
  verifies device signature independently
  derives the same ECDH request key
  decrypts request
  permits only:
    GET /v1/catalog
    GET/HEAD /v1/media/<trackRef>
  forwards to its existing pinned/local mobile HTTPS surface
  AES-GCM encrypts bounded response
  -> relay -> phone
```

Media uses bounded Range requests rather than one whole-track relay object. LAN remains first choice;
remote relay is fallback; verified Android-local bytes remain the final offline fallback.

The public relay itself still needs an actual HTTPS deployment. Source presence or a configured URL
does not prove that deployment/currentness.

This is an app-scoped remote-access candidate and deliberately does not relabel itself as accepted
canonical VexHome/P7 authority:

```text
SIGNED_PAIRED_DEVICE != CANONICAL_HOME_MEMBERSHIP
REMOTE_RELAY_REACHABILITY != CANONICAL_REMOTE_AUTHORIZATION
READ_ONLY_VEXSTREAM_REMOTE_CAPABILITY != GENERAL_PC_REMOTE_CONTROL
RELAY_ROUTING_METADATA != HOME_CUSTODY
RELAY_CIPHERTEXT != RELAY_MEDIA_OWNERSHIP
```

The accepted Home/Safety contracts remain the evolutionary destination for reusable cross-product
remote membership, capability leases, revocation generations and shared Home gateway semantics.

## Direct-PC successor — public paired-device ingress

The Home PC is the real VexStream host. A public relay is therefore not the preferred media path when
the already-paired phone can reach that PC directly.

The direct successor adds a second dedicated TLS listener:

```text
TCP 18767
HTTPS
POST /v1/direct/request
```

This listener is deliberately separate from both:

```text
ordinary desktop/admin listener
LAN pairing + catalog listener on 18766
```

It does **not** expose pairing, desktop UI, filesystem paths, uploads, playlist mutation, diagnostics,
repair, shell/process operations, arbitrary URLs, or general Vex computer control.

A direct request reuses the same remote cryptographic identity already formed during local approval:

```text
Android Keystore P-256 device signing key
+ desktop stable P-256 gateway key
+ signed routing envelope
+ ephemeral ECDH
+ HKDF
+ AES-GCM inner request
```

The public ingress validates the already-enrolled device signature before decrypting the inner request,
then independently applies:

```text
fresh issuedAt window
bounded replay cache keyed by deviceRef + requestRef
bounded request body
bounded source-IP request rate
bounded concurrent handlers
exact Home ref
exact paired device key
read-only mobile allowlist
```

Only these decrypted operations remain admitted:

```text
GET /v1/catalog
GET/HEAD /v1/media/<trackRef>
```

The response is encrypted with the same per-request ECDH-derived key before leaving the desktop
process.

### Direct route formation

VexStream may advertise direct routes only while the dedicated ingress is running for at least one
remote-enabled paired device.

Current candidate route formation is:

```text
global IPv6 address, when present
OR
temporary UPnP TCP mapping -> local 18767, when the router returns a genuinely public IPv4 address
```

UPnP mappings use bounded leases and are renewed while VexStream is running. RFC1918, loopback,
unspecified, multicast/reserved, and CGNAT `100.64.0.0/10` addresses are not advertised as public
routes. Revoking the final remote-enabled device stops the ingress and removes the active mapping
best-effort.

An optional `VEXSTREAM_DIRECT_PUBLIC_URL` remains a development/explicit-route override; it is not
required by the intended product flow.

The Android consumer learns direct route candidates only through its already-pinned local relationship
and still pins the exact desktop certificate when using those public addresses.

Route preference becomes:

```text
same-LAN pinned TLS
-> direct paired-device ingress on Home PC
-> content-blind VexRelay fallback
-> verified phone-local copy
```

Therefore:

```text
PUBLIC_ROUTE != AUTHORITY
UPNP_MAPPING != PAIRING
DIRECT_TLS_REACHABILITY != GENERAL_PC_ACCESS
DIRECT_VEXSTREAM_CAPABILITY != CANONICAL_HOME_AUTHORIZATION
VEXRELAY_FALLBACK != MEDIA_OWNERSHIP
```

The direct path changes reachability, not the ownership of the music, device identity, or future
canonical Home/Safety authority.

