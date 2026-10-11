# Build VexStream Music 2.2.0

`[VXG RealForever]`

## Source qualification

```bash
python3 tests/run_qualification.py --root . --browser-executable /path/to/chromium
```

The suite covers static/source invariants, deterministic Chromium interaction behavior (including 🌐 Explore Session v1 multi-axis behavior), real source-built server/media delivery, Go duplicate/import/discovery regressions, and the Python provider bridge.

## Windows

```bash
python3 tests/build_windows.py \
  --root . \
  --output ../VexStreamMusic-2.2.0.exe \
  --report docs/evidence/windows-build-qualification.json
```

## All platforms

```bash
python3 tests/build_all_platforms.py \
  --root . \
  --output-dir ../release \
  --report docs/evidence/all-platforms-distribution.json
```

Target-host preview availability, provider ads, real Windows audio-device output, and real provider responses remain host/provider observations rather than cross-build proof.

## VexLife phone connection

The lived ADB-reverse proof is preserved as a developer/recovery route, but it is no longer the
normal human path.

For the current same-Wi-Fi candidate:

```text
Windows
  run VexStreamMusic-2.2.0.exe
  -> click the phone icon
  -> first-time pairing window opens for 5 minutes
  -> six-digit confirmation code appears

Android
  Home -> VexStream Music -> Find VexStream
  -> select the nearby VexStream computer
  -> compare the six-digit code
  -> Codes match · Request access

Windows
  -> Allow

Android
  -> library appears
  -> Play streams through the phone
  -> Download retains a verified app-private copy
```

No ADB, USB cable, port number, or connection ID is part of that normal pairing flow.

If Windows asks whether VexStream may communicate on the network, allow it on the private/home
network so mDNS discovery and the dedicated HTTPS mobile listener can work.

The secure mobile LAN surface is separate from the localhost desktop administration server:

```text
ordinary VexStream UI/admin = loopback only
mobile pairing/catalog/media = dedicated HTTPS + pinned certificate + per-device bearer
```

A pairing window that expires with no remembered or in-flight phone now removes the nearby
advertisement and closes the LAN listener. A remembered phone keeps the secure mobile listener
available for automatic reconnect on future VexStream launches.

If the desktop receives another DHCP address, VexLife can use Find VexStream to rediscover the same
pinned certificate identity without requiring another desktop approval.

The earlier ADB route remains source-owned developer recovery only:

```text
adb reverse tcp:18765 tcp:18765
```

Neither local transport is canonical VexHome remote authority:

```text
LOCAL_VEXSTREAM_PAIRING
!= VEXHOME_MEMBERSHIP
!= CANONICAL_AUTHENTICATION
!= CAPABILITY_LEASE
!= REMOTE_ACCESS_AUTHORITY
```

Close an older VexStream process before launching a bridge build made from this branch. The current
desktop single-instance shortcut identifies the product by version and an already-running non-bridge
2.2.0 process cannot acquire the new listener retroactively.

