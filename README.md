# VexStream Music 3.0.0 — VexLife mobile bridge + 🌐 Explore

`[VXG RealForever]`

VexStream Music 3.0.0 promotes the VexLife phone connection to a first-class local product capability while preserving the source-native 2.2.0 playback/import/Explore baseline.

## 🌐 Explore from this song

The human-facing language is now **Explore** rather than a second use of “Discover” or a queue-shaped “Radio”.

From a local track's `⋯` menu choose:

```text
🌐 Explore from this song
```

VexStream forms a temporary metadata map without changing the ordinary library queue.

## Multi-axis candidate map

2.1.0 fetched two seed-heavy searches in sequence. The first successful artist-centric query could fill most of the frame before another direction was represented.

The preserved 2.2.0 Explore base forms factual search axes and balances them:

```text
Closer        artist / album depth
Neighborhood  genre breadth
Same era      year + genre breadth when available
Versions      cover / remix / live interpretations
```

Each axis gets its own candidate pool. VexStream projects the final frame round-robin and caps one creator/channel at two candidates per frame.

```text
provider=youtube
method=SEARCH_DERIVED_MULTI_AXIS
selection=ROUND_ROBIN
creatorCap=2
```

This is still search-derived exploration, not a claim that YouTube exposed its consumer recommendation graph.

## Candidate actions

- **Preview** — one visible YouTube embedded player; does not enter the VexStream queue.
- **Play local** — used when the candidate is the same known source as a local track.
- **Add to library…** — existing inspect → duplicate review → import flow.
- **🌐 Explore from here** — branches a child exploration frame.
- **Back** — reuses the prior metadata frame without another provider call.
- **Clear discovery** — destroys the temporary session.

## Boundaries

```text
EXPLORE_SESSION != LIBRARY_QUEUE
PREVIEWED != DOWNLOADED
PROVIDER_RESULT != AI_RECOMMENDATION
AI_RANKING != PROVIDER_RANKING
```

The map is metadata-first; media is requested only for the candidate the human elects to preview.

## Preserved behavior

- Songs filtering remains a display lens; playback uses the full sorted library.
- Queue actions remain explicitly intent-scoped.
- Windows import setup keeps search readiness separate from FFmpeg/import readiness.
- Real `/media/<track>` full/range-byte regression remains in qualification.
- Explore remains in-memory and does not persist across app/page exit.

## Qualification

```bash
python3 tests/run_qualification.py --root . --browser-executable /path/to/chromium
```

Current source proof includes static/source checks, Chromium interaction checks, real source-built media delivery, Go exploration/duplicate/import regressions, and Python bridge checks.

Build Windows:

```bash
python3 tests/build_windows.py --root . --output ../VexStreamMusic-3.0.0.exe
```

See `docs/architecture/EXPLORE-SESSION-V1.md` and `RELEASE-MANIFEST.json`.
## VexLife mobile connection

VexStream 3.0.0 adds a dedicated mobile surface for taking the canonical desktop library onto a
VexLife Android phone.

The ordinary desktop UI/admin server remains loopback-only. Phone access uses a separate bounded
HTTPS surface with only pairing, sanitized catalog, and range-capable media.

Human path:

```text
Windows VexStream
-> click the phone icon
-> VexStream becomes discoverable on the same Wi-Fi for five minutes
-> six-digit confirmation code

VexLife Android
-> Home -> VexStream Music -> Find VexStream
-> select the computer
-> compare the same six-digit code
-> Request access

Windows
-> Allow

VexLife
-> stream on the phone
-> optionally Download for verified offline playback
```

The remembered relationship uses the desktop certificate identity rather than a temporary LAN
address. Android pins that certificate, stores its bearer token encrypted with Android Keystore, and
can rediscover the same computer after a route/DHCP change.

The prior ADB-reverse transport remains only as a developer/recovery path.

```text
LOCAL_VEXSTREAM_PAIRING
!= VEXHOME_MEMBERSHIP
!= CANONICAL_HOME_AUTHENTICATION
!= CAPABILITY_LEASE
!= REMOTE_ACCESS_AUTHORITY
```

VexStream 3.0.0 also has an explicit **Exit VexStream** control in System status. Closing a browser
window is not treated as process shutdown.

The matching Android consumer is in `vgong24/VexLife-Android`, PR #13 /
`VXG-101126-vexstream-mobile-stream-offline`.

The historical 2.2.0 release manifest remains evidence for the preserved Explore base; current 3.0.0
mobile-candidate proof is produced by the branch qualification/build workflow.
