# VexStream Music

`[VXG RealForever]`

VexStream Music is a local-first music library and playback surface. This repository begins the durable public source/provenance history that was previously scattered across private continuity snapshots and versioned local artifacts.

## Current recovery baseline

The current executable baseline is **VexStream Music 1.1.5** supplied as a Windows PE artifact.

```text
VexStreamMusic-1.1.5.exe
sha256=e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f
bytes=6903808
```

The 1.1.5 executable contains its browser UI as an embedded UTF-8 HTML/JavaScript document. That exact UI has been recovered into `src/ui/index.html` and is used as the provenance anchor for the first repository-backed refinement.

The original Go backend source for 1.1.5 has **not** been recovered into this repository. Do not mistake the recovered UI or the patcher for a full source reconstruction.

## 1.1.6 regression repair

1.1.5 let the Songs search/genre filter accidentally become the playback universe. Starting playback from a filtered Songs result therefore replaced the queue with only the visible filtered subset. Clearing the filter later did not restore the full library queue.

1.1.6 separates those concerns:

```text
Songs search / genre filter = display lens
Songs playback context       = full sorted library
```

When shuffle is enabled, the chosen track remains current and every other library track is in the shuffled future queue. **Play all**, **Shuffle all**, and idle player Play no longer inherit the Songs filter. Row-level **Add to queue** remains an intentional one-track action.

See `docs/1.1.6-FILTERED-PLAYBACK-REGRESSION.md` and `tests/ui-queue-regression.test.mjs`.

## Reproducing the Windows 1.1.6 patch

The patcher is deliberately fail-closed. It only accepts the exact qualified 1.1.5 executable SHA-256 above, patches the exact embedded UI function, preserves the embedded Go-string byte length, and advances the three equal-width version markers to `1.1.6`.

```bash
python tools/patch_v1_1_5_to_v1_1_6.py \
  VexStreamMusic-1.1.5.exe \
  VexStreamMusic-1.1.6.exe
```

Current qualified output from that exact input:

```text
VexStreamMusic-1.1.6.exe
sha256=431148f2ef5fd3d2c76828d171955babc0a439ff7e65530a05af5a2ed375c156
bytes=6903808
```

This is a deterministic binary refinement of the supplied 1.1.5 build, **not** a from-source Go rebuild. Windows behavioral validation remains distinct from the structural/source regression tests in this repository.

## Local checks

```bash
node --test tests/ui-queue-regression.test.mjs
```

The test suite verifies the recovered 1.1.6 UI behavior directly.