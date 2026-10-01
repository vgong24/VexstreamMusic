# VexStream Creation + Publishing Frontier

[VXG RealForever]

**Status:** architecture checkpoint only; no provider account, upload, generation, network, source publication or protected effect is performed by this document.

## Product intent

Extend VexStream from:

```text
LISTEN + DISCOVER
```

into an optional creator path:

```text
CREATE → REFINE → RENDER → PRESERVE → PUBLISH/SHARE → STREAM
```

without making AI, YouTube, VexLife, or any one provider the canonical owner of a user's music.

## Core object model

```text
CreationProject
  identity for the user's editable work

GenerationSource
  optional AI/provider/local-tool generation receipt and provenance

SourceAsset / Stem
  bounded source material used by the project

Render
  immutable output bytes + media digest + technical metadata

LibraryRelease
  VexStream-native accepted local music object / album / collection projection

PublicationProjection
  provider-specific published representation (for example a video/music host)

ShareProjection
  VexLife/VexStream-federated catalog/stream/copy capability projection
```

Permanent non-collapse:

```text
AI_GENERATION != CANONICAL_PROJECT
RENDER != ACCEPTED_LIBRARY_RELEASE
YOUTUBE_UPLOAD != CANONICAL_MASTER
PROVIDER_URL != MEDIA_IDENTITY
STREAM_PERMISSION != COPY_PERMISSION
FRIENDSHIP != MUSIC_PERMISSION
REMOTE_SOURCE != LOCAL_LIBRARY_ACCEPTANCE
```

## Provider-neutral creation seam

VexStream should not become "a Suno clone with one hard-coded model." A creation adapter may expose capabilities such as:

```text
TEXT_TO_MUSIC
CONTINUE_OR_EXTEND
STEM_GENERATION
REMIX_OR_VARIATION
VOCAL_OR_INSTRUMENTAL_RENDER
```

Each adapter returns bounded provenance sufficient to explain which provider/model/tool produced which source or render. The canonical VexStream project remains provider-independent and must survive provider loss.

## Preservation and Content Forge relationship

The existing Content Forge lesson maps naturally to music:

```text
produced/captured material
→ partition + provenance
→ WIP intake
→ destination-native projection
→ explicit promotion
→ generated/index verification
```

For VexStream this becomes:

```text
generated or imported audio
→ staged media bundle
→ digest/provenance/rights metadata
→ user review
→ VexStream-native track/album projection
→ library acceptance
```

A provider result must never silently become canonical library state merely because generation or download succeeded.

## YouTube / external publication

A publication adapter should be an explicit user-authorized effect:

```text
canonical VexStream render
+ title/description/artwork/visibility chosen by user
+ current provider authorization
→ upload request
→ provider receipt / external object ref
→ PublicationProjection attached to local release
```

The local render remains canonical even if the provider later removes, alters, geoblocks, or loses the external projection.

Import and publish are inverse-looking operations but have different authority:

```text
IMPORT: external source → staged local intake
PUBLISH: canonical local render → external projection
```

They must not share one blanket credential or permission switch.

## VexLife Connect relationship

VexLife connection should remain infrastructure/identity/session ownership. VexStream consumes an authorized connection and supplies music semantics:

```text
VexLife / Home / CDR
  device identity
  relationship identity
  authentication + authorization
  route/session
        ↓
VexStream
  library/catalog
  creation projects
  canonical renders
  stream/copy/share capability semantics
```

AI is optional on both sides. Ordinary browse, playback, file streaming, catalog comparison and transfer should not require a model to be awake.

## Federation consequence

A creator can therefore have one release with several projections:

```text
Canonical VexStream Release
├── local Windows media
├── local Mac media
├── VexLife-authorized direct stream
├── friend-visible catalog projection
├── explicitly copyable source projection
└── YouTube/publication projection
```

This avoids treating YouTube as the database and avoids treating VexLife as a media blob store.

## Future proof ladder

```text
C0 deterministic project/render schemas + synthetic fixtures
C1 local user-created audio import/render → VexStream library
C2 one provider generation adapter with fake/sandboxed result contract
C3 real provider generation under explicit user authorization
C4 publication adapter with dry-run/request formation and no upload
C5 one real user-approved publication + receipt reconciliation
C6 VexLife catalog/stream sharing for canonical created release
C7 multi-device/friend federation with distinct catalog/stream/copy grants
```

Each stage keeps provider effects, rights/licensing, identity, and network authority explicit rather than implied by source code existing.
