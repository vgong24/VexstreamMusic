# VexStream Music

`[VXG RealForever]`

VexStream Music is a local-first music library and playback surface. This public repository is the durable source/provenance home for the product lineage that was previously preserved through private Vextreme-SDK continuity snapshots and versioned local artifacts.

## Current candidate: 1.1.7

The current candidate repairs the filtered-Songs playback regression **and** the executable startup/version guard that caused the prior 1.1.6 candidate to reopen an already-running 1.1.5 session.

### User behavior

A Songs search or genre filter is a **display lens**, not the playback universe.

- row/double-click Play uses the full sorted library;
- idle player Play in Songs uses the full sorted library;
- **Play all** uses the full sorted library;
- **Shuffle all** uses the full sorted library;
- with shuffle active, the chosen song remains current and every other library song becomes shuffled future playback;
- row-level **Add to queue** remains an intentional one-track action;
- bulk selection/queue still acts only on tracks the user explicitly selected.

### Startup/version invariant

The Windows executable has multiple representations of its version: served UI, server/helper string data, and a compiled machine-code comparison used when probing an already-running VexStream session at `/health`.

All of them must agree with the shipped release version. 1.1.6 failed this invariant: its UI/data markers said 1.1.6, while the compiled startup comparison still recognized 1.1.5 as “this same version.” That allowed 1.1.6 to open the old 1.1.5 server and exit.

1.1.7 patches and tests that compiled guard explicitly.

## Exact recovery baseline

```text
VexStreamMusic-1.1.5.exe
sha256=e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f
bytes=6903808
Go=1.23.2
GOOS=windows
GOARCH=amd64
```

The 1.1.5 executable contains its browser UI as one embedded UTF-8 HTML/JavaScript document. That exact document was recovered and is the source anchor for the current UI refinement.

The original stripped Go backend source for 1.1.5 has **not** been reconstructed or claimed. The release patcher therefore fails closed on the exact 1.1.5 executable digest and on exact recovered code/data windows.

## Repository layout

```text
src/ui/index.html
  recovered and repaired current browser UI

tests/ui-queue-regression.test.mjs
  executable JS behavior tests for filtered playback / queue semantics

tests/artifact-regression.py
  exact compiled-artifact identity, version-consistency, and byte-diff tests

tools/patch_v1_1_5_to_v1_1_7.py
  deterministic, digest-bound formation from the exact supplied 1.1.5 executable

docs/1.1.7-FILTERED-PLAYBACK-AND-LAUNCH-GUARD.md
  causal diagnosis, regression boundary, and evidence limitations
```

## Current qualification

```text
UI behavior tests          7 PASS / 0 FAIL
artifact regression tests 15 PASS / 0 FAIL
inline JS syntax parse     PASS
patch reproducibility      PASS
wrong-baseline rejection   PASS
Windows runtime smoke      NOT_RUN_NO_WINDOWS_EXECUTION_SURFACE
```

The absence of a Windows execution surface is kept explicit. Structural/artifact proof does not masquerade as target-Windows runtime proof.

## Historical continuity

The earlier private continuity checkpoint remains in `vgong24/Vextreme-SDK` draft PR #1630. It is provenance/history, not the active public VexStream source writer.