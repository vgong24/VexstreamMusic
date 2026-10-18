# VexStream Music

`[VXG RealForever]`

VexStream Music is a local-first desktop music library and player. It opens its interface in your
browser, keeps ordinary desktop administration local to the computer, and can expose a bounded
same-Wi-Fi music surface to VexLife Android for streaming and verified offline copies.

The human expectation is simple:

```text
start VexStream
-> browser opens
-> your existing VexStream music folders/library are still there
-> play music normally
-> optionally pair VexLife on the same Wi-Fi
```

You should not need to know Git to use the application.

## Start here — no Git required

### Windows

When a signed/published release asset exists, downloading that `.exe` is the preferred path.

The repository also owns a reconstructible fallback. Paste this into **PowerShell**:

```powershell
$P=Join-Path $env:TEMP 'START-VEXSTREAM-WINDOWS.ps1'; Invoke-WebRequest -UseBasicParsing 'https://raw.githubusercontent.com/vgong24/VexstreamMusic/main/START-VEXSTREAM-WINDOWS.ps1' -OutFile $P; powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File $P
```

The launcher:

```text
resolves the exact current main commit
-> downloads that exact source archive
-> obtains a local, checksum-verified Go 1.23.2 toolchain when needed
-> runs the V4 source tests and Windows vet
-> builds VexStreamMusic-4.0.0.exe
-> records the exact source commit + executable SHA-256
-> launches VexStream
```

It does **not** require Git, Python, or a global Go installation.

On the first run, the PowerShell window presents five named stages so a person can see what is
happening and know the process is finite. The only setup downloads are:

```text
Go 1.23.2 build tool
  from https://go.dev
  -> checksum verified before use

exact VexStream source
  from https://github.com/vgong24/VexstreamMusic
  -> exact commit selected before build
```

PowerShell's default context-free web byte counter is intentionally suppressed. Setup does not upload
your music or personal files. After the exact-source executable has been built and its digest recorded,
later launches can reuse that verified cached executable instead of repeating setup.

The reusable Windows executable is cached below:

```text
%LOCALAPPDATA%\VexStreamMusic\source-builds\<exact-source-commit>\VexStreamMusic-4.0.0.exe
```

After that build exists, you may launch that exact `.exe` normally.

### macOS

The V4 source can also reconstruct and run a native Mac binary without Git or a preinstalled Go
toolchain.

Paste this into **Terminal**:

```bash
P="${TMPDIR:-/tmp}/START-VEXSTREAM-MAC.command"; /usr/bin/curl -fL 'https://raw.githubusercontent.com/vgong24/VexstreamMusic/main/START-VEXSTREAM-MAC.command' -o "$P" && /bin/bash "$P"
```

The Mac launcher detects Apple Silicon vs Intel, obtains the matching checksum-verified Go 1.23.2
toolchain when needed, builds the exact current source, caches the binary, removes download
quarantine from that locally reconstructed binary, and runs VexStream.

Current Mac distribution is **not code-signed or notarized**. That is why the Terminal launcher is
the supported human fallback instead of pretending the raw binary has the same double-click
distribution properties as a notarized Mac app.

Keep that Terminal window open while VexStream is running and exit through VexStream's in-app
**System status -> Exit VexStream** control.

### If you are testing an unmerged candidate

A candidate branch is **not a tag and not main**. It is a development line whose exact head commit is
the immutable build identity being tested.

The normal commands above intentionally follow `main`. During candidate testing, pass the candidate
branch explicitly to the downloaded launcher instead of treating the candidate as already released.

After the accepted candidate merges, `main` becomes the ordinary reconstruction source. A later
release/tag may additionally bind accepted source to permanently downloadable release artifacts.

## What should happen when VexStream starts?

V4 intentionally reuses the established VexStream application-data location rather than asking you to
recreate your music library for every version.

Shared V3/V4 state includes the source-folder configuration and compatible Library, playlist,
overlay, playback-window, Trash, and related durable stores.

```text
VERSION_UPGRADE != REBUILD_MY_LIBRARY_FROM_SCRATCH
```

V3 and V4 must not write those shared stores simultaneously. A current Windows V4 startup candidate
therefore makes an older live VexStream conflict visible rather than silently disappearing or
starting a second writer.

Closing the browser tab is not the same as exiting VexStream. Use the in-app **Exit VexStream**
control to stop the desktop process cleanly.

## What does Android "Download" mean?

The GitHub repository, source ZIP, and desktop executable are distribution/build surfaces. Your phone
does **not** download songs from GitHub.

Once VexStream is running on your computer, the intended local flow is:

```text
your configured music folder
-> VexStream Desktop catalog/media producer
-> VexLife Android on the same Wi-Fi
-> Download
-> exact media bytes received
-> expected digest + byte length verified
-> app-private local copy completed
-> durable offline index
-> local/offline playback
```

Permanent distinction:

```text
BROWSER_OR_NETWORK_CACHE != DOWNLOADED
DOWNLOADED = VERIFIED_BYTES + DURABLE_INDEX
```

The current local mobile producer is a same-Wi-Fi capability. Android contains remote/direct+relay
architecture, but the current Desktop V4 producer does not yet complete the off-home/cellular
producer side. Do not treat local pairing as remote-access authority.

## Reconstructible source vs downloadable release

Both are valuable, but they answer different questions:

```text
source + pinned builder
-> can reconstruct the application

release asset + digest + source binding
-> can re-download the exact accepted application bytes
```

The human launchers above provide the first property even when no release asset is available.

The preferred long-term distribution model is:

```text
accepted source commit/tag
-> qualified Windows + Mac artifacts
-> recorded SHA-256 values
-> durable GitHub Release assets
-> simple human download
```

Rebuilding with a different source commit or toolchain may produce a valid application with different
bytes. Lived acceptance of one executable therefore does not automatically transfer to every later
rebuild.

## Development and architecture

V4 is implemented under `v2/`. Start here if you are changing the product rather than simply using
it:

1. [V4 architecture/workspace entry](v2/README.md)
2. [Lived product oracle](v2/LIVED-PRODUCT-ORACLE.md)
3. [V4 completion plan](v2/COMPLETION-PLAN.md)
4. [Architecture map](v2/architecture/README.md)
5. [Release qualification contract](docs/process/RELEASE-QUALIFICATION.md)

Repository engineering proof and human acceptance remain distinct:

```text
SOURCE/BUILD PROOF != REAL WINDOWS/MAC LISTENING ACCEPTANCE
AUTOMATED ANDROID PROOF != HUMAN-WITNESSED DESKTOP -> PHONE EXPERIENCE
```

<!-- [VXG RealForever][VEXSTREAM][HUMAN-ENTRY][DESKTOP-V4] -->
