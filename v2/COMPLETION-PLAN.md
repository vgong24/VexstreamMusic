# VexStream Music Desktop V2 — Completion Plan

`[VXG RealForever]`

## Companion

Android completion companion:

```text
vgong24/VexLife-Android
PR #84
[VexStream V2][DRAFT] Android companion migration + completion
```

Android PR #84 is stacked on VexArchitecture PR #82. Desktop does not depend on Kotlin or Android
implementation details; the two repositories share semantic intentions and one producer/consumer
contract.


## Desktop-first release law

Victor is actively using VexStream Music 3.0 as the daily desktop player. Protect that working
product while V4 is formed.

```text
v2/
    = architecture-generation workspace

VexStream Music 3.0
    = protected current daily driver

VexStream Music 4.0.0
    = intended user-facing successor after the refactored desktop is independently accepted
```

The directory name is not the marketing version. Keep `v2/` as the temporary architecture workspace
unless a later source-placement reason earns a rename; ship the completed application as 4.0.0.

Ordering is strict:

```text
complete local desktop reconstruction
-> browser/media/persistence proof
-> qualified Windows 4.0.0 candidate
-> lived desktop acceptance
-> ONLY THEN resume Android/mobile integration
```

Android PR #84 is therefore a held companion/currentness surface until the Desktop Readiness Gate
below is satisfied. No Android product mutation is required to advance the desktop.

## End state

Desktop V2 is complete when VexStream is still recognizably the product already built, but its source
can be entered by human questions instead of reconstructing a monolith.

The lived outcome is:

```text
launch one clearly versioned VexStream desktop app
-> local desktop library works
-> playback/queue/import/Explore/playlists remain intact
-> pair VexLife once
-> Android browses a sanitized catalog
-> Android streams exact range-capable media
-> Android downloads and verifies selected media
-> Android keeps playing those bytes with the desktop/network gone
-> Android reconnects to the remembered producer without unnecessary re-pairing
-> when an admitted private route exists, Android can stream while away from Wi-Fi
```

The ordinary desktop administrative surface remains local/private.

## D0 — V2 foundation — formed

Current source already establishes:

```text
v2/README.md
v2/architecture/README.md
v2/quarry/V1-INTENTION-MAP.md
v2/web/architecture/behavior.js
v2/web/features/playback/
v2/web/foundation/text.js
v2/web/locales/en.js
```

Playback has a browser-provider Connection and pure provider/semantic tests.

This is architecture proof only; it does not yet replace V1 playback.

## D1 — real browser shell + proof harness — PROVED

Create the smallest real V2 browser surface that can prove the first migrated family.

Needed:
- a real HTMLAudioElement;
- Playback Connection mounting/lifetime;
- stable semantic screen/control identity where useful;
- TextRef -> locale rendering;
- browser interaction test entry;
- one human Journey declaration reused by later ScreenCrawler evidence.

Do not begin by reproducing every V1 screen.

The first browser proof should answer only:

```text
known media
-> Play/Pause
-> provider observation
-> semantic projection
-> rendered control
```

Then prove stale-render re-resolution in a real browser.

Checkpoint:

```text
code = 2f347403d13a18fac5c3e632a8b895f30a27ed26
Chromium 144
8 / 8 browser checks PASS
pageErrors = 0
consoleErrors = 0
real HTMLMediaElement Play -> Pause = PASS
feature-owned Journey + TextRef projection = PASS
```

This establishes the browser substrate for later desktop Journeys/ScreenCrawler. It does not yet prove the complete V4 desktop product.

## D2 — Playback + Queue parity — PROVED CORE

Migrate the actual desktop listening core without feature expansion.

Playback owns:
- provider play/pause/restart truth;
- playback-window enforcement where it is a playback concern;
- current semantic playback projection.

Queue owns independently meaningful:
- ordered track refs;
- current position;
- previous/next legality;
- play-next/append;
- reorder;
- shuffle;
- repeat;
- clear-after-current.

Higher composition owns:

```text
Play track / Play album / Play artist / Play playlist
-> establish queue context
-> select media
-> request playback
```

Preserve V1 behavior while removing hidden ownership crossover.

Exact checkpoint:

```text
d2468c415c5d037ee1463421c26b0b6caf2a59a1
focused Queue + Player tests = 22 / 22 PASS
Chromium checks             = 18 / 18 PASS
pageErrors                  = 0
consoleErrors               = 0
```

Covered behavior includes top-level Play context seeding, Play Track, Play Queue Item, Next,
Previous/restart threshold, shuffle-upcoming, repeat wrap, Play next, Add to queue, reorder, remove
guards and Clear after current.

One browser-discovered currentness correction is now explicit: when higher Player composition changes
the selected media source, the Playback Connection refreshes provider truth before resolving an
immediate Play request.

Queue persistence remains D4 work. V1 persists order without a coherent current-selection/session
contract; V4 will not canonize that accident as Queue semantics.

## D3 — Library / playlists / Explore / import semantic extraction

Migrate by question, not by source-file size.

### Library — CURRENT ACTIVE SLICE

Current extraction now has:
- canonical raw Track model;
- filesystem discovery + ID3/_vexmedia provenance;
- raw cache v2 with deliberate v1 rebuild;
- V3-compatible firstSeen / overlay / playback-window stores;
- human-facing projection separated from raw truth;
- one Library Service composing load / snapshot / reconcile.

Exact service checkpoint:

```text
3c7590441fc509f5877e7b98079d4ba2f6aa8ddc
service exact committed bytes
focused service package test = PASS
```

Next: one read-only HTTP/media adapter consumed by the V4 browser. Configuration/source mutation
remains a separate product-composition concern.

Preserve:
- canonical desktop catalog;
- sources;
- overlays;
- playback windows;
- trash/deleted history;
- durable reconciliation.

Keep filtering/sorting/search as projections unless they earn independent behavioral ownership.

### Playlists

Preserve desktop as current canonical writer.

Keep durable ordered track references and restoration behavior.

### Explore

Preserve:

```text
EXPLORE_SESSION != LIBRARY_QUEUE
PREVIEWED != DOWNLOADED
PROVIDER_RESULT != AI_RECOMMENDATION
```

Explore remains temporary unless explicitly promoted into the library.

### Import / acquisition

Preserve inspect -> duplicate review -> stage -> commit/discard and runtime readiness/repair.

Do not turn provider acquisition into an unstructured library write.

## D4 — desktop application composition + persistence boundary

Thin the executable composition root so startup does not own feature semantics.

Desired reading:

```text
cmd/startup
-> persistence/config
-> library service
-> feature services
-> local UI server
-> bounded mobile service
-> browser app
```

Preserve:
- version + buildRef process identity;
- explicit Exit VexStream;
- local app data compatibility;
- current library without migration loss.

### Persistence compatibility law

The October 2026 mobile-pairing regression exposed a concrete desktop consistency bug:

```text
newer desktop
-> reads pairing store v1
-> writes pairing store v2

protected older desktop
-> reads v1 only
-> later encounters shared v2 state
-> startup/mobile bridge fails
```

Therefore V4 treats durable schema evolution as a compatibility boundary, not merely an
implementation detail.

```text
PERSISTED_STATE_UPGRADE
!=
PERMISSION_TO_BREAK_ROLLBACK
```

For every durable store that may be touched by more than one desktop build, one of these must be
explicitly proved before release:

1. old and new builds can both read the written representation;
2. the new writer preserves fields it does not own while writing a backward-readable envelope;
3. incompatible schemas use side-by-side/versioned custody so the protected previous release is not
   forced to consume newer state;
4. a deliberate irreversible migration exists with an explicit rollback boundary and user-visible
   release transition.

`version + buildRef` prevents stale-process impersonation; it does **not** protect shared persisted
state by itself.

The discovered pairing case is retained as the pressure example:

```text
vexstream.mobile.local-pairing/v1
-> vexstream.mobile.local-pairing/v2

FORWARD_READ_COMPATIBILITY existed
ROLLBACK_READ_COMPATIBILITY did not
```

V4 persistence tests must include **new-write -> protected-previous-read** or an explicit
side-by-side custody proof wherever rollback support is claimed.

V1 remains runnable/quarry until V2 proves replacement.

## D5 — mobile producer contract in V2 — DEFERRED UNTIL DESKTOP READINESS

Desktop owns the producer contract.

The mobile-facing service is separate from the desktop administration API.

Minimum contract:

```text
health / identity
sanitized catalog
range-capable media GET/HEAD
explicit playlist projection when admitted
pairing/session endpoints only where owned
```

Catalog carries:
- opaque current track ref;
- exact media SHA-256;
- exact byte length;
- sanitized title/artist/album/genre/year/duration;
- playback window;
- ordered playlist refs when present.

Never expose:
- raw desktop path/source folder;
- provider credentials;
- arbitrary file access;
- import/trash/repair/admin operations;
- shell/process/model proxy.

Keep:

```text
HUB_TRACK_REF != PORTABLE_MEDIA_IDENTITY
MEDIA_STREAM != MEDIA_COPY
```

Android PR #84 consumes this contract.

## D6 — connection / authorization / route separation — DEFERRED UNTIL DESKTOP READINESS

Same-Wi-Fi pairing already has useful V1 behavior and should be migrated without becoming the final
network architecture.

Separate:

```text
producer identity
pairing / human approval
music capability
current route/session
music request
```

Remote PR #3/#4 remain quarry for:
- signed/encrypted request envelopes;
- direct route experiments;
- content-blind relay fallback;
- replay/freshness/rate/concurrency controls.

They do not become V2 truth merely because source exists.

Desired V2 boundary:

```text
MusicService
    knows authorized semantic request

RouteAdapter
    carries it through an admitted session
```

A future canonical Vex Home/private transport must be replaceable without rewriting catalog,
playback, download, or library semantics.

No widening of the ordinary admin listener is acceptable.

## D7 — localization / design language / Journey / ScreenCrawler

This remains part of the desktop reconstruction and should be completed before Android resumes.

As V2 screens migrate:

- authored human text uses stable TextRef identity;
- product-supported language authority remains distinct from crawler test locales;
- stable semantic controls/screens feed browser automation;
- one Journey language describes human paths;
- ScreenCrawler reuses Journeys under selected locale/font/viewport/theme conditions;
- deterministic visual references represent meaningful named states only.

ScreenCrawler is evidence acquisition, not the product oracle.

Do not introduce a framework before the first real migrated screen needs it.

## D8 — Desktop 4.0 proof, packaging, and readiness gate

Proof ladder:

```text
semantic/unit
-> provider adapter
-> browser interaction
-> real media/range server
-> persistence/process lifecycle
-> Windows executable qualification
-> Android cross-repo integration
-> lived user acceptance
```

Release packaging deliberately produces a visibly fresh application identity:

```text
marketing version = 4.0.0
fresh buildRef
fresh Windows executable/package
```

Do not repeatedly bump the marketing version during architecture work. Keep V3 available while V4
is formed and qualified.

The V4 candidate must prove the complete local desktop experience before cross-repository work
resumes: library, playback/queue, browse/search, playlists, Explore/import, persistence/session
behavior, explicit Exit, and clean relaunch without stale V3/V4 process confusion.

When exact Windows-host proof is required, the current accepted VexRelay/VexLocalBridge route may
be used to bind the exact source head and return build/runtime evidence. Relay carries/proves the
effect; it does not own VexStream semantics.

**Desktop Readiness Gate:** after exact-head Windows qualification, Victor runs the qualified V4
candidate as an ordinary music player while V3 remains available. Android implementation remains
held until that lived desktop acceptance succeeds.

## Cross-repository join gates — begin only after Desktop Readiness

These gates are intentionally downstream of D8. They do not block the desktop from becoming a
complete local V4 product first.

### J1 — local producer/consumer parity

```text
Desktop V2 producer
-> Android catalog
-> stream
-> range seek
-> queue/playback semantics
```

### J2 — offline

```text
Android Download
-> exact length + SHA-256 verified
-> desktop unavailable
-> LOCAL_ONLY play
-> Android restart
-> local copy restored
```

Desktop proves exact producer bytes/range behavior; Android proves local custody.

### J3 — pairing/reconnect

```text
pair once
-> remembered producer identity
-> route/address change
-> reconnect without unnecessary approval
-> revoke/forget fails closed
```

### J4 — away from home

Only after a current private route/session predicate is real:

```text
phone Wi-Fi OFF
mobile data ON
-> authenticated/authorized producer reached
-> catalog
-> stream
-> seek
```

Then:

```text
mobile data OFF
-> downloaded media still plays
```

No public desktop admin exposure is an acceptable shortcut.

## Definition of done

Desktop PR is complete when:

1. V2 is the buildable/runnable desktop product, not an architecture sidecar;
2. V1 behavior that remains product intent has been explicitly migrated or consciously retired;
3. V1 remains historical/quarry evidence rather than a second live semantic owner;
4. feature neighborhoods are navigable by human question;
5. producer/mobile boundaries are read-only and capability-bounded;
6. localization/Journey/browser proof are source-owned and understandable;
7. Windows 4.0.0 process/build/package proof is green;
8. the lived Desktop Readiness Gate passes while V3 remains available;
9. the final executable has a fresh version + buildRef and cannot be confused with V3;
10. only after desktop acceptance do J1-J3 resume with Android PR #84; J4 remains dependent on a real private route.

<!-- [VXG RealForever][VEXSTREAM][DESKTOP-V2][COMPLETION] -->
