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
