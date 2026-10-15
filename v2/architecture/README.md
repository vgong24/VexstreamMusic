# VexStream Desktop V2 — Architecture

`[VXG RealForever]`

## Why this exists

V1 proved a great deal of product behavior quickly, but its implementation now concentrates many
independent questions in a small number of very large files. V2 preserves the achieved product while
making semantic ownership visible in source.

This architecture uses VexLife-Android PR #82 as an **intention blueprint**, not as a Kotlin framework
dependency.

The desktop adaptation keeps the useful questions and chooses browser/Go-native mechanisms.

## 1. Behavior family

A Behavior family owns one exact semantic question.

Canonical shape, when the responsibilities are real:

```text
<feature>/
    docs / navigation root
    state
    control
    act
    effect
    behavior
    connection        # only when provider/runtime realization exists
    presentation      # only when UI meaning exists
    pending/currentness # only when provider latency requires it
```

The exact file count is not sacred. The semantic roles are.

### Split law

Create a sibling family when several of these are true:

1. it has independently meaningful current truth;
2. its Controls/Acts/Effects make sense independently;
3. it can evolve or be observed independently;
4. it can be projected/converged independently;
5. provider/currentness/lifetime mechanics differ materially;
6. a developer naturally asks about it as a separate behavioral question.

Do not split just because a file is long.

## 2. Desktop provider boundary

V2 should keep product meaning portable and provider mechanics concrete.

Examples:

```text
Playback semantics
    -> browser Playback Connection
    -> HTMLAudioElement

Library semantics
    -> desktop library service
    -> filesystem/catalog persistence

Remote media semantics
    -> bounded feature service
    -> private route/transport adapter
```

A Connection may:

- observe provider truth;
- translate provider truth into semantic State;
- realize admitted Effects;
- protect stale async completion when the provider requires it.

A Connection must not invent provider success.

## 3. Queue is not Playback

V1 currently lets `togglePlayback()` seed a queue when there is no current track. That is useful
product behavior, but it crosses two questions.

V2 should make the composition explicit:

```text
Playback
    current media playback truth

Queue
    ordered media context + current position

higher composition
    Play pressed with no current media
    -> choose/seed Queue according to current product context
    -> Playback receives a selected media item
```

This is a concrete example of why V2 exists: preserve the human behavior while exposing the real
ownership boundary.

## 4. Relay / convergence pattern

Desktop V2 should adopt the *shape* of the Android Relay kernel only when several current truths need
one derived projection.

```text
dependency A ----\
dependency B -----+-> one serialized reconcile(...) -> current projection
dependency C ----/
```

A Relay-like owner should be boring to inspect:

1. dependencies — what changes around it;
2. public intentions — what callers may ask;
3. reconcile/process — the law;
4. output — what consumers may depend on.

It does not automatically own:

- scheduling;
- service hosting;
- network transport;
- persistence;
- provider lifecycle;
- UI rendering.

Do not create a generic event bus. A convergence owner is earned by a real multi-truth consumer.

## 5. Localization and design language

Human-visible authored text must be addressable by stable text identity rather than being inseparable
from rendering code.

V2 begins with:

```text
TextRef
-> locale catalog
-> rendered string
```

The first catalog is English. Adding a crawler locale does not declare product language support.

Permanent distinction:

```text
CRAWLER_LOCALE != PRODUCT_SUPPORTED_LANGUAGE
```

When V2 admits supported languages, one product-owned language registry should define them. Browser
evidence consumes that registry rather than creating another list.

Feature presentation should expose TextRefs where practical so semantic tests can verify meaning
without hard-coding translated prose.

## 6. Stable screen/element identity

V2 browser UI should gradually give meaningful screens and controls stable semantic identity.

The purpose is not instrumentation for its own sake. Stable identity lets the same source map serve:

- UI wiring;
- accessibility intent;
- human Journeys;
- browser automation;
- ScreenCrawler selection;
- diagnostics.

Do not build a second registry solely for tests.

## 7. Journey before crawler

A Journey owns an accepted human-readable semantic walk.

Conceptually:

```text
Journey
    semantic human steps
        |
        +-> browser functional execution
        +-> coverage/map checks
        +-> ScreenCrawler evidence
```

ScreenCrawler is an opt-in evidence runner over Journeys.

It may multiply selected Journeys across:

- locale;
- viewport;
- font scale;
- color/theme mode;
- browser/host conditions when required.

It does not own navigation or feature truth.

V2 does not implement the crawler framework yet because no V2 Screen/Journey has been migrated.
The first migrated screen should earn the smallest browser-native version. The current V1 Chromium
harness remains the initial implementation quarry.

## 8. Test / proof architecture

Governing rule:

```text
TEST THE TRANSFORMATION AT ITS OWNER
```

Desktop topology:

```text
architecture contract
    reusable semantic laws
        |
feature/domain test
    input -> semantic output
        |
presentation test
    semantic output -> UI-facing meaning/TextRef
        |
browser render/interaction
    real DOM + real browser events
        |
Journey
    human path through real composition
        |
ScreenCrawler / visual evidence
    selected environment capture
        |
host/lived proof
    filesystem / process / network / actual audio-provider reality
```

Coverage is diagnostic evidence, not correctness.

Existing V1 proof remains valuable quarry:

- Go unit/regression tests;
- static source checks;
- Chromium interaction checks;
- real media range delivery;
- Windows PE build qualification;
- lived host/device observations.

V2 should migrate proof with the behavior it protects instead of rebuilding every proof layer at once.

## 9. Foundation promotion rule

Do not create a generic foundation merely because one feature has a useful helper.

Promote when:

- a second real consumer needs the same semantic law;
- keeping it duplicated would create drift;
- the owner and proof boundary are clear.

This applies especially to Behavior helpers, Journey tooling, crawler configuration, Relay/convergence,
and localization utilities.

## 10. Orientation law

A future worker should be able to ask:

> How does Playback work?

and enter one neighborhood rather than search the whole repository.

A healthy family answers:

1. what can truthfully be true?
2. what can the human interact with?
3. what does that Control mean now?
4. what Effect is requested?
5. who realizes it?
6. who owns provider truth?
7. what protects stale activation/completion?
8. what proof establishes the claim?

If those answers require reconstructing a hidden switchboard, the migration is not finished.

<!-- [VXG RealForever][VEXSTREAM][DESKTOP-V2][ARCHITECTURE] -->
