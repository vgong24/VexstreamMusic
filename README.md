# VexStream Music 2.0.0 — source-native continuity baseline

`[VXG RealForever]`

VexStream Music 2.0.0 is the first release line formed from the recovered, byte-reproducible **1.1.5 complete source tree** inside the new public `vgong24/VexstreamMusic` continuity.

2.0.0 is intentionally not a rewrite. It preserves the qualified 1.1.5 product surface—local playback, persistent `History ← Current → Up Next`, Albums/Artists terrain, playlists, playback crops, duplicate-aware YouTube intake, normalized search, artist suggestions, responsive shell, and Windows/macOS source lineage—while making one user-requested playback semantic explicit and adding a missing real-server regression gate.

## 2.0.0 playback semantic

A Songs search or genre filter is a **display lens**, not a hidden replacement for the playback universe.

```text
filtered Songs rows
  = what is visible / selectable

full sorted library
  = Songs playback universe

explicit Add to queue / bulk Queue
  = only the tracks the user intentionally queues
```

Therefore:

- row/double-click Play from filtered Songs starts the selected track inside the full sorted library;
- with Shuffle active, the selected song remains current and every other library song appears once in shuffled future playback;
- **Play all** and **Shuffle all** use the full library even while a search is visible;
- row **Add to queue** remains a one-track action;
- bulk queue actions remain scoped to explicit selection;
- Home/recent and collection-scoped playback retain their established scopes.

## Source continuity

The recovered 1.1.5 source archive is independently bound by:

```text
VexStreamMusic-1.1.5-Source.zip
sha256=15d54909bd9227d4e0ef67f4879503d7e7fdf80366781f8ec2723a0d375c1243
```

and reproducibly builds the accepted historical Windows artifact:

```text
VexStreamMusic-1.1.5.exe
sha256=e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f
```

2.0.0 continues from that source—not from the rejected 1.1.6/1.1.7 binary-patching experiment.

## Qualification layers

```text
A  static / source / syntax
B  deterministic Chromium UI interaction
C  real source-built HTTP server + real MP3 scan + exact full/range media bytes
D  platform build/artifact structure
E  target-host human playback acceptance
```

A–D can be automated from source. E remains a distinct real Windows/macOS product acceptance surface.

Run:

```bash
python3 tests/run_qualification.py --root . --browser-executable /path/to/chromium
python3 tests/build_windows.py --root . --output ../VexStreamMusic-2.0.0.exe
```
