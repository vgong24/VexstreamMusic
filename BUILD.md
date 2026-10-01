# Build VexStream Music 2.1.0

`[VXG RealForever]`

## Source qualification

```bash
python3 tests/run_qualification.py --root . --browser-executable /path/to/chromium
```

The suite covers static/source invariants, deterministic Chromium interaction behavior (including Discovery Session v0), real source-built server/media delivery, Go duplicate/import/discovery regressions, and the Python provider bridge.

## Windows

```bash
python3 tests/build_windows.py \
  --root . \
  --output ../VexStreamMusic-2.1.0.exe \
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
