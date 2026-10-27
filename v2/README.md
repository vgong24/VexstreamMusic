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
2. `LIVED-PRODUCT-ORACLE.md`;
3. `architecture/README.md`;
4. `COMPLETION-PLAN.md`;
5. `quarry/V1-INTENTION-MAP.md` when translating existing behavior;
6. the exact V3 screen/human question being preserved;
7. the exact V4 feature family being changed;
8. its proof.

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

## Lived-product correction

The first exact Windows-qualified candidate passed engineering qualification but failed Victor's
lived Desktop Readiness because its browser composition had become a different product shell.

```text
V3 lived product = product oracle
V4 owners/services = refactor substrate
V4 practicum UI = proof scaffold, not replacement authority
```

Enter `LIVED-PRODUCT-ORACLE.md` before changing browser/product composition.

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

## D3 — Library truth + reconciliation — IN PROGRESS

Current source separates:

```text
raw discovered Track truth
-> filesystem scan/cache realization
-> durable firstSeen identity
-> human Overlay + PlaybackWindow state
-> projected desktop catalog
```

A composed Library Service now owns the reconciliation question without taking over source-folder
configuration, browser/API, playlists, import, or playback.

Exact service checkpoint:

```text
3c7590441fc509f5877e7b98079d4ba2f6aa8ddc
committed service blobs reconciled byte-for-byte
go test ./v2/desktop/library/service = PASS
```

Durability correction earned during extraction:

```text
persist firstSeen
-> persist rebuildable raw cache
```

so a cache-write failure cannot cause an already-observed song to silently receive a new AddedAt
identity on retry. Failed reconciliation does not replace the in-memory raw catalog.

### Current edge — real read-only Library/media boundary

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

Smallest next seam:

```text
Library Service
-> read-only catalog HTTP adapter
-> exact range-capable media reads
-> V4 browser catalog
```

Source-folder mutation/admin behavior remains outside this adapter.

<!-- [VXG RealForever][VEXSTREAM][DESKTOP-V2-WORKSPACE][V4][D3-LIBRARY] -->


## Current checkpoint — D4 persistence and executable qualification

The architecture workspace has crossed the earlier D3 extraction boundary.

Current local product source now includes:

```text
V4 executable entry
-> exact version + binary buildRef identity
-> V4-only current-session claim
-> local app composition
   -> Config / source discovery
   -> Library + exact range media
   -> reversible overlays / playback windows
   -> Playlists
   -> Trash + journaled purge recovery
   -> Explore session + provider search
   -> provider runtime setup/repair
   -> Import inspect / duplicate review / staged preparation
   -> crash-recoverable Import commit + retry/discard/recovery
   -> Player session persistence
   -> embedded browser
   -> explicit Exit
```

The latest Chromium journey checkpoint before D4-only Go changes is:

```text
4cbd4b782d9ee3b83172745b65adc1e79fdc3b57
Library view/admin + Playlist CRUD + Trash lifecycle
+ Playback/Queue + Explore/Import + player-session + source-management walk
```

### Current proof topology — source meaning through lived browser

V4 uses the same proof principle as the accepted VexLife Android test foundation without copying
Gradle/Android mechanics:

```text
feature/domain semantics
  -> focused Go / Node tests at the owning transformation

browser presentation contract
  -> deterministic DOM/API/media fixtures

human interaction walk
  -> dependency-free Chromium DevTools lived-shell smoke

broader browser regression
  -> Playwright qualification when that environment is available

real Windows/macOS product feel
  -> exact built artifact + lived acceptance
```

These layers do not replace one another. A duplicate-scoring Go PASS does not prove that the browser
shows the duplicate to the listener, and a browser PASS does not prove that Windows provider
subprocesses stay hidden on a real host.

Repository-owned V4 entries are:

```bash
cd v2

# all current Node semantic/model tests
npm test

# focused Import planning/request semantics
npm run test:imports

# Import semantics + the real integrated browser walk
npm run verify:imports

# aggregate local V4 source verification
# gofmt -> Go tests -> native-host vet -> Node semantic/model suite -> lived Chromium walk
npm run verify:local
```

`test:lived` uses the repository's dependency-free CDP runner and discovers common Chromium-family
browsers on Windows (Chrome, Edge, Brave), Linux and macOS. An explicit browser path remains available
for unusual installations.

`verify:local` is the canonical developer/source entry, analogous in purpose to Android's
`vexVerifyLocal`: CI, VexRelay, or a developer terminal should call this source-owned command rather
than restating the sequence in another workflow. It records its own proof limits: native-host vet is
not cross-platform release qualification, deterministic browser fixtures are not a live YouTube
provider, and automated PASS is not human product acceptance.

The human no-Git Windows source launcher intentionally does not acquire Node merely to run browser
automation. It continues to own its Go/source/Windows build gate. Browser proof is a developer/source
proof surface, just as Android Journey/device proof remains distinct from JVM/source proof.

### Current E2E boundary

The V4 proof stack is strong enough to catch **feature disappearance inside the browser product**:
the lived-shell journey executes the actual embedded V4 HTML/JS in Chromium and asserts Library,
Playback/Queue, Home track actions, Playlists, Trash, Discover/Explore/YouTube Import, player-session,
actionable System recovery (including held-session and safe old-session review), and Exit behavior.

It is not yet one literal process-to-provider E2E:

```text
current test:lived
  actual Chromium + actual V4 browser source
  + deterministic API/media fixtures

Go tests
  actual V4 service / HTTP / persistence / provider-adapter source

exact Windows lived run
  actual V4 process + actual provider runtime + human perception
```

The remaining integration seam worth closing is a **server-backed browser journey**: launch an
isolated real V4 local server/composition with deterministic provider fixtures, then drive that server
through the same Chromium journey. That would prove browser -> real V4 HTTP handlers -> persistence /
media composition without making live YouTube/network availability a test dependency.

Do not replace the deterministic lived-shell test with that future seam; keep both. The fixture-backed
journey is faster and diagnoses product behavior, while server-backed proof owns composition.

### Protected V3 compatibility

The pairing-store rollback incident established a release law:

```text
PERSISTED_STATE_UPGRADE
!=
PERMISSION_TO_BREAK_ROLLBACK
```

V4-only schemas now live under the V4 app-data root when they are not intentionally V3-readable.

Shared V3/V4 stores have one aggregate executable rollback contract:

```text
V4 owner writes shared state
-> protected V3-shaped reader consumes it
-> V3-shaped writer re-serializes it
-> semantic JSON must be lossless
```

Covered shared stores:

- config;
- firstSeen;
- overlays;
- playback windows;
- Library cache v2;
- Playlists;
- Trash;
- deleted history.

V4 also refuses writable startup while the protected V3 process claim is live and monitors that
claim while running. V3 remains available as a fallback, but V3 and V4 are not admitted as
simultaneous writers of shared Library state.

### Next proof edge

The current D4 Go changes have not yet been accepted merely because they are committed.

Next proof:

```text
exact current V4 head
-> VexRelay source-owner qualification
-> temporary task-local Go 1.23 toolchain when host Go is absent
-> gofmt
-> go test ./v2/...
-> Windows GOOS/amd64 vet/build structure
-> current Node semantic/model suite
-> canonical RETURN
```

GitHub Actions is not the proof dependency.

<!-- [VXG RealForever][VEXSTREAM][DESKTOP-V4][D4-ACTIVE][V3-ROLLBACK-COMPAT] -->
