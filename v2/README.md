# VexStream Music Desktop V2 Architecture Workspace

`[VXG RealForever]`

## The question

> Can VexStream Desktop preserve the geometry of human intention closely enough that a future
> developer or AI can find one behavioral question, understand its current truth, change it, and
> prove it without reconstructing the entire product from a monolith?

`v2/` is the architecture-generation workspace for the intended user-facing **VexStream Music
4.0.0** successor. VexStream Music 3.0 remains the protected daily driver while this work continues.

## Entry route

1. this file;
2. `architecture/README.md`;
3. `COMPLETION-PLAN.md`;
4. `quarry/V1-INTENTION-MAP.md` when translating existing behavior;
5. the exact feature family being changed;
6. its proof.

## Stable architecture law

```text
STATE   what is true now
CONTROL stable human-facing affordance
ACT     what Control means from current State
EFFECT  runtime transformation to attempt
```

```text
EFFECT_ATTEMPTED != PROVIDER_SUCCEEDED
```

Provider truth remains at provider boundaries. Split when the behavioral question becomes
independently meaningful, not because a file becomes large.

## Desktop-first posture

```text
VexStream Music 3.0
    = protected current daily driver

v2/
    -> complete local desktop V4
    -> browser/media/persistence/Windows proof
    -> VexStream Music 4.0.0 candidate
    -> lived desktop acceptance
    -> only then resume Android/mobile integration
```

Android PR #84 remains HOLD.

## D1 — Playback browser boundary — PROVED

Playback is the first real family. It owns provider play/pause/restart meaning without owning Queue.

First browser checkpoint:

```text
2f347403d13a18fac5c3e632a8b895f30a27ed26
Chromium 144
8 / 8 checks PASS
0 page errors
0 console errors
real HTMLMediaElement Play -> Pause PASS
Journey + TextRef projection PASS
```

## D2 — Playback + Queue — PROVED CORE

Queue is now an independent semantic neighbor to Playback.

Queue owns:
- ordered media refs;
- current position;
- Next / Previous queue legality;
- shuffle upcoming;
- repeat/wrap;
- Play next / Add to queue;
- reorder/remove with current identity preserved;
- Clear after current.

Higher Player composition owns cross-family human actions:

```text
top-level Play with no current media
-> establish current library context
-> Queue
-> selected media
-> Playback

Play Track / Play Queue Item
-> Queue selection/context
-> Playback realization

Previous
-> elapsed > 4 sec: restart current Playback
-> near start: Queue Previous when available
```

Important implementation discovery:

```text
changing selected media source
-> Playback provider truth must refresh explicitly
-> immediate Play may then resolve from current truth
```

This is owned by the Playback Connection through `refresh()`; Queue/UI do not infer Playback truth.

Exact D2 checkpoint:

```text
d2468c415c5d037ee1463421c26b0b6caf2a59a1

focused Queue + Player tests = 22 / 22 PASS
Chromium browser checks       = 18 / 18 PASS
page errors                   = 0
console errors                = 0
repository MP3 fixture        = exact bytes
changed browser/composition blobs = byte-for-byte matched before commit
```

Queue persistence is intentionally deferred to D4. V1 persists queue order without a clean current
selection model; V4 should define startup/listening restoration at the persistence boundary rather
than make that accident a Queue semantic law.

## Current edge — D3 Library first

The next question is:

> What music does VexStream actually know, and who owns that truth?

Start from the V1 backend, not the browser renderer:

```text
sources / catalog
-> durable library reconciliation
-> overlays + playback windows
-> trash/deleted history
-> library projection

then:
playlists
Explore
import/acquisition
```

The next real browser catalog should consume a V4 desktop Library boundary rather than hard-coded
fixture tracks.

<!-- [VXG RealForever][VEXSTREAM][DESKTOP-V2-WORKSPACE][V4][D2-PROVED] -->
