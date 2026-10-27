# V1 Intention Quarry Map

`[VXG RealForever]`

This file is a map for V2 migration. It is not a claim that V1 should be copied literally.

## Bound source checkpoint

V2 formation is anchored to:

```text
commit = c8732f35f4f173952dd6df24ee7931557b9de274
product = VexStream Music 3.0.0
Windows CI run #63 = SUCCESS
```

The V1 implementation remains at repository root.

## Why V2 was earned

At the bound checkpoint, the source carries substantial meaning inside concentrated files:

```text
main.go
    ~4.5k lines
    startup / config / library / playlists / import / diagnostics /
    repair / sessions / HTTP routing / product orchestration

ui/index.html
    ~200 KB
    one embedded browser surface
    hundreds of JavaScript functions
    product state + rendering + interaction + playback/queue behavior
```

Large files are not themselves the reason for V2. The reason is that independent behavioral questions
now require manual reconstruction from those files.

## V1 semantic neighborhoods

### Process/session identity

Human intention:

> Launch the build I asked for; do not silently reopen an older same-version process.

V1 truth:

```text
startup reuse identity = version + buildRef
browser close != process exit
explicit System status -> Exit VexStream exists
```

Keep this distinction in V2.

### Library

Human intention:

> My music library is durable local truth; views and filters should not redefine what exists.

V1 source includes:

- catalog loading/reconciliation;
- source folders;
- overlays;
- playback windows;
- trash/deleted history;
- cached library projection.

V2 should separate canonical library truth from presentation/filter lenses.

### Playback

Current V1 browser source:

```text
<audio id="audio">
togglePlayback()
playIndex()
next()
prev()
audio.onplay / onpause / onended / ontimeupdate
renderPlayer()
```

Important existing behavior:

- Play/Pause observes `audio.paused`;
- selecting media binds `/media/<track id>`;
- playback-window start/end is enforced;
- end advances to next queue item;
- repeat may wrap;
- render state reads the live audio element.

Architectural pressure discovered during V2 formation:

```text
togglePlayback() with no current track
-> chooses a product/library context
-> seeds Queue
-> starts Playback
```

That behavior should survive, but the ownership should become explicit:

```text
NO_CURRENT_MEDIA_PLAY
    = composition of Queue/selection + Playback
    != pure Playback meaning
```

### Queue

V1 owns:

- ordered track IDs;
- current index;
- append / play-next;
- shuffle upcoming;
- repeat;
- drag reorder;
- clear-after-current;
- history/current/up-next visual projection.

Queue has enough independent truth to become its own V2 behavioral question.

### Explore

V1 2.2/3.0 preserves a temporary metadata exploration session:

```text
EXPLORE_SESSION != LIBRARY_QUEUE
PREVIEWED != DOWNLOADED
PROVIDER_RESULT != AI_RECOMMENDATION
```

Do not collapse Explore into playback or durable library state.

### Playlists

Desktop is the canonical playlist writer for the current V1 milestone.

Playlist order is durable and may retain references to unavailable/deleted tracks so restoration can
reconnect them.

### Import/acquisition

V1 import behavior includes provider search, inspect, duplicate review, staging, commit/discard, and
runtime readiness/repair.

Do not turn provider acquisition into a generic Library mutation without preserving those stages.

### Diagnostics / repair

V1 has explicit diagnostic session, event, run, download, system preflight, repair plan, and session
retirement surfaces.

V2 diagnostics should project truth owned elsewhere rather than become another semantic owner.

### Mobile producer / local pairing

At 3.0 formation:

```text
ordinary UI/admin listener
    = loopback only

mobile surface
    = separate bounded HTTPS service
    = sanitized catalog + media
    = explicit pairing/approval
```

Portable media identity is:

```text
mediaSha256 + mediaByteLength
```

not the path-derived desktop track ID.

Keep:

```text
REACHABILITY != AUTHENTICATION
AUTHENTICATION != AUTHORIZATION
MEDIA_STREAM != MEDIA_COPY
```

### Remote experiments after the V2 base

These remain quarry only:

```text
PR #3
    content-blind remote relay / encrypted envelope candidate

PR #4
    direct paired-device ingress / IPv6 + temporary UPnP candidate
```

V2 must not treat either transport as accepted merely because source exists.

## Proof map

Known V1 proof classes include:

```text
pure Go regression
static source invariants
Chromium interaction
real media server/range delivery
Windows PE cross-build qualification
lived Windows process behavior
physical Android/mobile observations
```

When migrating a behavior, carry the proof that actually owns the claim. Do not substitute one green
surface for another.

## Migration discipline

For each V2 slice:

```text
QUESTION
-> V1 behavior/source
-> observed/accepted evidence
-> semantic owner
-> V2 family
-> provider Connection
-> presentation
-> parity/new-intention tests
-> browser/host proof when required
```

Do not delete or rewrite the V1 path until the V2 path has earned replacement.

<!-- [VXG RealForever][VEXSTREAM][DESKTOP-V2][V1-QUARRY] -->
