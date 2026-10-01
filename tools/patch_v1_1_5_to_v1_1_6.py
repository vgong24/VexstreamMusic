#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
from pathlib import Path

SOURCE_SHA256 = "e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f"
SOURCE_HTML_SHA256 = "fadc33d2b16d3a9582cc84bcb701395876dfa7cc20d48e0d7b0ed7a339ab65fd"
OLD_VERSION = b"1.1.5"
NEW_VERSION = b"1.1.6"
HTML_START = b'<html lang="en">'
HTML_END = b'</html>'
OLD_PLAY_LIBRARY = (
    "function playLibraryTrack(id){const context=currentView==='home'?recentTracks(8):sortedSongTracks(),"
    "i=context.findIndex(t=>t.id===id);if(i>=0){setQueue(context,i,true);return}playTrack(id)}"
)
NEW_PLAY_LIBRARY = (
    "function playLibraryTrack(id){let c=currentView==='home'?recentTracks(8):currentView==='songs'?"
    "sortTrackList(state.tracks):sortedSongTracks(),i=c.findIndex(t=>t.id===id);if(i>=0){if(currentView==='songs'"
    "&&state.shuffle)c=[c[i],...c.slice(0,i),...c.slice(i+1)],i=0;setQueue(c,i,true);return}playTrack(id)}"
)
OLD_TOGGLE_PLAYBACK = "function togglePlayback(){if(!audio.paused){audio.pause();return}const t=currentTrack();if(!t){const context=sortedSongTracks();if(context.length)setQueue(context,0,true);return}if(audio.dataset.trackId!==t.id||!audio.currentSrc){playIndex(state.queueIndex);return}audio.play().catch(()=>{})}"
NEW_TOGGLE_PLAYBACK = "function togglePlayback(){if(!audio.paused){audio.pause();return}const t=currentTrack();if(!t){const c=currentView==='songs'?sortTrackList(state.tracks):sortedSongTracks();if(c.length)setQueue(c,0,true);return}if(audio.dataset.trackId!==t.id||!audio.currentSrc){playIndex(state.queueIndex);return}audio.play().catch(()=>{})}"
OLD_SHUFFLE_ALL = "bindClick('shuffleAllBtn',()=>setQueue(shuf(sortedSongTracks()),0,true));"
NEW_SHUFFLE_ALL = "bindClick('shuffleAllBtn',()=>setQueue(shuf(sortTrackList(state.tracks)),0,true));"
OLD_PLAY_ALL = "bindClick('playAllBtn',()=>setQueue(sortedSongTracks(),0,true));"
NEW_PLAY_ALL = "bindClick('playAllBtn',()=>setQueue(sortTrackList(state.tracks),0,true));"
OLD_UI_VERSION = '<span class="version">1.1.5</span>'
NEW_UI_VERSION = '<span class="version">1.1.6</span>'

def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()

def extract_html(data: bytes) -> tuple[int, int, bytes]:
    if data.count(HTML_START) != 1 or data.count(HTML_END) != 1:
        raise SystemExit("expected exactly one embedded HTML document")
    start = data.index(HTML_START)
    end = data.index(HTML_END, start) + len(HTML_END)
    return start, end, data[start:end]

def trim_inert_leading_whitespace(html: str, bytes_to_remove: int) -> str:
    head, marker, tail = html.partition("<script>")
    if not marker:
        raise SystemExit("embedded UI is missing expected <script> marker")
    lines = head.splitlines(keepends=True)
    remaining = bytes_to_remove
    out: list[str] = []
    for line in lines:
        if remaining and line.startswith(" "):
            removable = len(line) - len(line.lstrip(" "))
            take = min(removable, remaining)
            line = line[take:]
            remaining -= take
        out.append(line)
    if remaining:
        raise SystemExit(f"not enough inert pre-script indentation to remove {bytes_to_remove} bytes")
    return "".join(out) + marker + tail

def patch_html(html_bytes: bytes) -> bytes:
    if sha256(html_bytes) != SOURCE_HTML_SHA256:
        raise SystemExit("embedded HTML digest does not match the qualified 1.1.5 baseline")
    html = html_bytes.decode("utf-8")
    replacements = [
        (OLD_PLAY_LIBRARY, NEW_PLAY_LIBRARY, "playLibraryTrack"),
        (OLD_TOGGLE_PLAYBACK, NEW_TOGGLE_PLAYBACK, "togglePlayback"),
        (OLD_SHUFFLE_ALL, NEW_SHUFFLE_ALL, "Shuffle all binding"),
        (OLD_PLAY_ALL, NEW_PLAY_ALL, "Play all binding"),
        (OLD_UI_VERSION, NEW_UI_VERSION, "UI version badge"),
    ]
    patched = html
    for old, new, label in replacements:
        if patched.count(old) != 1:
            raise SystemExit(f"{label} baseline not found exactly once")
        patched = patched.replace(old, new, 1)

    delta = len(patched.encode("utf-8")) - len(html_bytes)
    if delta < 0:
        patched = patched.replace("</html>", (" " * -delta) + "</html>", 1)
    elif delta > 0:
        patched = trim_inert_leading_whitespace(patched, delta)

    out = patched.encode("utf-8")
    if len(out) != len(html_bytes):
        raise SystemExit(f"embedded HTML length changed: {len(html_bytes)} -> {len(out)}")
    for old, new, label in replacements[:-1]:
        if old.encode() in out:
            raise SystemExit(f"old {label} logic survived patch")
        if new.encode() not in out:
            raise SystemExit(f"new {label} logic missing after patch")
    return out


def patch_exe(source: Path, output: Path) -> tuple[str, str]:
    data = source.read_bytes()
    source_digest = sha256(data)
    if source_digest != SOURCE_SHA256:
        raise SystemExit(f"source SHA-256 mismatch: expected {SOURCE_SHA256}, got {source_digest}")
    if data.count(OLD_VERSION) != 3:
        raise SystemExit(f"expected exactly three 1.1.5 version markers; found {data.count(OLD_VERSION)}")
    start, end, html = extract_html(data)
    patched_html = patch_html(html)
    out = data[:start] + patched_html + data[end:]
    if out.count(OLD_VERSION) != 2:
        raise SystemExit(f"expected two non-UI 1.1.5 markers after UI patch; found {out.count(OLD_VERSION)}")
    out = out.replace(OLD_VERSION, NEW_VERSION)
    if len(out) != len(data):
        raise SystemExit("binary byte length changed")
    if not out.startswith(b"MZ"):
        raise SystemExit("output is not a PE image")
    if out.count(OLD_VERSION) != 0 or out.count(NEW_VERSION) < 3:
        raise SystemExit("version marker patch incomplete")
    output.write_bytes(out)
    return source_digest, sha256(out)

def main() -> int:
    parser = argparse.ArgumentParser(description="Deterministically patch qualified VexStream Music 1.1.5 to 1.1.6 UI behavior.")
    parser.add_argument("source", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    src, dst = patch_exe(args.source, args.output)
    print(f"source_sha256={src}")
    print(f"output_sha256={dst}")
    print(f"output={args.output}")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
