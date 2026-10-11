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
fits rather than asking for blanket LAN access by default. Remote/off-home streaming belongs behind
the accepted VexHome gateway/reachability path, not port forwarding of the desktop application.

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

