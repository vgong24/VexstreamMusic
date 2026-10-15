#!/usr/bin/env python3
"""Build and structurally qualify the isolated VexStream Music 4.0.0 Windows executable."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import struct
import subprocess
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
VERSION = "4.0.0"
PACKAGE = "./v2/cmd/vexstream-v4"
EXPECTED_NAME = f"VexStreamMusic-{VERSION}.exe"


def run(
    command: list[str],
    *,
    env: dict[str, str] | None = None,
) -> dict[str, Any]:
    proc = subprocess.run(
        command,
        cwd=ROOT,
        env=env,
        capture_output=True,
        text=True,
    )
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


def git_identity() -> tuple[str, str]:
    head = run(["git", "rev-parse", "HEAD"])
    tree = run(["git", "rev-parse", "HEAD^{tree}"])
    if head["exitCode"] != 0 or tree["exitCode"] != 0:
        return "", ""
    return head["stdout"].strip(), tree["stdout"].strip()


def go_files() -> list[str]:
    return [
        str(path.relative_to(ROOT))
        for path in sorted((ROOT / "v2").rglob("*.go"))
    ]


def pe_projection(path: Path) -> dict[str, Any]:
    data = path.read_bytes()
    require(len(data) >= 512, "output is too small to be a Windows executable")
    require(data[:2] == b"MZ", "missing DOS MZ header")
    pe_offset = struct.unpack_from("<I", data, 0x3C)[0]
    require(
        data[pe_offset : pe_offset + 4] == b"PE\0\0",
        "missing PE signature",
    )
    machine = struct.unpack_from("<H", data, pe_offset + 4)[0]
    optional_offset = pe_offset + 24
    magic = struct.unpack_from("<H", data, optional_offset)[0]
    subsystem = struct.unpack_from("<H", data, optional_offset + 68)[0]

    require(
        machine == 0x8664,
        f"expected AMD64 machine 0x8664, got 0x{machine:04x}",
    )
    require(
        magic == 0x20B,
        f"expected PE32+ optional header, got 0x{magic:04x}",
    )
    require(
        subsystem == 2,
        f"expected Windows GUI subsystem 2, got {subsystem}",
    )
    return {
        "byteLength": len(data),
        "sha256": hashlib.sha256(data).hexdigest(),
        "machine": "AMD64",
        "peMagic": "PE32+",
        "subsystem": "WINDOWS_GUI",
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--output",
        type=Path,
        default=ROOT / "dist" / EXPECTED_NAME,
    )
    parser.add_argument(
        "--report",
        type=Path,
        default=ROOT / "dist" / "VexStreamMusic-4.0.0-Windows-qualification.json",
    )
    parser.add_argument(
        "--source-ref",
        default=os.environ.get("VEXSTREAM_BUILD_REF", ""),
        help=(
            "Human/source label bound into main.buildRef. "
            "Defaults to the current git HEAD when available."
        ),
    )
    args = parser.parse_args()

    output = args.output.resolve()
    report_path = args.report.resolve()
    require(
        output.name == EXPECTED_NAME,
        f"output must be named {EXPECTED_NAME}",
    )
    output.parent.mkdir(parents=True, exist_ok=True)
    report_path.parent.mkdir(parents=True, exist_ok=True)

    head, tree = git_identity()
    source_ref = args.source_ref.strip() or head or "vexstream.desktop-v4"

    files = go_files()
    require(files, "no V4 Go source files found")

    steps: list[dict[str, Any]] = []

    formatted = run(["gofmt", "-d", *files])
    steps.append(formatted)
    require(
        formatted["exitCode"] == 0 and not formatted["stdout"].strip(),
        "gofmt reports a V4 source delta:\n" + formatted["stdout"],
    )

    host_tests = run(["go", "test", "./v2/..."])
    steps.append(host_tests)
    require(
        host_tests["exitCode"] == 0,
        "V4 Go tests failed:\n" + host_tests["stderr"],
    )

    windows_env = os.environ.copy()
    windows_env.update(
        {
            "GOOS": "windows",
            "GOARCH": "amd64",
            "CGO_ENABLED": "0",
        }
    )

    vetted = run(["go", "vet", "./v2/..."], env=windows_env)
    steps.append(vetted)
    require(
        vetted["exitCode"] == 0,
        "Windows V4 go vet failed:\n" + vetted["stderr"],
    )

    ldflags = [
        "-H=windowsgui",
        "-s",
        "-w",
        f"-X main.buildRef={source_ref}",
    ]
    built = run(
        [
            "go",
            "build",
            "-trimpath",
            "-ldflags",
            " ".join(ldflags),
            "-o",
            str(output),
            PACKAGE,
        ],
        env=windows_env,
    )
    steps.append(built)
    require(
        built["exitCode"] == 0,
        "Windows V4 build failed:\n" + built["stderr"],
    )

    version = run(["go", "version"])
    require(version["exitCode"] == 0, "go version failed")
    projection = pe_projection(output)

    report = {
        "schemaVersion": "vexstream.desktop-v4.windows-build-qualification/v1",
        "appVersion": VERSION,
        "sourceRef": source_ref,
        "sourceHead": head or None,
        "sourceTree": tree or None,
        "package": PACKAGE,
        "artifact": str(output),
        "goVersion": version["stdout"].strip(),
        "expectedRuntimeBuildRefPrefix": source_ref + ".bin-",
        "steps": steps,
        "artifactProjection": projection,
        "status": "PASS",
    }
    report_path.write_text(
        json.dumps(report, indent=2) + "\n",
        encoding="utf-8",
    )
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
