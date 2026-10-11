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

## Android mobile-bridge lived proof

The mobile bridge on this branch is a loopback-only development surface. It is intentionally not the
final LAN/Home transport.

Build the Windows candidate through the ordinary source-owned route above, run it with the normal
VexStream library, then map the Android device's loopback port to the desktop bridge:

```text
adb reverse tcp:18765 tcp:18765
```

The matching Android consumer branch is:

```text
vgong24/VexLife-Android
VXG-101126-vexstream-mobile-stream-offline
```

Expected first proof:

```text
desktop VexStream bridge running
-> adb reverse established
-> Android VexStream Music screen reports Desktop library connected
-> real desktop tracks appear without desktop paths/source folders
-> Play streams one track through phone Media3
-> Download stores and verifies an app-private copy
-> stop/remove desktop transport
-> downloaded track remains playable on the phone
```

Close an older VexStream process before launching a bridge build made from this branch. The current
desktop single-instance shortcut identifies the product by version and an already-running non-bridge
2.2.0 process cannot acquire the new listener retroactively.

The ADB route proves product/media behavior only:

```text
ADB reverse != pairing != authentication != authorization != product Home gateway
```
