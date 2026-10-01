#!/usr/bin/env python3
"""Real VexStream HTTP/media regression against the actual Go server.

Unlike ui_runtime_regression.py, this launches source-built server code, adds an
isolated source folder, waits for reconciliation, and verifies exact media bytes
and HTTP Range behavior. It does not claim audio-device/browser-decoder proof.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

VERSION = "2.1.0"
FIXTURE_SHA256 = "602e675b3d27d2496ac968172b68ca1dc74ddaec455be92d41b51ec65fa359a7"


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def request_json(url: str, method: str = "GET", payload: dict | None = None) -> dict:
    data = None if payload is None else json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url, data=data, method=method)
    if data is not None:
        req.add_header("Content-Type", "application/json")
    with urllib.request.urlopen(req, timeout=5) as response:
        return json.loads(response.read().decode("utf-8"))


def free_port() -> int:
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    root = args.root.resolve()
    fixture = root / "tests" / "fixtures" / "vexstream-test-tone.mp3"
    fixture_bytes = fixture.read_bytes()
    if sha256(fixture_bytes) != FIXTURE_SHA256:
        raise AssertionError("synthetic MP3 fixture digest drift")

    checks: list[dict[str, str]] = []
    def passed(name: str) -> None:
        checks.append({"name": name, "status": "PASS"})

    with tempfile.TemporaryDirectory(prefix="vexstream-media-regression-") as td:
        temp = Path(td)
        music = temp / "music"
        music.mkdir()
        target = music / fixture.name
        shutil.copy2(fixture, target)
        binary = temp / "vexstream-test-server"
        build = subprocess.run(
            ["go", "build", "-trimpath", "-o", str(binary), "main.go", "procattr_unix.go"],
            cwd=root, capture_output=True, text=True,
        )
        if build.returncode != 0:
            raise RuntimeError(build.stdout + build.stderr)
        passed("native-source-server-build")

        port = free_port()
        config_home = temp / "config"
        env = dict(os.environ)
        env.update({"XDG_CONFIG_HOME": str(config_home), "VEXSTREAM_PORT": str(port)})
        process = subprocess.Popen([str(binary)], cwd=root, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        base = f"http://127.0.0.1:{port}"
        try:
            deadline = time.time() + 10
            health = None
            while time.time() < deadline:
                try:
                    health = request_json(base + "/health")
                    break
                except Exception:
                    time.sleep(0.05)
            if not health:
                raise AssertionError("source-built server did not become healthy")
            if health.get("version") != VERSION:
                raise AssertionError(f"health version mismatch: {health}")
            passed("health-version-source-binding")

            with urllib.request.urlopen(base + "/", timeout=5) as response:
                html = response.read().decode("utf-8")
            if f'<span class="version">{VERSION}</span>' not in html:
                raise AssertionError("served UI version does not match source version")
            passed("served-ui-version-source-binding")

            request_json(base + "/api/source/add", "POST", {"path": str(music)})
            deadline = time.time() + 15
            library = None
            while time.time() < deadline:
                library = request_json(base + "/api/library")
                if library.get("tracks") and not library.get("scan", {}).get("running"):
                    break
                time.sleep(0.05)
            tracks = (library or {}).get("tracks") or []
            if len(tracks) != 1:
                raise AssertionError(f"real library scan did not discover one fixture track: {library}")
            track = tracks[0]
            passed("real-library-scan-discovers-mp3")

            media_url = base + "/media/" + track["id"]
            with urllib.request.urlopen(media_url, timeout=5) as response:
                body = response.read()
                headers = {k.lower(): v for k, v in response.headers.items()}
                status = response.status
            if status != 200 or body != fixture_bytes:
                raise AssertionError("full media response did not equal fixture MP3 bytes")
            if headers.get("accept-ranges", "").lower() != "bytes":
                raise AssertionError(f"Accept-Ranges missing: {headers}")
            passed("full-media-response-exact-bytes")
            passed("media-advertises-byte-ranges")

            req = urllib.request.Request(media_url, headers={"Range": "bytes=0-127"})
            with urllib.request.urlopen(req, timeout=5) as response:
                ranged = response.read()
                content_range = response.headers.get("Content-Range", "")
                status = response.status
            if status != 206 or ranged != fixture_bytes[:128]:
                raise AssertionError("range media response did not equal requested fixture prefix")
            if not content_range.startswith("bytes 0-127/"):
                raise AssertionError(f"unexpected Content-Range: {content_range}")
            passed("media-range-206-exact-prefix")
        finally:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill(); process.wait(timeout=5)

    report = {
        "schemaVersion": "vexstream.real-media-server-regression/v1",
        "appVersion": VERSION,
        "fixtureSha256": FIXTURE_SHA256,
        "checks": checks,
        "status": "PASS",
        "doesNotProve": ["real browser audio decode", "real audio-device output", "Windows host acceptance"],
    }
    report_path = args.report
    if report_path:
        report_path.parent.mkdir(parents=True, exist_ok=True)
        report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
