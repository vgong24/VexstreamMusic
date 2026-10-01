# VexStream Music 2.2.0 release notes

## 🌐 Explore — multi-axis candidate diversity

2.2.0 refines the temporary provider exploration map after 2.1.0 target use showed excessive repetition from the same artist/channel.

### Changed

- Track action renamed to **🌐 Explore from this song**.
- Discover sub-surface renamed from **Radio** to **🌐 Explore**.
- Candidate branch action renamed to **🌐 Explore from here**.
- Candidate generation now uses multiple labeled factual directions instead of two seed-heavy searches.
- Candidate pools are interleaved round-robin.
- A single creator/channel is capped at two candidates per frame.
- Candidate cards expose their exploration axis: **Closer**, **Neighborhood**, **Same era**, or **Versions**.
- Seed year is carried into exploration when local metadata provides it.

### Preserved

- Explore does not mutate the normal library queue.
- Preview does not download.
- Add to library uses the existing inspect / duplicate / import flow.
- Back reuses prior metadata frames.
- Unsaved exploration remains memory-only.
- YouTube preview remains one visible embedded player; VexStream does not suppress provider ads or controls.
- 2.0.x playback and Windows import-runtime repairs remain intact.

### Provider truth boundary

Candidate generation is explicitly `SEARCH_DERIVED_MULTI_AXIS`. The result map is diversified by VexStream's deterministic projection; it is not represented as YouTube's consumer recommendation algorithm.
