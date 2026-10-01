# VexStream Music 2.1.0 release notes

## Discovery Session v0

2.1.0 adds an intentionally temporary discovery/radio surface without turning provider results into library tracks or queue entries.

### New

- `⋯ → Discover more like this` from a local track.
- Metadata-only, search-derived provider candidate map.
- One visible YouTube embedded preview player at a time.
- Candidate library-match badges using existing duplicate/source identity logic.
- **Play local** for exact local source matches.
- **Add to library…** handoff to the established inspect/duplicate/import flow.
- Branch **More like this** from any provider candidate.
- Back navigation across discovery frames.
- Clear lifecycle and in-memory session persistence across Library/Discover tab navigation.
- Start radio from the current local track.

### Explicit boundaries

- Discovery does not alter the normal VexStream queue unless the human later chooses a local playback action.
- Preview does not download or promote a candidate into the library.
- Candidate generation is `SEARCH_DERIVED`, not represented as YouTube algorithmic recommendation.
- Provider preview uses the visible YouTube player; VexStream does not suppress provider ads or controls.
- Discovery frames are not persisted across app/page exit in v0.
- No AI ranking is required or performed by v0.

### Preserved repairs

2.0.0 filtered-Songs playback semantics and 2.0.1 Windows import-runtime repair remain unchanged.
