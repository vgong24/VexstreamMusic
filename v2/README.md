# VexStream Music Desktop V2 Architecture Workspace

`[VXG RealForever]`

## The question

> Can VexStream Desktop preserve the geometry of human intention closely enough that a future
> developer or AI can find one behavioral question, understand its current truth, change it, and
> prove it without reconstructing the entire product from a monolith?

V2 is a clean architectural reconstruction beside the existing desktop source. It is **not** a
rewrite that erases V1, and it is not permission to silently change product behavior.

`v2/` names the **architecture-generation workspace**, not the eventual marketing version. The
intended user-facing successor release is **VexStream Music 4.0.0** once the refactored desktop has
earned release acceptance.

The existing VexStream Music 3.0 desktop remains the protected daily-driver baseline while this work
continues. V2/V4 work must not require replacing or mutating that installed/running baseline before
the new Windows candidate is independently usable.

The existing repository root remains the V1 implementation and evidence quarry. V2 grows under
`v2/` and migrates one behavior at a time.

## Source ancestry

This V2 formation starts from the exact VexStream 3.0.0 Windows-green checkpoint:

```text
c8732f35f4f173952dd6df24ee7931557b9de274
Windows CI run #63 = SUCCESS
version = 3.0.0
startup reuse identity = version + buildRef
explicit process exit = present
```

Later remote experiments remain useful quarry, not inherited V2 truth:

```text
PR #3  content-blind off-home relay candidate
PR #4  direct paired-device ingress candidate
```

See `quarry/V1-INTENTION-MAP.md`.

## Entry route

A fresh V2 worker should read only as far as the current question requires:

1. this file;
2. `architecture/README.md`;
3. `quarry/V1-INTENTION-MAP.md` when translating existing V1 behavior;
4. the exact feature family being changed;
5. the proof owned by that family.

Do not reread every V1 file to make an unrelated V2 change.

## V2 laws

### One family answers one human behavioral question

Examples:

```text
How is playback behaving?
How is the queue behaving?
What media is available here?
What is downloading?
What remote route is currently usable?
```

A package/family is earned when the question itself becomes independently meaningful, not because a
file became large.

### State / Control / Act / Effect

V2 adopts the semantic grammar pressure-tested by VexLife-Android PR #82:

```text
STATE
    what is actually true now?

CONTROL
    what stable affordance did the human interact with?

ACT
    what does that Control mean from current State?

EFFECT
    what runtime transformation should be attempted?
```

Causal reading:

```text
observed State
-> Control resolves Act
-> Act selects Effect
-> Effect is admitted
-> provider/runtime observation
-> next truthful State
```

Permanent distinction:

```text
EFFECT_ATTEMPTED != PROVIDER_SUCCEEDED
```

### Provider truth stays with the provider boundary

HTMLMediaElement, filesystem, HTTP, provider bridges, remote transports, and host process state are
not replaced by architecture objects. A Connection/adapter translates those facts into semantic
State and realizes Effects.

### One truth owner, many projections

UI, diagnostics, journeys, crawler evidence, mobile projections, and tests consume existing truth.
They do not become parallel state owners.

### Proof follows the claim

Test the transformation at its cheapest truthful owner. Browser interaction, real media delivery,
Windows process lifecycle, remote reachability, and lived audio output are distinct proof surfaces.

### V1 is quarry, not a template to copy

When migrating a V1 behavior:

```text
locate human intention
-> locate existing source behavior
-> identify current truth owner
-> preserve known acceptance evidence
-> form the smallest V2 family
-> prove semantic parity
-> only then retire/replace the V1 path
```

A successful migration does not require preserving accidental V1 coupling.

## First real family

Playback is the first V2 semantic family because it already has:

- a real V1 browser provider (`HTMLAudioElement`);
- a visible human Play/Pause control;
- V1 behavior worth preserving;
- an Android architecture exemplar;
- enough semantic pressure to expose coupling with Queue without pretending Queue is Playback.

Current seed source:

```text
web/architecture/behavior.js
web/features/playback/behavior.js
web/features/playback/presentation.js
web/features/playback/connection.js
web/foundation/text.js
web/locales/en.js
tests/playback-behavior.test.mjs
tests/playback-connection.test.mjs
```

The seed deliberately does **not** yet replace V1 playback. It now includes a concrete
HTMLMediaElement-like Connection that:

- translates provider facts into Playback State;
- re-resolves stable Control meaning at activation time;
- keeps admitted intent distinct from provider acknowledgement;
- delegates restart-position ownership to the concrete media owner;
- refuses to make Playback silently own Queue selection.

Current local semantic/provider proof:

```text
Node 22.16.0
13 tests
13 PASS
0 FAIL
```

This is not yet browser or Windows host proof.

## Desktop-first completion posture

Android/mobile integration is deliberately held while the desktop reconstruction is incomplete.

```text
VexStream Music 3.0
    = protected current daily driver

v2/ architecture workspace
    -> complete local desktop product
    -> browser + media + persistence + Windows proof
    -> VexStream Music 4.0.0 release candidate
    -> lived desktop acceptance
    -> only then resume Android/mobile integration
```

No Android source change is required to prove a Desktop V4 milestone. The Android companion PR may
preserve planning/currentness, but implementation waits for the Desktop V4 readiness gate.

## What comes next

The next implementation edge is:

```text
real V2 browser surface
-> real <audio> element
-> V2 Playback Connection
-> V2 presentation/TextRef
-> browser interaction proof
-> compare against preserved V1 playback intention
```

Only after that slice is truthful should Queue, media availability/download, or whole-screen migration
be pulled into V2.

<!-- [VXG RealForever][VEXSTREAM][DESKTOP-V2][FOUNDATION] -->
