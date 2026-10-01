# VexStream Music 2.0.1 — Windows import-runtime repair

`[VXG RealForever]`

VexStream Music 2.0.1 continues the source-native 2.x line established from the byte-reproducible 1.1.5 source tree. It preserves the 2.0.0 playback/filter correction and repairs the Windows YouTube import bootstrap observed during target-host use.

## 2.0.1 repair

A Windows host can have a healthy private Python/yt-dlp runtime while FFmpeg is already installed through WinGet but not visible on the VexStream process PATH. In 2.0.0 the setup path bundled Python provisioning and `winget install Gyan.FFmpeg` into one PowerShell command. WinGet can report "No available upgrade found" for an already-installed package, which caused the whole setup request to be surfaced as a failure even though YouTube search was already usable.

2.0.1 separates those states and effects:

```text
Python + yt-dlp ready
    !=
FFmpeg ready
    !=
MP3 import ready
```

On Windows:

- the private Python virtual environment is created directly, without a monolithic PowerShell bootstrap;
- pip/yt-dlp setup and WinGet/FFmpeg setup are separate commands;
- VexStream searches both the WinGet Links directory and nested `Gyan.FFmpeg_*` package directories for `ffmpeg.exe`;
- a WinGet no-upgrade/nonzero result does not erase already-earned YouTube search readiness;
- `Download & add` performs an import-readiness preflight and will not call `/api/import/start` until FFmpeg is actually discoverable;
- the UI explicitly distinguishes "YouTube search is ready" from "MP3 download is ready".

## Preserved 2.0.0 playback semantic

Songs search/genre filtering remains a display lens. Playback from Songs uses the full sorted library; explicit Add-to-queue and bulk Queue operations remain intent-scoped.

## Source continuity

The 2.x source line remains grounded in the exact recovered 1.1.5 source archive:

```text
VexStreamMusic-1.1.5-Source.zip
sha256=15d54909bd9227d4e0ef67f4879503d7e7fdf80366781f8ec2723a0d375c1243
```

which reproducibly builds the historical 1.1.5 Windows artifact:

```text
VexStreamMusic-1.1.5.exe
sha256=e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f
```

## Qualification layers

```text
A  static / source / syntax
B  deterministic Chromium UI interaction
C  real source-built HTTP server + real MP3 scan + exact full/range media bytes
D  Go import-runtime unit regression
E  platform build/artifact structure
F  target-host Windows import acceptance
```

A–E are source-automated. F remains a distinct target-Windows confirmation because WinGet/package registration is a Windows-host effect.

Run:

```bash
python3 tests/run_qualification.py --root . --browser-executable /path/to/chromium
python3 tests/build_windows.py --root . --output ../VexStreamMusic-2.0.1.exe
```
