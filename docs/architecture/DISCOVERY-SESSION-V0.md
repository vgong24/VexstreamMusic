# Historical note

**Superseded for current behavior by `EXPLORE-SESSION-V1.md` in VexStream Music 2.2.0.** This document remains the formation record for the original temporary-session lifecycle.

# Discovery Session v0

`[VXG RealForever]`

## Purpose

Discovery Session v0 lets a human explore music adjacent to a local or provider seed without committing those candidates to the durable library or the ordinary VexStream playback queue.

## Core separation

```text
Library queue
  durable-enough session playback projection of local Track IDs

Discovery session
  temporary provider candidate graph held in browser memory

Provider preview
  one externally controlled YouTube embed selected from that graph

Library promotion
  explicit handoff into existing inspect / duplicate / import workflow
```

Therefore:

```text
DISCOVERY_SESSION != LIBRARY_QUEUE
PREVIEWED != DOWNLOADED
PROVIDER_RESULT != AI_RECOMMENDATION
AI_RANKING != PROVIDER_RANKING
MORE_LIKE_THIS != COMMIT_TO_PLAYLIST
```

## Backend projection

`POST /api/discovery/radio`

Accepts either:

```json
{"trackId":"local-track-id"}
```

or a provider seed:

```json
{"seed":{"providerId":"...","url":"...","title":"...","channel":"...","duration":123}}
```

The endpoint forms at most two deterministic search queries from factual seed metadata, invokes the existing provider search bridge, deduplicates candidate metadata, removes the seed itself, caps the current frame, and annotates each candidate using existing library duplicate/provenance matching.

Response basis is explicit:

```json
{"provider":"youtube","method":"SEARCH_DERIVED","queries":["..."]}
```

No queue, playlist, media file, or discovery-session state is mutated server-side.

## Browser lifecycle

The browser holds:

```text
frames[]
frameIndex
previewIndex
resumeLocal
```

- Starting from another local track replaces the current unsaved session.
- Branching `More like this` appends a child frame and discards any abandoned forward branch.
- Back decrements the frame cursor and reuses retained metadata.
- Leaving Radio stops provider preview; leaving Discover may resume the previously playing local track.
- Metadata frames survive Library/Discover navigation in the same page.
- Clear destroys the frames.
- App/page exit destroys unsaved v0 state.

## Loading discipline

A frame holds metadata only. V0 does not create hidden players or pre-download candidate audio.

```text
candidate map -> metadata
focused preview -> one visible provider player
Add to library -> explicit existing import path
```

This keeps resource use proportional to the map rather than to every media object represented by the map.

## Library match states

Provider candidates are annotated as:

```text
IN_LIBRARY
LIKELY_MATCH
POSSIBLE_MATCH
NOT_IN_LIBRARY
```

`IN_LIBRARY` requires the existing duplicate matcher to classify the provider candidate as the same source. The other states remain advisory.

## AI seam

A later AI/VexLife actor may consume:

- seed metadata;
- candidate metadata;
- library-match state;
- current branch/frame history;
- future user-authored notes/preferences.

The AI may rank/explain candidates but does not rewrite the provider-generation basis. V0 remains useful with AI absent.

## Provider preview policy boundary

Preview uses a visible YouTube iframe. VexStream does not cover the player, suppress its controls, or attempt to suppress provider advertising. A candidate can also be unavailable for embedding; that is a provider availability condition, not a reason to silently download it for preview.

## v0 non-goals

- no durable radio history;
- no saved provider playlist object;
- no automatic downloads;
- no AI/autonomous selection;
- no background multi-player preloading;
- no provider recommendation-graph claim;
- no VexLife federation or remote sharing.
