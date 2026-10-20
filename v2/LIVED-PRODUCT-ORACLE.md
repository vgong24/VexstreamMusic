# VexStream V4 — Lived Product Oracle

`[VXG RealForever]`

## Why this source exists

The first exact Windows-qualified V4 candidate proved that the refactored implementation could build,
run, serve media, preserve rollback boundaries, and reject protected V3 concurrency.

It then failed the first lived Desktop Readiness check for a more fundamental reason:

```text
A029 native Windows qualification = PASS

A030 lived Desktop Readiness = FAIL
human note = "ui changed completely from original 3.0.0"
```

That failure corrects the interpretation of the V4 program.

## Product oracle

VexStream Music 3.0 at the protected base remains the lived-product oracle while V4 is formed:

```text
protected V3 head
= 8ffc302b3f03281ada06acab134cef5a11ac85b8

protected V3 product shell
= ui/index.html
```

The refactor target is:

```text
V3 recognizable human product
        ↓
same screens / navigation / workflows / affordances unless deliberately changed
        ↓
V4 question-owned semantic and provider owners underneath
```

The current V4 `v2/web/index.html` practicum is a proof scaffold. It is not product-shell authority.

## What "refactor" means here

The purpose of V4 is to make the implementation easier to enter by human question, in the same
spirit as the accepted VexLife Android architecture.

A developer or AI should be able to ask:

```text
How does Playback work?
How does Queue work?
Who owns Library truth?
Who realizes source-folder changes?
What owns Explore session truth?
What owns Import staging and commit?
```

and enter one bounded neighborhood rather than reconstruct the old monolith.

Conceptually:

```text
V3-looking control
    -> feature Experience / UI membrane
    -> Control + CURRENT State
    -> Act
    -> Effect
    -> Connection / concrete provider
    -> observed CURRENT State
    -> Presentation
    -> same recognizable product surface
```

Better structure does NOT itself authorize a new product design.

## What must remain recognizable by default

Unless Victor explicitly admits a product/UX change, preserve the lived V3 geometry including the
intent of:

- Library shell and navigation;
- Home / Songs / Albums / Artists;
- Recently Discovered;
- album and artist detail terrain;
- contextual and bulk track actions;
- Playlists and playlist detail;
- Deleted Music / Trash;
- Discover with PC / YouTube / Explore paths;
- fixed Now Playing surface;
- Queue drawer/timeline with history/current/up-next meaning;
- System / diagnostics / repair surfaces;
- mobile/pairing surfaces when their downstream gate is resumed.

Preservation is semantic and interaction-level, not a demand to keep one 200 KB HTML file forever.

## Player, Queue, Home, and System lived contract

Recent V4 lived review exposed a specific class of regressions that source-level Player/Queue
correctness alone cannot catch. The product must preserve the human's ability to answer four
ordinary questions without guessing:

```text
What is playing right now?
What will play next, and what exactly is the current Queue item?
What can I do with this song from the place where I found it?
What does this System attention signal mean, and what safe action resolves it?
```

Unless Victor explicitly admits a different UX, preserve these V3-visible outcomes:

- the fixed player gives current-track title and artist/collection enough horizontal room to remain
  recognizable; Volume must not consume that identity budget merely because it is a horizontal
  slider;
- overflowing title text moves at a readable **distance-derived velocity**. A longer title receives
  proportionally more time; it must not accelerate merely to finish inside one fixed duration;
- the expanded Queue keeps the current item human- and accessibility-identifiable through title,
  artist, album/collection, available description/source detail, and current/played/up-next state;
- Home song rows expose the same normal track-action vocabulary as Songs: Play, Add to queue, and the
  contextual actions such as Play next, Explore, Edit, Deleted music, and Playlist;
- a held listening-session badge explains what is held and exposes a bounded reset/adopt-current
  action instead of presenting an unexplained count;
- System Status can review parallel VexStream server sessions. Retirement remains explicit and
  fail-safe: the current process is never eligible; sessions with active imports, unverified state,
  or live track IDs not covered by the current Library remain held; retiring an eligible old server
  never deletes Library folders or media.

The proof ownership for this slice is intentionally layered:

```text
web/foundation/marquee.js
  overflow distance -> readable duration
    -> tests/marquee-foundation.test.mjs

desktop/lifecycle/sessions*.go
  Windows process discovery + old-session safety classification/retirement
    -> desktop/lifecycle/sessions_test.go

web/features/system/client.js
  explicit review/retire browser contract
    -> tests/system-client.test.mjs

actual V4 HTML/JS + deterministic API/media fixture
  current-player geometry
  Home action parity
  Queue identity
  held-session recovery
  old-session review/retire interaction
    -> tests/lived-shell-browser-smoke.mjs

exact Windows lived run
  real process discovery/retirement boundary + actual product feel
```

A green Queue reducer test does not prove the listener can read the current track. Likewise, a visible
System badge is not complete merely because its underlying state is technically accurate.

## What V4 work remains valid

Do not throw away the V4 extraction merely because the practicum shell failed lived acceptance.

Current V4 source has useful bounded owners for, among other things:

- Playback and Queue semantics;
- Library truth, filesystem discovery/cache, overlays, and playback windows;
- source configuration and discovery;
- Playlists;
- Trash / deleted history / purge recovery;
- Explore;
- Import inspect / review / staging / jobs / commit recovery;
- player-session persistence;
- app-data placement and V3 rollback compatibility;
- process/build identity and explicit Exit;
- YouTube/provider runtime boundaries.

The recovery direction is therefore NOT:

```text
discard V4
-> restore monolith
```

It is:

```text
preserve V3 lived product
-> keep/extract the V4 owners that answer real questions
-> reconnect the V3 product shell to those owners
-> move old monolithic browser/backend logic behind those boundaries slice by slice
```

## YouTube Import lived contract

The YouTube path is one cause-and-effect journey, not a bag of individually green endpoints.

Unless Victor explicitly admits a UX change, preserve:

```text
paste/link or Explore candidate
-> Review gives immediate in-app "Inspecting source…" feedback
   -> provider subprocess remains background implementation
   -> no acquisition has begun
-> Source review keeps the chosen media recognizable
   -> artwork / title / creator / duration
   -> destination / track layout / metadata edits
-> current Library is checked before acquisition
   -> no match: say so
   -> same/likely/possible match: show the actual existing track + reasons
   -> SAME_SOURCE: make "Already in your Library" explicit
   -> Preview 12s reads the existing local media only
   -> duplicate evidence remains advisory for deliberate alternate edits/copies
-> Download & add is the acquisition boundary
-> active Import keeps the same source identity visible
   -> provider stage + real bounded download progress
   -> cancellation while active
-> preparation / Library commit remains recoverable
-> completion visibly becomes Added to Library
-> committed track is reachable through View in Library
```

The proof map intentionally follows the same claim layering used by the Android VexStream feature:

```text
desktop/imports/duplicates/review_test.go
    duplicate scoring / SAME_SOURCE / different-edit / quality meaning

desktop/imports/review/service_test.go
    provider inspection + current Library composition

web/features/imports review/client tests
    reviewed evidence + typed acquisition request

desktop/provider/youtube process tests
    provider subprocess platform behavior

tests/lived-shell-browser-smoke.mjs
    integrated human browser journey over deterministic API/media fixtures

tests/player-browser-regression.py
    broader browser interaction regression

exact Windows build + Victor lived review
    native host/process behavior and product feel
```

A lower layer going green never authorizes retiring a higher-layer product behavior.

## Current acceptance state

Exact Windows qualification remains valid evidence for the candidate that was actually tested:

```text
head = 59864b36676a11cee0460afa8d7065bb3fc69c17
tree = 84aeb925e2d1cf4fc53945f9ee66ddceb030bee2

qualified Windows executable SHA-256
= acb47ce25d6b4beb87f0fe2bc18a23c4dd95aa774ea7a0539732ec3df7e14293
```

That proves the implementation was executable and internally coherent on Windows. It does NOT prove
lived product parity.

Desktop Readiness is currently:

```text
FAIL — product-shell / lived-experience parity incomplete
```

PR #5 remains draft. Android/mobile implementation remains held.

## Next implementation edge

Do not reopen Windows qualification yet.

The next smallest product-bearing seam is:

```text
V3 product shell
-> real V4 Library projection
-> real V4 Playback + Queue owners
-> real /media/<ref> bytes
-> real HTMLAudioElement
```

Fast perception should happen on Mac/browser first:

```text
launch
-> looks recognizably like V3
-> sample/real admitted tracks render
-> Play produces real audio
-> Next / Previous operate
-> fixed player and Queue surface remain coherent
```

Only after the lived shell is recognizably recovered should broader feature parity and another exact
Windows qualification be spent.

## Fresh-entry discipline

A fresh executor must not treat this document or a successor bundle as proof that mutable repository
state is unchanged.

Before source mutation:

```text
read the current successor/oracle
-> re-ground PR #5 head/base/draft state
-> if head moved, inspect the delta from the recorded coordinate
-> resume only from the first still-unconsumed compatible edge
```

Do not restart the program from old attempts merely because the conversation is fresh.

## Role of V3 source during recovery

The protected V3 source is the lived-product and behavior quarry. It is not the backend implementation
to revive as the new architecture.

In particular:

```text
V3 ui/index.html
    = product/presentation oracle

V3 root main.go and historical provider/network dependencies
    = quarry for behavior and compatibility
    != target runtime topology
```

Do not spend the recovery edge modernizing the old V3 backend just to make it run on a newer host.
Reconnect the recognizable V3 shell/geometry to the current V4 owners and providers.

A mock or UI-only preview may be used to understand the oracle, but it cannot satisfy the next edge.
The next accepted checkpoint must use real V4 Library/media/Playback/Queue logic and a real
HTMLAudioElement/media request.

## Migration discipline

For each visible V3 behavior:

```text
V3 screen / human question
-> V3 control and lived transition
-> current V4 semantic owner, if one exists
-> missing Experience / adapter / presentation seam
-> reconnect without changing product geometry
-> browser/lived parity proof
```

If a V3 behavior has no current V4 owner, mark it missing. Do not silently retire it because the
practicum did not expose it.

If a UX change would be beneficial, propose it separately. Do not smuggle it through as "refactor."

<!-- [VXG RealForever][VEXSTREAM][V4][LIVED-PRODUCT-ORACLE] -->
