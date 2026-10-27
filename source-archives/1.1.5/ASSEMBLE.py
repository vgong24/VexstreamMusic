#!/usr/bin/env python3
from pathlib import Path
import base64, hashlib

EXPECTED = "15d54909bd9227d4e0ef67f4879503d7e7fdf80366781f8ec2723a0d375c1243"
ROOT = Path(__file__).resolve().parent
parts = sorted(ROOT.glob("VexStreamMusic-1.1.5-Source.zip.b64.part*"))
if len(parts) != 4:
    raise SystemExit(f"expected 4 source parts, found {len(parts)}")
payload = "".join(p.read_text(encoding="utf-8").strip() for p in parts)
data = base64.b64decode(payload, validate=True)
digest = hashlib.sha256(data).hexdigest()
if digest != EXPECTED:
    raise SystemExit(f"source archive SHA-256 mismatch: expected {EXPECTED}, got {digest}")
out = ROOT / "VexStreamMusic-1.1.5-Source.zip"
out.write_bytes(data)
print(out)
print(f"sha256={digest}")
