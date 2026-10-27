# Explore Session v1 — multi-axis search-derived terrain

`[VXG RealForever]`

## Purpose

Explore Session v1 refines Discovery Session v0 after target-host use showed that two seed-heavy searches frequently returned many songs from the same artist/channel.

The correction is architectural rather than random:

```text
ONE SEED
  -> MULTIPLE FACTUAL SEARCH AXES
  -> PER-AXIS CANDIDATE POOLS
  -> ROUND-ROBIN PROJECTION
  -> CREATOR CAP
  -> TEMPORARY EXPLORE MAP
```

The provider generation basis remains explicit search derivation. VexStream does not claim that YouTube supplied a semantic recommendation graph.

## Human language

The track action is now:

```text
🌐 Explore from this song
```

The temporary provider surface is labeled **Explore**, not Radio. The top-level **Discover** area remains the umbrella for bringing/finding music; **Explore** means branching outward from an existing musical seed.

## Axes

When seed metadata supports them, VexStream can form:

```text
CLOSER
  depth around artist / album / seed context

NEIGHBORHOOD
  broader genre neighborhood

ERA
  genre/era neighborhood when year metadata exists

VERSIONS
  alternate cover / remix / live interpretations of the title
```

Axes are factual query recipes, not semantic judgments.

## Diversity projection

Each provider query is fetched into its own pool. Candidates are then projected with:

```text
selection = ROUND_ROBIN
creatorCap = 2 per frame
frameLimit = 16
```

This prevents the first successful artist-heavy search from filling the whole map before other dimensions are represented.

Provider identity / URL deduplication remains global across the final projection.

## Candidate annotation

Each candidate carries:

```text
axis
axisLabel
query
libraryMatch
```

The UI exposes the axis label so the human/AI can distinguish why a result is present.

## Preserved lifecycle

```text
EXPLORE_SESSION != LIBRARY_QUEUE
PREVIEWED != DOWNLOADED
PROVIDER_RESULT != AI_RECOMMENDATION
AI_RANKING != PROVIDER_RANKING
EXPLORE_FROM_HERE != COMMIT_TO_PLAYLIST
```

Frames remain in browser memory. Back reuses prior metadata; Clear destroys the session; app/page exit clears unsaved state.

Preview continues to use one visible YouTube iframe at a time and never preloads every represented media object.

## AI seam

An AI actor can consume the axis-labeled candidate map and reason separately about:

- provider generation basis;
- local library match;
- branch history;
- human preference / conversational intent.

AI can re-rank or explain the map, but provider-derived axis identity remains factual provenance rather than being overwritten by an AI score.
