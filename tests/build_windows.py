#!/usr/bin/env python3
"""Build and structurally qualify the versioned Windows VexStream executable."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import struct
import subprocess

VERSION = "2.0.1"
EXPECTED_NAME = f"VexStreamMusic-{VERSION}.exe"


def run(command: list[str], cwd: Path, env: dict[str, str] | None = None) -> dict:
    proc = subprocess.run(command, cwd=cwd, env=env, capture_output=True, text=True)
    return {
        "command": command,
        "exitCode": proc.returncode,
        "stdout": proc.stdout,
        "stderr": proc.stderr,
        "status": "PASS" if proc.returncode == 0 else "FAIL",
    }


def require(condition: bool, message: str) -> None:
    if not condition:
        raise RuntimeError(message)


def pe_projection(path: Path) -> dict:
    data = path.read_bytes()
    require(len(data) >= 512, "output is too small to be a Windows executable")
    require(data[:2] == b"MZ", "missing DOS MZ header")
    pe_offset = struct.unpack_from("<I", data, 0x3C)[0]
    require(data[pe_offset : pe_offset + 4] == b"PE\0\0", "missing PE signature")
    machine = struct.unpack_from("<H", data, pe_offset + 4)[0]
    optional_offset = pe_offset + 24
    magic = struct.unpack_from("<H", data, optional_offset)[0]
    require(machine == 0x8664, f"expected AMD64 machine 0x8664, got 0x{machine:04x}")
    require(magic == 0x20B, f"expected PE32+ optional header, got 0x{magic:04x}")
    subsystem = struct.unpack_from("<H", data, optional_offset + 68)[0]
    require(subsystem == 2, f"expected Windows GUI subsystem 2, got {subsystem}")
    return {
        "byteLength": len(data),
        "sha256": hashlib.sha256(data).hexdigest(),
        "machine": "AMD64",
        "peMagic": "PE32+",
        "subsystem": "WINDOWS_GUI",
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    root = args.root.resolve()
    output = args.output.resolve()
    require(output.name == EXPECTED_NAME, f"output must be named {EXPECTED_NAME}")
    output.parent.mkdir(parents=True, exist_ok=True)

    steps: list[dict] = []
    steps.append(run(["gofmt", "-d", "main.go", "procattr_windows.go", "procattr_unix.go"], root))
    require(steps[-1]["exitCode"] == 0 and not steps[-1]["stdout"].strip(), "gofmt reports a source delta")

    env = dict(os.environ)
    env.update({"GOOS": "windows", "GOARCH": "amd64", "CGO_ENABLED": "0"})
    steps.append(run(["go", "vet", "main.go", "procattr_windows.go"], root, env))
    require(steps[-1]["exitCode"] == 0, f"go vet failed: {steps[-1]['stderr']}")

    steps.append(run(["go", "build", "-trimpath", "-ldflags=-H=windowsgui -s -w", "-o", str(output), "main.go", "procattr_windows.go"], root, env))
    require(steps[-1]["exitCode"] == 0, f"go build failed: {steps[-1]['stderr']}")

    version = run(["go", "version"], root)
    require(version["exitCode"] == 0, "go version failed")
    projection = pe_projection(output)
    report = {
        "schemaVersion": "vexstream.windows-build-qualification/v1",
        "appVersion": VERSION,
        "sourceRoot": str(root),
        "artifact": str(output),
        "goVersion": version["stdout"].strip(),
        "steps": steps,
        "artifactProjection": projection,
        "status": "PASS",
    }
    report_path = args.report
    if report_path:
        report_path.parent.mkdir(parents=True, exist_ok=True)
        report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
