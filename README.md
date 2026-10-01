# VexStream Music 2.1.0 — Discovery Session v0

`[VXG RealForever]`

VexStream Music 2.1.0 is the first source-native discovery/radio capability on top of the accepted 2.0.x playback and Windows import-runtime repairs.

## Discovery is a temporary map, not the library queue

From a local track's `⋯` menu choose **Discover more like this**. VexStream forms a temporary, metadata-first candidate map using the existing YouTube search runtime.

```text
DISCOVERY_SESSION != LIBRARY_QUEUE
PREVIEWED != DOWNLOADED
PROVIDER_RESULT != AI_RECOMMENDATION
MORE_LIKE_THIS != COMMIT_TO_PLAYLIST
```

The initial candidate basis is deliberately labeled `SEARCH_DERIVED`. VexStream is not claiming access to YouTube's consumer recommendation graph.

### Candidate actions

- **Preview** — opens the candidate in one visible YouTube embedded player. It does not add the item to the VexStream queue or library.
- **Play local** — shown when the provider candidate is the same source as a local track.
- **Add to library…** — hands the provider URL into the existing inspect → duplicate review → import flow.
- **More like this** — forms a child discovery frame from that candidate.
- **Back** — restores the prior frame from memory without rebuilding the old candidate map.
- **Clear discovery** — ends the temporary session.

Leaving Discover stops provider preview playback but preserves the in-memory candidate frames. Returning to Discover resumes the map. Unsaved discovery state naturally disappears when the VexStream page/app process ends.

## Provider preview boundary

The preview surface is one visible YouTube iframe at a time. VexStream does not cover the player, suppress provider controls, or try to suppress provider advertising. Candidate metadata is cheap to hold; media is only requested for the candidate the human elects to preview.

## Library awareness

Candidate metadata is compared with the existing VexStream duplicate/provenance logic. Each candidate projects one of:

```text
IN_LIBRARY
LIKELY_MATCH
POSSIBLE_MATCH
NOT_IN_LIBRARY
```

An exact source match can play the local copy. Non-local candidates remain external until the human explicitly enters the existing Add-to-library workflow.

## AI seam

Discovery v0 is deterministic and usable without AI. A later VexLife/AI actor can read the seed, candidate metadata, library-match state, and branch history to rank or explain the map. AI ranking must remain distinct from provider generation.

## Preserved 2.0.x behavior

- Songs search/genre filtering remains a display lens; Songs playback uses the full sorted library.
- Explicit row/bulk queue actions remain scoped to human intent.
- Windows YouTube setup keeps search readiness separate from FFmpeg/import readiness and discovers nested WinGet FFmpeg installations.
- Real `/media/<track>` full/range byte regression remains part of source qualification.

## Qualification

Run:

```bash
python3 tests/run_qualification.py --root . --browser-executable /path/to/chromium
```

Current source qualification includes static/source checks, deterministic Chromium interaction checks, real source-built media delivery, Go regressions (including Discovery Session helpers and Windows import runtime), and the Python bridge.

Build Windows:

```bash
python3 tests/build_windows.py --root . --output ../VexStreamMusic-2.1.0.exe
```

See `docs/architecture/DISCOVERY-SESSION-V0.md` and `RELEASE-MANIFEST.json` for the explicit lifecycle and proof boundary.
