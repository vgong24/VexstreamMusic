# VexStream Music — recovered canonical 1.1.5 source checkpoint

`[VXG RealForever]`

## Why this checkpoint exists

The VexStream Music product lineage was developed and qualified before this public repository existed. A later recovery lane had only the stripped Windows 1.1.5 executable and therefore began reconstructing/patching 1.1.6/1.1.7 behavior from binary evidence.

The exact source package that produced the qualified 1.1.5 artifact has now been recovered from the original source handoff and is being placed into the existing recovery PR instead of opening a competing writer.

```text
repository=vgong24/VexstreamMusic
baseMain=5af7caa92a6502e53c0d0b0bd7f7e862154327ed
existingRecoveryPr=github.pull.vexstreammusic.1
existingRecoveryBranch=VXG-093026-vexstream-1.1.6-filtered-playback-queue
preRecoverySourceHead=badf2a1cc7a0cefc61567e0cc450ec36537a260a
```

## Exact recovered source identity

```text
sourceArchive=VexStreamMusic-1.1.5-Source.zip
sourceArchiveSha256=15d54909bd9227d4e0ef67f4879503d7e7fdf80366781f8ec2723a0d375c1243
sourceFileCount=29
appVersion=1.1.5
```

The source archive contains the real Go server/player source, embedded UI, Python provider bridge, platform-specific process adapters, deterministic build tooling, functional Chromium regression harness, release process documentation and retained evidence.

Fresh recovery qualification:

```text
staticReleaseChecks=38_PASS
chromiumInteractions=31_PASS__0_FAIL
browserRuntimeErrors=0
goDuplicateQualityArtistSuggestionTests=PASS
pythonBridgeCompile=PASS
pythonBridgeSelfTest=6_PASS
```

Most importantly, rebuilding Windows from these exact source bytes produced:

```text
VexStreamMusic-1.1.5.exe
bytes=6903808
sha256=e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f
```

That is byte-identical to the exact 1.1.5 executable used as the baseline by PR #1. This binds the recovered source to the artifact the binary-recovery lane was reverse-engineering.

## Repository source carrier

The exact source archive is preserved in text-safe form under:

```text
source-archives/1.1.5/
  README.md
  ASSEMBLE.py
  VexStreamMusic-1.1.5-Source.zip.b64.part00
  VexStreamMusic-1.1.5-Source.zip.b64.part01
  VexStreamMusic-1.1.5-Source.zip.b64.part02
  VexStreamMusic-1.1.5-Source.zip.b64.part03
```

Run:

```bash
python3 source-archives/1.1.5/ASSEMBLE.py
```

The assembler reconstructs the exact ZIP and refuses success unless SHA-256 is exactly `15d54909bd9227d4e0ef67f4879503d7e7fdf80366781f8ec2723a0d375c1243`.

Existing PR #1 binary-recovery artifacts such as `src/ui/index.html`, `tools/patch_v1_1_5_to_v1_1_7.py`, `tests/ui-queue-regression.test.mjs`, and `tests/artifact-regression.py` are intentionally retained as historical/diagnostic evidence. They are not promoted into canonical backend source merely because they exist.

## Source lineage through 1.1.5

```text
0.9.5  playback regression repair; real Chromium interaction harness
0.9.6  persistent History <- Current -> Up Next timeline; Album drill-down; truthful shuffle order
1.0.0  adaptive navigation/table/card shell; intent-first track actions
1.0.1  Windows/macOS process-source split; Mac cross-builds
1.0.2  macOS YouTube bootstrap/readiness split; reusable Terminal fallback direction
1.1.0  Artist terrain: Artist -> Albums -> searchable/sortable Songs
1.1.1  reverse Album -> credited Artist lattice
1.1.3  scope-root navigation correction; explicit Home/wordmark semantics; Now Playing lattice links
1.1.4  punctuation/token-normalized search; duplicate advisory + bounded quality comparison
1.1.5  existing-library artist suggestion for provider intake
```

1.1.2 briefly encoded an incorrect repeat-scope-to-global-Home interpretation and was corrected by 1.1.3; do not revive that behavior.

## Current truth versus later binary-recovery work

PR #1 also contains a later 1.1.7 binary-patched candidate. Its durable status says it is **not accepted** after a real Windows playback regression report.

```text
RECOVERED_1_1_5_SOURCE = exact producer source, rebuild-proven
1_1_7_BINARY_PATCH = in-flight diagnostic/candidate history, not accepted source truth
```

Do not continue product evolution by binary patching the stripped EXE now that producer source is available. If the 1.1.7 intended filtered-playback behavior remains desired, implement it against this recovered source, add focused regression coverage, rebuild all platforms, and rerun real Windows playback acceptance.

## Key product contracts preserved in source

- local-first library and `/media/<track>` byte-range playback;
- source folders are user-selected, not hard-coded;
- queue is a cursor-preserving timeline, not a consumable future-only list;
- visible shuffle order equals actual next-play order;
- Album and Artist terrain are bidirectionally navigable;
- repeat scope navigation returns to that scope root only;
- wordmark/Home are explicit global-home controls;
- Now Playing links to Track Details / Artist / Album without changing playback;
- search normalizes apostrophes/punctuation/diacritics and supports token prefixes;
- duplicate intake is advisory/evidence-based and does not invent codec-quality certainty;
- YouTube creator/channel provenance is not canonical artist identity;
- artist suggestion requires an existing library artist plus an explicit title-credit boundary;
- Windows + Mac Silicon + Mac Intel share one source lineage;
- unsigned Mac delivery currently uses the reusable Terminal launcher rather than claiming signed/notarized `.app` delivery;
- VexLife federation and AI music creation/publishing remain separate product frontiers.

## Successor start

```text
1. Read this file and source-archives/1.1.5/README.md.
2. Reconstruct the exact source ZIP with ASSEMBLE.py.
3. Run: python3 tests/run_qualification.py
4. Rebuild Windows and require SHA-256 e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f for exact baseline reproduction.
5. Read PR #1's later regression findings as diagnostic input, not canonical implementation source.
6. Make the next repair in source, not in the stripped executable.
7. Preserve one product/source writer and requalify affected behavior before another versioned handoff.
```

This recovery does not accept or merge PR #1, does not declare 1.1.7 good, does not perform VexLife networking, does not publish a release, and does not activate the separate VexStream Create frontier.
