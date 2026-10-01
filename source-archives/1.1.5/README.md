# VexStream Music 1.1.5 exact source carrier

`[VXG RealForever]`

This directory preserves the exact `VexStreamMusic-1.1.5-Source.zip` as four UTF-8 base64 parts so GitHub/connector readers can retrieve the bytes without binary-attachment ambiguity.

```text
expectedZipSha256=15d54909bd9227d4e0ef67f4879503d7e7fdf80366781f8ec2723a0d375c1243
partCount=4
```

To reconstruct locally from a checkout:

```bash
python3 source-archives/1.1.5/ASSEMBLE.py
```

The assembler concatenates the four parts in lexical order, strict-base64 decodes them, verifies the exact SHA-256, and writes `VexStreamMusic-1.1.5-Source.zip` beside the parts.

The ZIP contains 29 files including `main.go`, `ui/index.html`, `media_bridge.py`, platform adapters, functional tests, build tooling, release process docs and retained evidence. Its Windows build reproduces the exact 1.1.5 executable SHA-256 `e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f`.
