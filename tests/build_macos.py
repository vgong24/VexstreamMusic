#!/usr/bin/env python3
"""Cross-build and structurally qualify raw macOS VexStream Music executables."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import stat
import struct
import subprocess

VERSION = "2.0.1"
ARCH_CPU = {
    "arm64": 0x0100000C,
    "amd64": 0x01000007,
}
ARCH_NAME = {
    "arm64": "Apple-Silicon",
    "amd64": "Intel",
}


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


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as fh:
        for chunk in iter(lambda: fh.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def macho_projection(path: Path, arch: str) -> dict:
    data = path.read_bytes()
    require(len(data) >= 32, "output is too small to be a Mach-O executable")
    magic = struct.unpack_from("<I", data, 0)[0]
    require(magic == 0xFEEDFACF, f"expected 64-bit little-endian Mach-O magic, got 0x{magic:08x}")
    cpu = struct.unpack_from("<I", data, 4)[0]
    require(cpu == ARCH_CPU[arch], f"expected {arch} CPU type 0x{ARCH_CPU[arch]:08x}, got 0x{cpu:08x}")
    mode = path.stat().st_mode
    require(mode & stat.S_IXUSR, "macOS binary is not executable")
    return {
        "byteLength": len(data),
        "sha256": hashlib.sha256(data).hexdigest(),
        "format": "Mach-O 64-bit",
        "architecture": arch,
        "cpuType": f"0x{cpu:08x}",
        "executable": True,
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--output-dir", type=Path, required=True)
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    root = args.root.resolve()
    output_dir = args.output_dir.resolve()
    output_dir.mkdir(parents=True, exist_ok=True)

    steps: list[dict] = []
    steps.append(run(["gofmt", "-d", "main.go", "procattr_windows.go", "procattr_unix.go"], root))
    require(steps[-1]["exitCode"] == 0 and not steps[-1]["stdout"].strip(), "gofmt reports a source delta")

    artifacts = []
    for arch in ("arm64", "amd64"):
        env = dict(os.environ)
        env.update({"GOOS": "darwin", "GOARCH": arch, "CGO_ENABLED": "0"})
        vet = run(["go", "vet", "main.go", "procattr_unix.go"], root, env)
        steps.append({"architecture": arch, "phase": "go-vet", **vet})
        require(vet["exitCode"] == 0, f"darwin/{arch} go vet failed: {vet['stderr']}")

        binary = output_dir / f"VexStreamMusic-{VERSION}-macOS-{ARCH_NAME[arch]}"
        build = run(["go", "build", "-trimpath", "-ldflags=-s -w", "-o", str(binary), "main.go", "procattr_unix.go"], root, env)
        steps.append({"architecture": arch, "phase": "go-build", **build})
        require(build["exitCode"] == 0, f"darwin/{arch} go build failed: {build['stderr']}")
        binary.chmod(binary.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
        projection = macho_projection(binary, arch)
        artifacts.append({
            "architecture": arch,
            "binary": str(binary),
            "binaryProjection": projection,
            "sha256": sha256(binary),
            "signed": False,
            "notarized": False,
        })

    version = run(["go", "version"], root)
    require(version["exitCode"] == 0, "go version failed")
    report = {
        "schemaVersion": "vexstream.macos-build-qualification/v2",
        "appVersion": VERSION,
        "sourceRoot": str(root),
        "goVersion": version["stdout"].strip(),
        "steps": steps,
        "artifacts": artifacts,
        "distributionMode": "RAW_EXECUTABLE_PLUS_REUSABLE_TERMINAL_GUIDE",
        "hostExecution": "NOT_RUN_ON_MACOS",
        "codeSigning": "NOT_PERFORMED",
        "notarization": "NOT_PERFORMED",
        "status": "PASS_CROSS_BUILD_AND_RAW_BINARY_STRUCTURE__REAL_MAC_HOST_ACCEPTANCE_PENDING",
    }
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
