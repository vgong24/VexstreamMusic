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

## D1 — real browser shell + proof harness

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

## D2 — Playback + Queue parity

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

## D3 — Library / playlists / Explore / import semantic extraction

Migrate by question, not by source-file size.

### Library

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

V1 remains runnable/quarry until V2 proves replacement.

## D5 — mobile producer contract in V2

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

## D6 — connection / authorization / route separation

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

As V2 screens migrate:

- authored human text uses stable TextRef identity;
- product-supported language authority remains distinct from crawler test locales;
- stable semantic controls/screens feed browser automation;
- one Journey language describes human paths;
- ScreenCrawler reuses Journeys under selected locale/font/viewport/theme conditions;
- deterministic visual references represent meaningful named states only.

ScreenCrawler is evidence acquisition, not the product oracle.

Do not introduce a framework before the first real migrated screen needs it.

## D8 — proof and packaging

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

Release packaging should deliberately produce a visibly fresh application identity.

Choose the final marketing version at release time; do not repeatedly bump it during architecture work.

The release candidate must have a new buildRef and final version so an old session cannot impersonate
the new V2 executable.

## Cross-repository join gates

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
7. Windows process/build/package proof is green;
8. J1-J3 pass with Android PR #84;
9. J4 passes when its private-route dependency is actually available;
10. the final executable has a fresh version + buildRef and can replace the old session cleanly.

<!-- [VXG RealForever][VEXSTREAM][DESKTOP-V2][COMPLETION] -->
