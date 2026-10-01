#!/usr/bin/env python3
"""Build the exact one-ZIP VexStream Music multi-platform distribution."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import zipfile

VERSION = "2.1.0"
MASTER_NAME = f"VexStreamMusic-{VERSION}-All-Platforms.zip"


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as fh:
        for chunk in iter(lambda: fh.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def run(command: list[str], cwd: Path) -> dict:
    proc = subprocess.run(command, cwd=cwd, capture_output=True, text=True)
    if proc.returncode != 0:
        raise RuntimeError(f"command failed: {command}\n{proc.stdout}\n{proc.stderr}")
    return {"command": command, "exitCode": proc.returncode, "stdout": proc.stdout, "stderr": proc.stderr, "status": "PASS"}


def mac_guide(display: str, arch: str, name: str, expected: str) -> str:
    return f'''# VexStream Music {VERSION} — {display}

Copy-paste this script in Terminal and press Return. The script will find this exact version of the VexStream Music executable for you and run it. You can move the folder contents wherever you choose; keep this file safe and reuse the same script whenever you want to launch VexStream Music.

```bash
bash <<'VEXSTREAM'
set -eu

NAME='{name}'
EXPECTED='{expected}'
EXPECTED_ARCH='{arch}'
CACHE="$HOME/Library/Application Support/VexStreamMusic/launcher-{VERSION}-{arch}.path"

if [ "$(uname -m)" != "$EXPECTED_ARCH" ]; then
  printf 'This launcher is for {display}. Detected: %s\\n' "$(uname -m)" >&2
  exit 2
fi

valid_binary() {{
  [ -f "$1" ] || return 1
  observed="$(/usr/bin/shasum -a 256 "$1" 2>/dev/null | /usr/bin/awk '{{print $1}}')"
  [ "$observed" = "$EXPECTED" ]
}}

P=''
if [ -f "$CACHE" ]; then
  cached="$(/bin/cat "$CACHE" 2>/dev/null || true)"
  if [ -n "$cached" ] && valid_binary "$cached"; then P="$cached"; fi
fi

if [ -z "$P" ] && command -v mdfind >/dev/null 2>&1; then
  while IFS= read -r candidate; do
    if [ -n "$candidate" ] && valid_binary "$candidate"; then P="$candidate"; break; fi
  done < <(/usr/bin/mdfind "kMDItemFSName == '$NAME'c" 2>/dev/null || true)
fi

if [ -z "$P" ]; then
  for root in "$HOME" /Applications /Volumes; do
    [ -e "$root" ] || continue
    while IFS= read -r -d '' candidate; do
      if valid_binary "$candidate"; then P="$candidate"; break 2; fi
    done < <(/usr/bin/find "$root" -type f -name "$NAME" -print0 2>/dev/null || true)
  done
fi

if [ -z "$P" ]; then
  printf 'VexStream Music {VERSION} was not found in the usual locations; checking the rest of this Mac...\\n'
  while IFS= read -r -d '' candidate; do
    if valid_binary "$candidate"; then P="$candidate"; break; fi
  done < <(/usr/bin/find / -type f -name "$NAME" -print0 2>/dev/null || true)
fi

if [ -z "$P" ]; then
  printf 'Could not find the verified VexStream Music {VERSION} executable named:\\n  %s\\n' "$NAME" >&2
  exit 3
fi

/bin/mkdir -p "$(/usr/bin/dirname "$CACHE")"
printf '%s' "$P" > "$CACHE"
/bin/chmod u+x "$P"
/usr/bin/xattr -d com.apple.quarantine "$P" 2>/dev/null || true

printf 'Launching verified VexStream Music {VERSION}:\\n  %s\\n' "$P"
exec "$P"
VEXSTREAM
```
'''


def zip_tree(staging: Path, output: Path) -> None:
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as zf:
        for path in sorted(staging.rglob("*")):
            arc = path.relative_to(staging).as_posix()
            if path.is_dir():
                info = zipfile.ZipInfo(arc.rstrip("/") + "/", date_time=(1980, 1, 1, 0, 0, 0))
                info.create_system = 3
                info.external_attr = (0o40755 << 16) | 0x10
                zf.writestr(info, b"")
            else:
                info = zipfile.ZipInfo(arc, date_time=(1980, 1, 1, 0, 0, 0))
                info.create_system = 3
                info.compress_type = zipfile.ZIP_DEFLATED
                mode = 0o100755 if path.name.startswith("VexStreamMusic-") and path.suffix != ".exe" else 0o100644
                info.external_attr = mode << 16
                zf.writestr(info, path.read_bytes(), compress_type=zipfile.ZIP_DEFLATED, compresslevel=9)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--output-dir", type=Path, required=True)
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    root = args.root.resolve()
    out = args.output_dir.resolve()
    out.mkdir(parents=True, exist_ok=True)
    build = out / "build"
    if build.exists():
        shutil.rmtree(build)
    build.mkdir()

    windows = build / f"VexStreamMusic-{VERSION}.exe"
    win_report = build / "windows.json"
    mac_dir = build / "mac"
    mac_report = build / "macos.json"
    steps = [
        run([sys.executable, str(root / "tests" / "build_windows.py"), "--root", str(root), "--output", str(windows), "--report", str(win_report)], root),
        run([sys.executable, str(root / "tests" / "build_macos.py"), "--root", str(root), "--output-dir", str(mac_dir), "--report", str(mac_report)], root),
    ]
    mac = json.loads(mac_report.read_text(encoding="utf-8"))
    by_arch = {row["architecture"]: row for row in mac["artifacts"]}

    staging = out / f"VexStreamMusic-{VERSION}-All-Platforms"
    if staging.exists():
        shutil.rmtree(staging)
    (staging / "Windows").mkdir(parents=True)
    (staging / "Mac Silicon").mkdir(parents=True)
    (staging / "Mac Intel").mkdir(parents=True)
    shutil.copy2(windows, staging / "Windows" / windows.name)

    mac_specs = [
        ("arm64", "Mac Silicon", "Apple Silicon Macs"),
        ("amd64", "Mac Intel", "Intel Macs"),
    ]
    artifacts = {"windows": {"path": f"Windows/{windows.name}", "sha256": sha256(windows)}}
    for arch, folder, display in mac_specs:
        src = Path(by_arch[arch]["binary"])
        dst = staging / folder / src.name
        shutil.copy2(src, dst)
        dst.chmod(dst.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
        digest = sha256(dst)
        guide = mac_guide(display, arch, dst.name, digest)
        (staging / folder / "COPY-PASTE-TO-TERMINAL.md").write_text(guide, encoding="utf-8")
        artifacts[arch] = {"path": f"{folder}/{dst.name}", "sha256": digest}

    master = out / MASTER_NAME
    if master.exists():
        master.unlink()
    zip_tree(staging, master)
    with zipfile.ZipFile(master) as zf:
        files = sorted(n for n in zf.namelist() if not n.endswith("/"))
        expected = sorted([
            f"Windows/VexStreamMusic-{VERSION}.exe",
            f"Mac Silicon/VexStreamMusic-{VERSION}-macOS-Apple-Silicon",
            "Mac Silicon/COPY-PASTE-TO-TERMINAL.md",
            f"Mac Intel/VexStreamMusic-{VERSION}-macOS-Intel",
            "Mac Intel/COPY-PASTE-TO-TERMINAL.md",
        ])
        if files != expected:
            raise RuntimeError(f"master ZIP member set mismatch: {files}")
        for member in [f"Mac Silicon/VexStreamMusic-{VERSION}-macOS-Apple-Silicon", f"Mac Intel/VexStreamMusic-{VERSION}-macOS-Intel"]:
            if ((zf.getinfo(member).external_attr >> 16) & 0o111) == 0:
                raise RuntimeError(f"master ZIP lost Mac executable permission: {member}")

    report = {
        "schemaVersion": "vexstream.all-platform-distribution/v1",
        "appVersion": VERSION,
        "sourceRoot": str(root),
        "masterZip": str(master),
        "masterZipSha256": sha256(master),
        "memberCount": 5,
        "artifacts": artifacts,
        "steps": steps,
        "status": "PASS",
    }
    report_path = args.report or (out / "all-platforms.json")
    report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
