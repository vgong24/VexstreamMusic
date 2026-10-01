# VexStream Music release qualification

Continuity: `[VXG RealForever]`  
Process version: `vexstream.release-qualification/v1`

## Why this exists

A release is not qualified because the page renders, JavaScript parses, or a new feature appears in one screenshot. Playback is an interaction graph: activating a song must load it, start it, preserve a useful queue, allow mid-stream switching, keep transport controls reachable, and preserve non-destructive crop behavior.

The 0.9.4 regression demonstrated the gap:

```text
static syntax PASS
+ visually plausible UI
!= executable playback behavior PASS
```

## Required release loop

1. **Bind the baseline and candidate.** Record the last user-observed stable reference, the candidate version, and the exact changed source paths. “Latest” is not an identity.
2. **Reproduce the report before repair.** Run the candidate through the same deterministic interaction sequence used for the baseline. Preserve browser errors and observable player/queue state.
3. **Classify findings separately.** Use `candidate regression`, `pre-existing latent defect`, `harness`, `environment`, or `unknown`. One report may contain more than one class.
4. **Add a failing regression proof.** The test must fail for the observed behavior, not merely search for a source string.
5. **Repair the smallest causal source.** Do not redesign adjacent features during stabilization.
6. **Run the full affected interaction gate.** A playback repair must cover all playback controls and preserved crop behavior, not only the line that threw.
7. **Build final bytes once the source gate passes.** The distributable filename must contain the semantic version.
8. **Re-run qualification against the final artifact binding.** The test report must name the final versioned artifact.
9. **Record the release ledger.** Include source identity, executable digest, tests actually run, evidence surface, and limits.
10. **Keep human acceptance distinct.** Headless deterministic browser proof is required, but it is not relabeled as a real Windows listening-session acceptance.

## Mandatory source-level gates

### Static / formation

- Go, Python bridge, and UI version strings agree.
- Embedded JavaScript passes `node --check`.
- DOM IDs are unique.
- Required player, queue, and crop functions/bindings exist.
- `gofmt` reports no delta.
- Windows `go vet` passes.
- macOS arm64 and amd64 `go vet` pass.
- Windows-only process attributes are isolated from shared source.
- Final Windows executable name is exactly `VexStreamMusic-<version>.exe`.
- macOS arm64/amd64 outputs are raw versioned Mach-O executables with executable permissions preserved.
- The source-managed all-platform builder emits one master ZIP with exactly one Windows EXE and, per Mac architecture, one raw executable plus one reusable digest-bound Terminal guide.

### Browser-runtime regression

The actual embedded `ui/index.html` is executed in Chromium with a deterministic local API/audio fixture. The gate must prove:

- initial render produces no browser runtime error;
- desktop navigation collapses/expands and narrow navigation becomes a complete overlay drawer;
- configurable song fields and Title-space preference apply without corrupting table state;
- narrow windows switch to track cards with no document-level horizontal overflow;
- essential compact-player controls remain within the viewport;
- Track actions expose intent hierarchy, crop visualization, one Close control, and inline playlist addition;
- double-clicking a library row starts playback;
- a filtered Songs result starts inside the full sorted library playback universe, while explicit queue actions remain scoped;
- double-clicking another song switches mid-stream;
- Next and Previous change to the expected track;
- queue History / Current / Up Next remains perceptually complete after advancing;
- the current track is explicitly identifiable across queue/library collection views;
- Add to queue provides immediate acknowledgement;
- album cards open to all mapped tracks and support album-scope queueing;
- Artist cards are navigation-first visual surfaces with no default Play/Shuffle controls;
- Artist detail exposes artist-scoped search/sort plus a collapsible bounded album shelf that filters the same song terrain;
- drag-reordering a past/future queue node preserves the currently playing node;
- Shuffle visibly reorders the future sequence and Next follows that visible order;
- Play/Pause toggles;
- collapsed queue does not intercept player controls;
- Play recovers an unloaded/stale current source;
- non-destructive crop start is applied;
- crop end advances to the next queued track;
- the full interaction run produces no browser runtime error.

### Real server / media

The actual source-built Go server is launched against an isolated real MP3 fixture. The gate must prove:

- `/health` and the served UI agree on the source version;
- a real source folder is scanned and produces a library track;
- `/media/<track>` returns the exact MP3 bytes;
- `Accept-Ranges: bytes` is present;
- a bounded Range request returns `206 Partial Content`, the correct `Content-Range`, and the exact requested byte prefix.

This proves source-server media delivery. It still does not prove browser decoding or physical audio-device output.

### Bridge / build

- Python bridge compiles.
- Python bridge self-test passes.
- Windows x64 GUI cross-build succeeds.
- macOS arm64 and amd64 cross-builds succeed and have the expected Mach-O CPU identity.
- The master all-platform ZIP contains the exact five-file platform layout and preserves executable permissions on both Mac binaries.
- Final Windows and Mac artifacts are nonempty and SHA-256 is recorded.

## Evidence rules

`PASS` belongs only to an executed gate. A source inspection is a static finding, not a browser run. A deterministic Chromium run is browser-runtime proof, not a real Windows or Mac audio-device acceptance. A cross-build is build proof, not proof that the user launched the Windows or Mac executable on a real host.

When a test cannot run, hold only the release claim that depends on it. Do not replace it with confidence or a screenshot.

## Commands

From the source root:

```bash
python tests/run_qualification.py --browser-executable /path/to/chromium
```

Build and structurally qualify the versioned Windows executable:

```bash
python tests/build_windows.py \
  --output ../VexStreamMusic-2.0.0-Windows/VexStreamMusic-2.0.0.exe \
  --report docs/evidence/windows-build-qualification.json
```

Then bind the final artifact into the full qualification run:

```bash
python tests/run_qualification.py \
  --artifact ../VexStreamMusic-2.0.0-Windows/VexStreamMusic-2.0.0.exe \
  --browser-executable /path/to/chromium
```

On a machine where Playwright manages its own Chromium, omit `--browser-executable` after installing `tests/requirements-test.txt` and the Playwright Chromium runtime.

Build and structurally qualify both raw macOS architectures:

```bash
python tests/build_macos.py \
  --output-dir ../VexStreamMusic-2.0.0-macOS \
  --report docs/evidence/macos-build-qualification.json
```

A Mac cross-build PASS remains distinct from real-host launch, Gatekeeper, audio-device, Finder-folder-selection and quit/relaunch acceptance.

## Master distribution

Build the exact user-facing one-ZIP package:

```bash
python tests/build_all_platforms.py \
  --output-dir ../VexStreamMusic-2.0.0-release \
  --report docs/evidence/all-platforms-distribution.json
```

The canonical member set is exactly:

```text
Windows/VexStreamMusic-2.0.0.exe
Mac Silicon/VexStreamMusic-2.0.0-macOS-Apple-Silicon
Mac Silicon/COPY-PASTE-TO-TERMINAL.md
Mac Intel/VexStreamMusic-2.0.0-macOS-Intel
Mac Intel/COPY-PASTE-TO-TERMINAL.md
```
