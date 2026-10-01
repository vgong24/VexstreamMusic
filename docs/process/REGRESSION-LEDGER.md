# VexStream Music regression ledger

Continuity: `[VXG RealForever]`

## VSM-REG-001 — 0.9.4 player render aborts activation

```text
class=candidate regression
introducedIn=0.9.4
observedBy=user real-use report + deterministic Chromium reproduction
status=REPAIRED_IN_0.9.5
```

`renderPlayer()` declared playback bounds as `b` inside the `if (art)` block and then used `b` afterward. The browser raised `ReferenceError: b is not defined`.

`setQueue()` renders before it invokes `playIndex()`. Therefore the exception occurred after the queue/title changed but before audio source assignment and playback. The partial UI update made the failure look like a transport-control problem rather than a JavaScript exception.

Repair: compute the playback bounds at function scope and run the actual browser interaction suite.

Evidence: `docs/evidence/0.9.4-playback-regression-reproduction.json` and `docs/evidence/ui-runtime-regression.json`.

## VSM-REG-002 — Library activation creates a one-song queue

```text
class=pre-existing latent interaction defect
presentIn=0.9.3_and_0.9.4
observedBy=differential source/runtime review
status=REPAIRED_IN_0.9.5
```

The Songs row action called `playTrack(id)`, which created `setQueue([track])` whenever the track was not already queued. Playback of the selected file could work, but Next had no second item to advance to.

Repair: Songs/Home row activation now establishes the current visible/sorted library sequence and starts at the selected index. Explicit queue/playlist operations retain their own queue semantics.

## VSM-REG-003 — Play cannot recover an unloaded current source

```text
class=interaction-hardening defect
status=REPAIRED_IN_0.9.5
```

The Play button previously called `audio.play()` directly. If UI/queue state identified a current track but the source was absent after a partial render, reset, or stale session state, Play could not reconstruct the intended source.

Repair: the player retains the loaded track identity. Play calls `playIndex(currentIndex)` when the current queue item and loaded audio source are not aligned.

## VSM-PROC-001 — Unversioned distributable filename

```text
class=packaging/process defect
status=REPAIRED_IN_0.9.5
```

A generic `VexStreamMusic.exe` filename hid which candidate was being tested and made side-by-side rollback ambiguous.

Rule: every distributable executable is named `VexStreamMusic-<semantic-version>.exe`; the package folder and ZIP carry the same version.

## VSM-INT-004 — Queue history is hidden while still semantically present

```text
class=interaction representation defect
observedIn=0.9.5 real Windows use + source inspection
status=REPAIRED_IN_1.0.0
```

The drawer rendered only `queue.slice(queueIndex + 1)`. Pressing Next therefore removed the previous node from perception even though the underlying queue still retained it and Previous could return to it.

Repair: project one persistent queue as History / Current / Up Next and keep the current node explicit.

## VSM-INT-005 — Shuffle makes visible queue order untruthful

```text
class=interaction/state-machine defect
observedIn=0.9.5 real Windows use + source inspection
status=REPAIRED_IN_1.0.0
```

When Shuffle was enabled, `next()` chose a random item from the future slice while the drawer continued showing the original order. A user could therefore see one next node while playback skipped to another.

Repair: shuffle the upcoming sequence itself so the rendered queue becomes the playback-order source of truth; `Next` always advances one visible node.

## VSM-UX-006 — Albums have no nested track terrain

```text
class=collection navigation capability gap
observedIn=0.9.5 real Windows use
status=FORMED_IN_1.0.0
```

Album cards exposed Play/Shuffle but could not be opened to inspect contained tracks or queue at album/track granularity.

Repair: add metadata-mapped album drill-down with album-scope and track-scope playback/queue actions.

## VSM-UX-007 — Queue affordance is ambiguous and silent

```text
class=interaction affordance gap
observedIn=0.9.5 real Windows use
status=REPAIRED_IN_1.0.0
```

The bare `+` did not distinguish queueing from playlist/favorite semantics and gave no immediate acknowledgment.

Repair: use a queue/list-plus glyph with explicit tooltip/aria label and transient acknowledgement (`Queued ✓`).

## VSM-PROV-002 — "This video is not available" misclassified as retryable

```text
class=provider diagnostic classification defect
observedIn=0.9.5 Windows diagnostic packet
status=REPAIRED_IN_1.0.0
```

The bridge matched `unavailable` but not the provider phrase `not available`, causing a non-retryable source-unavailable condition to fall through to `YOUTUBE_PROVIDER_ERROR` with `retryable=true`.

Repair: recognize `not available` and cover it in the bridge self-test.

## VSM-UX-008 — Permanent sidebar consumes narrow-window terrain

```text
class=responsive shell / information architecture gap
observedIn=0.9.6 real Windows use
status=REPAIRED_IN_1.0.0
```

Repair: one hamburger control collapses the wide sidebar to an icon rail and opens it as an overlay drawer on narrow windows.

## VSM-UX-009 — Desktop table clips instead of changing representation

```text
class=responsive representation defect
observedIn=0.9.6 real Windows resize test
status=REPAIRED_IN_1.0.0
```

Fixed desktop columns competed for widths below their usable threshold. Repair: configurable desktop fields plus a narrow track-card projection. The harness asserts no horizontal document overflow and preserves Title / Artist / Album / Time perception.

## VSM-UX-010 — Song metadata priority is fixed

```text
class=information-density limitation
observedIn=0.9.6 real Windows use
status=REPAIRED_IN_1.0.0
```

Repair: user-selectable Artist / Album / Added / Time / Genre fields and a persistent Title-space preference. Genre remains discoverable but no longer occupies default desktop space.

## VSM-UX-011 — Track actions lack intent hierarchy

```text
class=interaction affordance / safety layout gap
observedIn=0.9.6 real Windows use
status=REPAIRED_IN_1.0.0
```

Repair: three frequent playback intents in one quick row; descriptive icon rows for secondary actions; inline playlist selection; one top-right Close; destructive move isolated lower-left.

## VSM-UX-012 — Compact player controls become crowded

```text
class=responsive player geometry gap
observedIn=0.9.6 real Windows resize test
status=REPAIRED_IN_1.0.0
```

Repair: narrow two-row player projection. Essential transport, seek, queue, current title and art remain reachable; optional volume chrome yields space first. The Chromium harness checks control bounding boxes against the viewport.

## R-101 — Darwin build rejected Windows-only process attributes

```text
firstAffectedCandidate=1.0.0
classification=candidate portability defect
symptom=darwin arm64/amd64 compile fails on SysProcAttr HideWindow/CreationFlags fields
cause=Windows-only syscall fields defined in shared main.go
repair=move process attributes behind Go build-tagged Windows and Unix/macOS files
regressionProof=darwin/arm64 + darwin/amd64 go vet/build + Mach-O structural qualification
status=REPAIRED_IN_1.0.1
realMacHostAcceptance=PENDING
```

## VSM-MAC-013 — macOS import setup existed only as a consumer, not a bootstrap

```text
class=platform runtime-setup gap
observedIn=1.0.1 real Mac use
status=REPAIRED_IN_1.0.2
```

The Mac build could use a compatible Python/yt-dlp runtime if one already existed, but `/api/import/setup` stopped on non-Windows hosts. Repair: app-local macOS virtualenv bootstrap, Homebrew-aware dependency discovery/setup, and separate search/inspect readiness from FFmpeg-dependent MP3 import readiness.

## VSM-UX-014 — Artist browse cards are action-heavy dead ends

```text
class=collection navigation / information-architecture gap
observedIn=1.0.2 real cross-platform use
status=REPAIRED_IN_1.1.0
```

Artist cards exposed Play/Shuffle but could not be entered to inspect the artist's actual works. Repair: artwork-first square artist cards become navigation surfaces; opening one produces a stable artist detail terrain.

## VSM-UX-015 — Artist albums and songs cannot coexist as browse modes

```text
class=nested collection / scoped-search gap
observedIn=1.0.2 real cross-platform use
status=REPAIRED_IN_1.1.0
```

Users who think in albums need album grouping, while users who know the song should not be forced through album navigation. Repair: optional collapsible album shelf + artist-local mini-search + sortable artist-scoped song table. The expanded shelf is bounded to approximately three responsive rows before `Show all`; selecting an album filters the same song terrain rather than creating a second queue or library model.

## VSM-UX-016 — Album terrain cannot navigate back to credited artists

```text
class=bidirectional collection / relationship-navigation gap
observedIn=1.1.0 real cross-platform use
status=REPAIRED_IN_1.1.1
```

Album detail exposed artist text but not a traversable relationship, so `Artist -> Album -> Songs` had no reverse `Album -> Artist` edge. Repair: derive the unique stored artist identities represented by album-member tracks, render them as navigable relation chips, make each album-track artist navigable, and preserve the open album when traversing into an artist terrain. Multi-artist collections expose every distinct stored artist identity. VexStream does not split artist names on punctuation; structured collaborator identity must come from structured metadata rather than heuristics.



## VSM-UX-017 — Active Album/Artist scope escaped to global Home

```text
class=navigation-scope interpretation regression
firstAffectedCandidate=1.1.2
observedIn=1.1.2 successor feedback
status=REPAIRED_IN_1.1.3
```

1.1.2 interpreted repeated active **Albums** / **Artists** clicks as a two-step ladder from detail -> scope browse -> global Library Home. The intended contract is narrower: the scope control is the home/back control **for its own scope only**. Repair: Album detail -> Albums browse root and Artist detail -> Artists browse root; clicking the already-rooted scope again is idempotent. Global Home remains an explicit Home/wordmark action.

## VSM-IMP-018 — Whole-track duplicate evidence was hidden by chapter-only UI gate

```text
class=import duplicate-awareness presentation defect
firstAffectedCandidate=duplicate advisory introduction
observedIn=1.1.3 real YouTube import review
status=REPAIRED_IN_1.1.4
```

`duplicateHTML()` was rendered only inside the `has track sections` branch, so ordinary single-track videos could never show an existing-library warning even when duplicate evidence existed. Repair: render duplicate advisories independently of chapter/section layout and add a Chromium regression for a no-sections source.

## VSM-IMP-019 — Different YouTube source identity obscures same-song candidate

```text
class=duplicate identity / source-title normalization gap
observedIn=1.1.3 real YouTube import review
status=REPAIRED_IN_1.1.4
```

A provider title such as `Paramore: Ain't It Fun [OFFICIAL VIDEO]` and local `Ain't It Fun` may be the same song despite different provider IDs and different edit durations. Repair: compare normalized title variants with bounded creator-prefix stripping, use source creator only as corroborating evidence, preserve duration divergence as a possible-edit reason, and keep provider/channel identity distinct from canonical artist metadata.

## VSM-SRCH-020 — Apostrophe/punctuation/token variants fail library rediscovery

```text
class=retrieval normalization gap
observedIn=1.1.3 real library search
status=REPAIRED_IN_1.1.4
```

Literal lowercase substring matching made `aint` fail to rediscover `Ain't It Fun`. Repair: shared case/diacritic/punctuation normalization, token-prefix matching, and bounded compact-form matching for global and Artist-scoped search. Generic edit-distance fuzzy matching remains intentionally absent.

## VSM-IMP-021 — Audio quality comparison risks false cross-codec ranking

```text
class=duplicate quality evidence boundary
observedIn=1.1.3 duplicate-intake refinement
status=HARDENED_IN_1.1.4
```

Repair: expose source/local format indicators, permit bitrate direction only within a comparable codec family, and report different codecs as not directly comparable rather than treating bitrate alone as perceived quality.

## 1.1.5 — existing-library artist suggestion

```text
finding=PROVIDER_ARTIST_EMPTY_WHILE_EXISTING_LIBRARY_ARTIST_IS_EXPLICIT_TITLE_CREDIT
repair=HIGH_CONFIDENCE_EDITABLE_EXISTING_LIBRARY_ARTIST_SUGGESTION
negativeGuard=LOOSE_SUBSTRING_MATCH_DOES_NOT_SUGGEST_ARTIST
status=REPAIRED_IN_1.1.5
```

## VSM-INT-022 — Songs filter silently redefines playback universe

```text
class=interaction/state-scope defect
presentThrough=1.1.5
observedBy=real Windows filtered-library use
status=REPAIRED_IN_2.0.0
```

The Songs view reused `sortedSongTracks()` for both display and playback. Because `sortedSongTracks()` sorts `baseFilteredTracks()`, a four-result search could form a four-track playback timeline even when hundreds of library tracks existed.

Repair: preserve `sortedSongTracks()` as the visible Songs projection and introduce `sortedLibraryTracks()` as the full-library playback projection. Row playback, idle Songs Play, Play all, and Shuffle all use the full library. Explicit Add-to-queue and bulk queue actions remain intent-scoped.

## VSM-PROC-023 — Executable survived while canonical source continuity did not

```text
class=source-continuity/process defect
observedAfter=1.1.5
failedCandidates=1.1.6,1.1.7
status=RECOVERED_IN_2.0.0
```

A later lane received the Windows executable without the complete source lineage and attempted equal-length binary patches. 1.1.7 was rejected after real Windows use reported nonfunctional playback. The exact 1.1.5 source was subsequently recovered and shown to reproduce the historical 1.1.5 Windows executable byte-for-byte.

Rule: VexStream product evolution is source-first. A stripped executable may be diagnostic evidence, not the canonical writer surface when exact source exists.

## VSM-QUAL-024 — UI state-machine PASS did not prove real media delivery

```text
class=qualification-coverage gap
status=HARDENED_IN_2.0.0
```

The deterministic Chromium harness intentionally mocks `HTMLAudioElement` and the API. It proves browser state-machine behavior but not server media bytes. 2.0.0 adds a real source-built server regression with a real MP3 fixture, library scan, exact full response, `Accept-Ranges: bytes`, and exact `206` range response. Browser decoder/audio-device acceptance remains a separate target-host gate.
