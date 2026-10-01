#!/usr/bin/env python3
"""Run the source-level VexStream Music qualification suite."""
from __future__ import annotations

import argparse
import json
from pathlib import Path
import subprocess
import sys


def run(name: str, command: list[str], root: Path) -> dict:
    proc = subprocess.run(command, cwd=root, capture_output=True, text=True)
    row = {
        "name": name,
        "command": command,
        "exitCode": proc.returncode,
        "stdout": proc.stdout,
        "stderr": proc.stderr,
        "status": "PASS" if proc.returncode == 0 else "FAIL",
    }
    if proc.returncode != 0:
        raise RuntimeError(json.dumps(row, indent=2))
    return row


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--artifact", type=Path)
    parser.add_argument("--browser-executable")
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    root = args.root.resolve()
    evidence = root / "docs" / "evidence"
    evidence.mkdir(parents=True, exist_ok=True)

    steps: list[dict] = []
    static_cmd = [sys.executable, str(root / "tests" / "static_release_checks.py"), "--root", str(root), "--report", str(evidence / "static-release-checks.json")]
    if args.artifact:
        static_cmd += ["--artifact", str(args.artifact.resolve())]
    steps.append(run("static-release-checks", static_cmd, root))

    ui_cmd = [sys.executable, str(root / "tests" / "ui_runtime_regression.py"), "--html", str(root / "ui" / "index.html"), "--report", str(evidence / "ui-runtime-regression.json")]
    if args.browser_executable:
        ui_cmd += ["--browser-executable", args.browser_executable]
    steps.append(run("ui-runtime-regression", ui_cmd, root))
    steps.append(run("real-media-server-regression", [sys.executable, str(root / "tests" / "media_server_regression.py"), "--root", str(root), "--report", str(evidence / "real-media-server-regression.json")], root))
    steps.append(run("go-duplicate-quality-tests", ["go", "test", "main.go", "procattr_unix.go", "duplicate_logic_test.go", "import_runtime_test.go", "discovery_radio_test.go"], root))
    steps.append(run("python-bridge-compile", [sys.executable, "-m", "py_compile", str(root / "media_bridge.py")], root))
    steps.append(run("python-bridge-self-test", [sys.executable, str(root / "media_bridge.py"), "self-test"], root))

    report = {
        "schemaVersion": "vexstream.release-qualification/v1",
        "appVersion": "2.2.0",
        "sourceRoot": str(root),
        "artifact": str(args.artifact.resolve()) if args.artifact else None,
        "steps": steps,
        "status": "PASS",
    }
    report_path = args.report or evidence / "qualification-summary.json"
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
