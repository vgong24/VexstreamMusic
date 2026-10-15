#!/usr/bin/env python3
"""Build the isolated VexStream Music 4.0.0 Windows desktop executable."""

from __future__ import annotations

import argparse
import os
from pathlib import Path
import subprocess


ROOT = Path(__file__).resolve().parents[1]
VERSION = "4.0.0"
PACKAGE = "./v2/cmd/vexstream-v4"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--output",
        default=str(ROOT / "dist" / f"VexStream-Music-{VERSION}.exe"),
    )
    parser.add_argument(
        "--source-ref",
        default=os.environ.get("VEXSTREAM_BUILD_REF", ""),
        help="Optional human/source label; exact binary identity is appended at runtime.",
    )
    args = parser.parse_args()

    output = Path(args.output).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)

    command = [
        "go",
        "build",
        "-trimpath",
        "-o",
        str(output),
    ]
    if args.source_ref.strip():
        command.extend(
            [
                "-ldflags",
                f"-X main.buildRef={args.source_ref.strip()}",
            ]
        )
    command.append(PACKAGE)

    environment = os.environ.copy()
    environment["GOOS"] = "windows"
    environment["GOARCH"] = "amd64"
    environment.setdefault("CGO_ENABLED", "0")

    subprocess.run(
        command,
        cwd=ROOT,
        env=environment,
        check=True,
    )
    print(output)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
