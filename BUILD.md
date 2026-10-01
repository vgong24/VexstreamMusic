# Build VexStream Music 2.0.0

`[VXG RealForever]`

## Source qualification

```bash
python3 tests/run_qualification.py --root . --browser-executable /path/to/chromium
```

The suite covers static/source invariants, deterministic Chromium interaction behavior, a real source-built server/media regression, Go duplicate/quality/artist-inference tests, and the Python bridge.

## Windows

```bash
python3 tests/build_windows.py \
  --root . \
  --output ../VexStreamMusic-2.0.0.exe \
  --report docs/evidence/windows-build-qualification.json
```

## All platforms

```bash
python3 tests/build_all_platforms.py \
  --root . \
  --output-dir ../release \
  --report docs/evidence/all-platforms-distribution.json
```
