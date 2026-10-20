#!/usr/bin/env python3
from __future__ import annotations

import argparse
import base64
import copy
import json
import os
import re
import shutil
from pathlib import Path
from urllib.parse import urlparse

from playwright.sync_api import Route, sync_playwright

TRACKS = [
    {
        "id": "media.vexstream.test-tone.1",
        "title": "VexStream Test Tone One",
        "artist": "VexStream QA",
        "album": "V4 Practicum",
        "genre": "Test",
        "year": "2026",
        "duration": 1.25,
        "addedAt": "2026-10-12T00:00:00Z",
        "tags": ["qa", "tone-one"],
        "channel": "VexStream QA Channel",
        "playbackStart": 0,
    },
    {
        "id": "media.vexstream.test-tone.2",
        "title": "VexStream Test Tone Two",
        "artist": "VexStream QA",
        "album": "V4 Practicum",
        "genre": "Test",
        "year": "2026",
        "duration": 2.5,
        "addedAt": "2026-10-14T00:00:00Z",
        "tags": ["qa", "tone-two"],
        "channel": "VexStream QA Channel",
        "playbackStart": 0,
    },
    {
        "id": "media.vexstream.test-tone.3",
        "title": "VexStream Test Tone Three",
        "artist": "Other Artist",
        "album": "Other Album",
        "genre": "Ambient",
        "year": "2025",
        "duration": 3.75,
        "addedAt": "2026-10-13T00:00:00Z",
        "tags": ["ambient", "tone-three"],
        "channel": "Other Channel",
        "playbackStart": 0,
    },
]

PLAYLISTS = [
    {
        "id": "road-trip",
        "name": "Road Trip",
        "trackIds": [
            "media.vexstream.test-tone.1",
            "media.vexstream.missing",
            "media.vexstream.test-tone.3",
        ],
        "createdAt": "2026-10-14T00:00:00Z",
        "updatedAt": "2026-10-14T00:00:00Z",
    }
]

EXPLORE_FRAMES = {
    "track": {
        "schemaVersion": "vexstream.discovery-candidates/v2",
        "seed": {
            "kind": "library",
            "trackId": "media.vexstream.test-tone.1",
            "title": "VexStream Test Tone One",
            "artist": "VexStream QA",
        },
        "basis": {
            "provider": "test-provider",
            "method": "SEARCH_DERIVED_MULTI_AXIS",
            "queries": [],
            "creatorCap": 2,
            "selection": "ROUND_ROBIN",
        },
        "candidates": [
            {
                "providerId": "candidate.one",
                "url": "https://www.youtube.com/watch?v=candidate.one",
                "title": "Candidate One",
                "channel": "Creator One",
                "axis": "CLOSER",
                "axisLabel": "Closer",
                "libraryMatch": {"state": "NOT_IN_LIBRARY"},
            },
            {
                "providerId": "candidate.two",
                "url": "https://www.youtube.com/watch?v=candidate.two",
                "title": "Candidate Two",
                "channel": "Creator Two",
                "axis": "VERSIONS",
                "axisLabel": "Versions",
                "libraryMatch": {
                    "state": "IN_LIBRARY",
                    "trackId": "media.vexstream.test-tone.2",
                },
            },
        ],
        "generatedAt": "2026-10-14T00:00:00Z",
    },
    "branch": {
        "schemaVersion": "vexstream.discovery-candidates/v2",
        "seed": {
            "kind": "provider",
            "providerId": "candidate.one",
            "title": "Candidate One",
            "channel": "Creator One",
        },
        "basis": {
            "provider": "test-provider",
            "method": "SEARCH_DERIVED_MULTI_AXIS",
            "queries": [],
            "creatorCap": 2,
            "selection": "ROUND_ROBIN",
        },
        "candidates": [
            {
                "providerId": "candidate.branch",
                "url": "https://www.youtube.com/watch?v=candidate.branch",
                "title": "Branch Candidate",
                "channel": "Branch Creator",
                "axis": "NEIGHBORHOOD",
                "axisLabel": "Neighborhood",
                "libraryMatch": {"state": "NOT_IN_LIBRARY"},
            },
        ],
        "generatedAt": "2026-10-14T00:00:01Z",
    },
}

IMPORT_REVIEW = {
    "inspection": {
        "id": "candidate.one",
        "url": "https://www.youtube.com/watch?v=candidate.one",
        "title": "Candidate One",
        "channel": "Creator One",
        "duration": 120,
        "artist": "",
        "album": "Candidate Collection",
        "genre": "Test",
        "hasSections": True,
        "sections": [
            {
                "index": 1,
                "start": 0,
                "end": 60,
                "title": "Candidate Part One",
                "artist": "",
                "selected": True,
            },
            {
                "index": 2,
                "start": 60,
                "end": 120,
                "title": "Candidate Part Two",
                "artist": "",
                "selected": True,
            },
        ],
        "sourceQuality": {
            "codec": "opus",
            "bitrateKbps": 160,
        },
    },
    "items": [
        {
            "kind": "whole",
            "title": "Candidate One",
            "duration": 120,
            "matches": [
                {
                    "trackId": "media.vexstream.test-tone.2",
                    "title": "VexStream Test Tone Two",
                    "artist": "VexStream QA",
                    "score": 78,
                    "confidence": "likely",
                    "relation": "SAME_SONG_POSSIBLE",
                    "reasons": ["similar title"],
                    "localQuality": {},
                    "qualityComparison": {
                        "state": "INSUFFICIENT_DATA",
                        "summary": "Not enough format data.",
                    },
                }
            ],
        }
    ],
    "matchCount": 1,
    "advisory": True,
}


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def browser_executable(explicit: str | None) -> str | None:
    if explicit:
        return explicit
    if os.environ.get("VEXSTREAM_CHROMIUM"):
        return os.environ["VEXSTREAM_CHROMIUM"]
    for name in ("chromium", "chromium-browser", "google-chrome", "chrome"):
        found = shutil.which(name)
        if found:
            return found
    return None


def as_data_url(mime: str, data: bytes) -> str:
    return f"data:{mime};base64,{base64.b64encode(data).decode('ascii')}"


def bundle_module(path: Path, cache: dict[Path, str]) -> str:
    path = path.resolve()
    if path in cache:
        return cache[path]
    source = path.read_text(encoding="utf-8")
    pattern = re.compile(
        r'(?P<prefix>\bfrom\s+|\bimport\s+)(?P<quote>["\'])(?P<spec>\.[^"\']+)(?P=quote)'
    )

    def replace(match: re.Match[str]) -> str:
        target = (path.parent / match.group("spec")).resolve()
        url = bundle_module(target, cache)
        return f'{match.group("prefix")}{match.group("quote")}{url}{match.group("quote")}'

    source = pattern.sub(replace, source)
    url = as_data_url("text/javascript;charset=utf-8", source.encode("utf-8"))
    cache[path] = url
    return url


def assembled_html(root: Path) -> str:
    web = root / "v2" / "web"
    html = (web / "index.html").read_text(encoding="utf-8")
    css = (web / "styles.css").read_text(encoding="utf-8")
    app = bundle_module(web / "app.js", {})
    html = html.replace(
        "<head>",
        '<head><base href="http://vexstream.local/">',
        1,
    )
    html = html.replace(
        '<link rel="stylesheet" href="./styles.css">',
        f"<style>{css}</style>",
    )
    inject = (
        "<script>"
        "window.__VEXSTREAM_V4_SHUFFLE=(values)=>[...values].reverse();"
        "</script>"
    )
    html = html.replace(
        '<script type="module" src="./app.js"></script>',
        inject + f'<script type="module">import {json.dumps(app)};</script>',
    )
    return html


def media_response(route: Route, fixture: bytes) -> None:
    request = route.request
    headers = request.headers
    range_header = headers.get("range", "")
    response_headers = {
        "Accept-Ranges": "bytes",
        "Content-Type": "audio/mpeg",
        "X-Content-Type-Options": "nosniff",
    }

    if request.method == "HEAD":
        response_headers["Content-Length"] = str(len(fixture))
        route.fulfill(status=200, headers=response_headers, body=b"")
        return

    if range_header:
        match = re.fullmatch(r"bytes=(\d+)-(\d*)", range_header.strip())
        if match:
            start = int(match.group(1))
            end = int(match.group(2)) if match.group(2) else len(fixture) - 1
            end = min(end, len(fixture) - 1)
            if start <= end and start < len(fixture):
                body = fixture[start : end + 1]
                response_headers["Content-Range"] = (
                    f"bytes {start}-{end}/{len(fixture)}"
                )
                response_headers["Content-Length"] = str(len(body))
                route.fulfill(
                    status=206,
                    headers=response_headers,
                    body=body,
                )
                return

    response_headers["Content-Length"] = str(len(fixture))
    route.fulfill(
        status=200,
        headers=response_headers,
        body=fixture,
    )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--root",
        type=Path,
        default=Path(__file__).resolve().parents[2],
    )
    parser.add_argument("--browser-executable")
    args = parser.parse_args()
    root = args.root.resolve()
    fixture_path = root / "tests" / "fixtures" / "vexstream-test-tone.mp3"
    require(fixture_path.is_file(), "media fixture missing")
    fixture = fixture_path.read_bytes()
    html = assembled_html(root)

    report = {
        "schemaVersion": "vexstream.v4.player-browser-regression/v2",
        "checks": [],
        "pageErrors": [],
        "console": [],
        "requests": [],
    }

    def checked(name, fn) -> None:
        fn()
        report["checks"].append({"name": name, "status": "PASS"})

    source_state = {
        "sources": ["/music"],
        "ignored": [],
        "chosen": "/picked",
        "candidates": [
            {"name": "Candidate Music", "path": "/candidate", "mp3Count": 7},
            {"name": "Ignore Me", "path": "/ignore-me", "mp3Count": 2},
        ],
    }
    import_job_state = {
        "started": False,
        "startBody": None,
    }
    system_state = {
        "exitBody": None,
    }
    library_admin_state = {
        "overlays": {},
        "windows": {},
    }
    trash_state = {
        "trashedRefs": set(),
        "trash": [],
        "deletedHistory": [],
        "pendingPurge": None,
    }
    playlist_state = {
        "rows": copy.deepcopy(PLAYLISTS),
        "next": 1,
    }

    def projected_tracks():
        if "/music" not in source_state["sources"]:
            return []
        rows = []
        for source in TRACKS:
            if source["id"] in trash_state["trashedRefs"]:
                continue
            row = copy.deepcopy(source)
            row["sourceTitle"] = source.get("title", "")
            row["sourceArtist"] = source.get("artist", "")
            row["sourceAlbum"] = source.get("album", "")
            row["sourceGenre"] = source.get("genre", "")
            overlay = library_admin_state["overlays"].get(source["id"])
            if overlay:
                row.update(overlay)
                row["overlayUpdatedAt"] = "2026-10-14T23:00:00Z"
            window = library_admin_state["windows"].get(source["id"])
            if window:
                row["playbackStart"] = window.get("start", 0)
                if window.get("end") is not None:
                    row["playbackEnd"] = window["end"]
                row["playbackUpdatedAt"] = "2026-10-14T23:00:01Z"
            rows.append(row)
        return rows

    player_session_state = {
        "found": False,
        "session": {
            "schemaVersion": "vexstream.desktop-player-session/v1",
            "items": [],
            "currentIndex": -1,
            "currentRef": "",
            "shuffle": False,
            "repeat": False,
            "updatedAt": "",
        },
        "writes": [],
    }

    def route_request(route: Route) -> None:
        request = route.request
        parsed = urlparse(request.url)
        report["requests"].append(
            {
                "method": request.method,
                "path": parsed.path,
                "range": request.headers.get("range"),
            }
        )

        if parsed.path == "/":
            route.fulfill(
                status=200,
                content_type="text/html; charset=utf-8",
                body=html,
            )
            return

        if parsed.path == "/health":
            if request.method != "GET":
                route.fulfill(status=405, body="GET required")
                return
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(
                    {
                        "ok": True,
                        "version": "4.0.0",
                        "buildRef": "browser-proof.bin-deadbeef",
                        "warnings": [],
                    }
                ),
            )
            return

        if parsed.path == "/api/system/exit":
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            body = json.loads(request.post_data or "{}")
            system_state["exitBody"] = body
            if body != {"confirm": "EXIT_VEXSTREAM"}:
                route.fulfill(status=400, body="confirmation required")
                return
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps({"ok": True, "state": "EXITING"}),
            )
            return

        if parsed.path == "/api/library":
            if request.method != "GET":
                route.fulfill(status=405, body="GET required")
                return
            route.fulfill(
                status=200,
                headers={
                    "Access-Control-Allow-Origin": "*",
                    "Content-Type": "application/json; charset=utf-8",
                    "X-Content-Type-Options": "nosniff",
                },
                body=json.dumps(
                    {
                        "tracks": projected_tracks()
                    }
                ),
            )
            return

        if parsed.path == "/api/source/config":
            if request.method != "GET":
                route.fulfill(status=405, body="GET required")
                return
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(
                    {
                        "sources": source_state["sources"],
                        "ignored": source_state["ignored"],
                    }
                ),
            )
            return

        if parsed.path == "/api/source/choose":
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps({"path": source_state["chosen"]}),
            )
            return

        if parsed.path in {
            "/api/source/add",
            "/api/source/remove",
            "/api/candidate/ignore",
            "/api/candidate/unignore",
        }:
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            body = json.loads(request.post_data or "{}")
            path = str(body.get("path") or "")
            if not path:
                route.fulfill(status=400, body="path required")
                return
            if parsed.path == "/api/source/add":
                if path not in source_state["sources"]:
                    source_state["sources"].append(path)
                source_state["ignored"] = [
                    value
                    for value in source_state["ignored"]
                    if value != path
                ]
            elif parsed.path == "/api/source/remove":
                source_state["sources"] = [
                    value
                    for value in source_state["sources"]
                    if value != path
                ]
            elif parsed.path == "/api/candidate/ignore":
                if path not in source_state["ignored"]:
                    source_state["ignored"].append(path)
            else:
                source_state["ignored"] = [
                    value
                    for value in source_state["ignored"]
                    if value != path
                ]
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(
                    {
                        "ok": True,
                        "config": {
                            "sources": source_state["sources"],
                            "ignored": source_state["ignored"],
                        },
                        "libraryRefresh": (
                            "CURRENT"
                            if parsed.path
                            in {"/api/source/add", "/api/source/remove"}
                            else ""
                        ),
                    }
                ),
            )
            return

        if parsed.path == "/api/ignored/clear":
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            body = json.loads(request.post_data or "{}")
            if body != {"confirm": "CLEAR_IGNORED_FOLDERS"}:
                route.fulfill(status=400, body="confirmation required")
                return
            source_state["ignored"] = []
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(
                    {
                        "ok": True,
                        "config": {
                            "sources": source_state["sources"],
                            "ignored": [],
                        },
                    }
                ),
            )
            return

        if parsed.path == "/api/discover":
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            body = json.loads(request.post_data or "{}")
            candidates = [
                candidate
                for candidate in source_state["candidates"]
                if candidate["path"] not in source_state["sources"]
                and candidate["path"] not in source_state["ignored"]
            ]
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(
                    {
                        "candidates": candidates,
                        "scannedDirs": 42 if body.get("deep") else 12,
                    }
                ),
            )
            return

        if parsed.path == "/api/playlists":
            if request.method != "GET":
                route.fulfill(status=405, body="GET required")
                return
            route.fulfill(
                status=200,
                headers={
                    "Access-Control-Allow-Origin": "*",
                    "Content-Type": "application/json; charset=utf-8",
                    "X-Content-Type-Options": "nosniff",
                },
                body=json.dumps({"playlists": playlist_state["rows"]}),
            )
            return

        if parsed.path.startswith("/api/playlists/"):
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            body = json.loads(request.post_data or "{}")
            action = parsed.path.rsplit("/", 1)[-1]
            rows = playlist_state["rows"]

            if action == "create":
                name = str(body.get("name") or "").strip()
                if not name:
                    route.fulfill(status=400, body="name required")
                    return
                row = {
                    "id": f"playlist-new-{playlist_state['next']}",
                    "name": name,
                    "trackIds": list(dict.fromkeys(body.get("trackIds") or [])),
                    "createdAt": "2026-10-14T23:00:10Z",
                    "updatedAt": "2026-10-14T23:00:10Z",
                }
                playlist_state["next"] += 1
                rows.append(row)
                payload = {"ok": True, "playlist": row}
            else:
                playlist_id = str(body.get("id") or "")
                row = next(
                    (value for value in rows if value["id"] == playlist_id),
                    None,
                )
                if row is None:
                    route.fulfill(status=404, body="playlist not found")
                    return
                if action == "add":
                    before = len(row["trackIds"])
                    for ref in body.get("trackIds") or []:
                        if ref not in row["trackIds"]:
                            row["trackIds"].append(ref)
                    payload = {
                        "ok": True,
                        "added": len(row["trackIds"]) - before,
                        "playlist": row,
                    }
                elif action == "order":
                    wanted = list(dict.fromkeys(body.get("trackIds") or []))
                    if set(wanted) != set(row["trackIds"]):
                        route.fulfill(status=409, body="order mismatch")
                        return
                    row["trackIds"] = wanted
                    payload = {"ok": True}
                elif action == "remove":
                    remove = set(body.get("trackIds") or [])
                    row["trackIds"] = [
                        ref for ref in row["trackIds"] if ref not in remove
                    ]
                    payload = {"ok": True, "playlist": row}
                elif action == "rename":
                    row["name"] = str(body.get("name") or "").strip()
                    payload = {"ok": True, "playlist": row}
                elif action == "delete":
                    if body.get("confirm") != "DELETE_PLAYLIST_ONLY":
                        route.fulfill(status=400, body="confirmation required")
                        return
                    playlist_state["rows"] = [
                        value for value in rows if value["id"] != playlist_id
                    ]
                    payload = {"ok": True}
                else:
                    route.fulfill(status=404, body="not found")
                    return
                if action != "delete":
                    row["updatedAt"] = "2026-10-14T23:00:11Z"

            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(payload),
            )
            return

        if parsed.path == "/api/library/overlay":
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            body = json.loads(request.post_data or "{}")
            ids = list(body.get("ids") or [])
            if body.get("id"):
                ids.append(body["id"])
            for ref in ids:
                if body.get("restore"):
                    library_admin_state["overlays"].pop(ref, None)
                    continue
                patch = {}
                for key in ("title", "artist", "album", "genre"):
                    if key in body:
                        patch[key] = body[key]
                library_admin_state["overlays"][ref] = patch
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps({"ok": True, "updated": len(ids)}),
            )
            return

        if parsed.path == "/api/library/playback-window":
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            body = json.loads(request.post_data or "{}")
            ref = str(body.get("id") or "")
            if body.get("restore"):
                library_admin_state["windows"].pop(ref, None)
                payload = {"ok": True, "restored": True}
            else:
                library_admin_state["windows"][ref] = {
                    "start": float(body.get("start") or 0),
                    "end": body.get("end"),
                }
                payload = {
                    "ok": True,
                    "window": library_admin_state["windows"][ref],
                }
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(payload),
            )
            return

        if parsed.path == "/api/library/trash":
            if request.method == "GET":
                route.fulfill(
                    status=200,
                    headers={"Content-Type": "application/json; charset=utf-8"},
                    body=json.dumps(
                        {
                            "trash": trash_state["trash"],
                            "deletedHistory": trash_state["deletedHistory"],
                            "pendingPurge": trash_state["pendingPurge"],
                        }
                    ),
                )
                return
            if request.method != "POST":
                route.fulfill(status=405, body="GET or POST required")
                return
            body = json.loads(request.post_data or "{}")
            if body.get("confirm") != "MOVE_TO_VEXSTREAM_TRASH":
                route.fulfill(status=400, body="confirmation required")
                return
            current = {row["id"]: row for row in projected_tracks()}
            moved = 0
            for ref in body.get("ids") or []:
                if ref not in current:
                    continue
                trash_id = "trash-" + ref
                trash_state["trashedRefs"].add(ref)
                trash_state["trash"].append(
                    {
                        "id": trash_id,
                        "track": current[ref],
                        "originalPath": "/music/" + ref + ".mp3",
                        "trashPath": "/trash/" + ref + ".mp3",
                        "trashedAt": "2026-10-14T23:00:20Z",
                    }
                )
                moved += 1
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(
                    {
                        "ok": True,
                        "moved": moved,
                        "libraryRefresh": "CURRENT",
                    }
                ),
            )
            return

        if parsed.path == "/api/library/trash/restore":
            body = json.loads(request.post_data or "{}")
            wanted = set(body.get("ids") or [])
            restored = [
                item for item in trash_state["trash"] if item["id"] in wanted
            ]
            trash_state["trash"] = [
                item for item in trash_state["trash"] if item["id"] not in wanted
            ]
            for item in restored:
                trash_state["trashedRefs"].discard(item["track"]["id"])
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(
                    {
                        "ok": True,
                        "restored": len(restored),
                        "libraryRefresh": "CURRENT",
                    }
                ),
            )
            return

        if parsed.path == "/api/library/trash/purge":
            body = json.loads(request.post_data or "{}")
            if body.get("confirm") != "PERMANENTLY_DELETE_TRASH":
                route.fulfill(status=400, body="confirmation required")
                return
            wanted = set(body.get("ids") or [])
            deleted = [
                item for item in trash_state["trash"] if item["id"] in wanted
            ]
            trash_state["trash"] = [
                item for item in trash_state["trash"] if item["id"] not in wanted
            ]
            for item in deleted:
                trash_state["deletedHistory"].append(
                    {
                        "id": "deleted-" + item["track"]["id"],
                        "track": item["track"],
                        "originalPath": item["originalPath"],
                        "deletedAt": "2026-10-14T23:00:21Z",
                    }
                )
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(
                    {
                        "ok": True,
                        "deleted": len(deleted),
                        "preservedMetadata": bool(body.get("preserveMetadata")),
                    }
                ),
            )
            return

        if parsed.path == "/api/library/trash/purge/recover":
            trash_state["pendingPurge"] = None
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps({"ok": True, "recovered": 0}),
            )
            return

        if parsed.path == "/api/discovery/radio":
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            body = json.loads(request.post_data or "{}")
            if body.get("trackId") == "media.vexstream.test-tone.1":
                payload = EXPLORE_FRAMES["track"]
            elif (body.get("seed") or {}).get("providerId") == "candidate.one":
                payload = EXPLORE_FRAMES["branch"]
            else:
                route.fulfill(status=400, body="unexpected Explore seed")
                return
            route.fulfill(
                status=200,
                headers={
                    "Access-Control-Allow-Origin": "*",
                    "Content-Type": "application/json; charset=utf-8",
                    "X-Content-Type-Options": "nosniff",
                },
                body=json.dumps(payload),
            )
            return

        if parsed.path == "/api/player/session":
            if request.method == "GET":
                route.fulfill(
                    status=200,
                    headers={"Content-Type": "application/json; charset=utf-8"},
                    body=json.dumps(
                        {
                            "found": player_session_state["found"],
                            "session": player_session_state["session"],
                        }
                    ),
                )
                return
            if request.method == "PUT":
                body = json.loads(request.post_data or "{}")
                session = {
                    "schemaVersion": "vexstream.desktop-player-session/v1",
                    "items": list(body.get("items") or []),
                    "currentIndex": int(body.get("currentIndex", -1)),
                    "currentRef": str(body.get("currentRef") or ""),
                    "shuffle": bool(body.get("shuffle")),
                    "repeat": bool(body.get("repeat")),
                    "updatedAt": "2026-10-14T22:45:00Z",
                }
                player_session_state["found"] = True
                player_session_state["session"] = session
                player_session_state["writes"].append(session.copy())
                route.fulfill(
                    status=200,
                    headers={"Content-Type": "application/json; charset=utf-8"},
                    body=json.dumps({"saved": True, "session": session}),
                )
                return
            route.fulfill(status=405, body="GET or PUT required")
            return

        if parsed.path == "/api/import/jobs":
            if request.method != "GET":
                route.fulfill(status=405, body="GET required")
                return
            active = []
            if import_job_state["started"]:
                active = [
                    {
                        "id": "job-import-1",
                        "title": "Candidate One",
                        "status": "queued",
                        "stage": "Waiting for preparation",
                        "progress": 0,
                        "detail": "Queued in reversible staging.",
                    }
                ]
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(
                    {
                        "groups": {
                            "active": active,
                            "ready": [],
                            "completed": [],
                        },
                        "recovery": {},
                        "issues": [],
                    }
                ),
            )
            return

        if parsed.path == "/api/import/destinations":
            if request.method != "GET":
                route.fulfill(status=405, body="GET required")
                return
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps({"destinations": ["/music"]}),
            )
            return

        if parsed.path == "/api/import/inspect":
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            body = json.loads(request.post_data or "{}")
            if body != {
                "url": "https://www.youtube.com/watch?v=candidate.one"
            }:
                route.fulfill(status=400, body="unexpected Import inspection")
                return
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(IMPORT_REVIEW),
            )
            return

        if parsed.path == "/api/import/start":
            if request.method != "POST":
                route.fulfill(status=405, body="POST required")
                return
            body = json.loads(request.post_data or "{}")
            if (
                body.get("url")
                != "https://www.youtube.com/watch?v=candidate.one"
                or body.get("destination") != "/music"
                or body.get("mode") != "split"
                or len(body.get("sections") or []) != 2
                or not all(
                    section.get("selected") is True
                    for section in body.get("sections") or []
                )
            ):
                route.fulfill(
                    status=400,
                    body="unexpected reviewed Import plan",
                )
                return
            import_job_state["started"] = True
            import_job_state["startBody"] = body
            route.fulfill(
                status=200,
                headers={"Content-Type": "application/json; charset=utf-8"},
                body=json.dumps(
                    {
                        "jobId": "job-import-1",
                        "status": "queued",
                    }
                ),
            )
            return

        if parsed.path.startswith("/media/"):
            media_response(route, fixture)
            return

        route.fulfill(status=404, body="not found")

    with sync_playwright() as p:
        launch = {
            "headless": True,
            "args": [
                "--no-sandbox",
                "--autoplay-policy=no-user-gesture-required",
            ],
        }
        executable = browser_executable(args.browser_executable)
        if executable:
            launch["executable_path"] = executable

        browser = p.chromium.launch(**launch)
        report["browserVersion"] = browser.version
        page = browser.new_page(viewport={"width": 1000, "height": 840})
        page.on(
            "pageerror",
            lambda error: report["pageErrors"].append(str(error)),
        )
        page.on(
            "console",
            lambda message: report["console"].append(
                {"type": message.type, "text": message.text}
            ),
        )
        page.route("http://vexstream.local/**", route_request)
        page.route(
            "https://www.youtube.com/**",
            lambda route: route.fulfill(
                status=200,
                content_type="text/html",
                body="<html><body>preview fixture</body></html>",
            ),
        )
        # The execution sandbox blocks HTTP top-level navigation by administrator policy.
        # Keep the real product-relative URLs, but host this proof in an in-memory document whose
        # <base> points at the intercepted VexStream origin.
        page.set_content(html, wait_until="load")
        page.wait_for_function("window.vexV4?.ready===true")

        checked(
            "real-browser-loads-v3-shaped-v4-shell",
            lambda: require(
                not report["pageErrors"],
                f"page errors: {report['pageErrors']}",
            ),
        )
        checked(
            "browser-binds-visible-exact-desktop-identity",
            lambda: require(
                page.evaluate("window.vexV4.identity.version") == "4.0.0"
                and page.evaluate("window.vexV4.identity.buildRef")
                == "browser-proof.bin-deadbeef"
                and "4.0.0" in page.locator("#buildIdentity").inner_text()
                and page.locator("#buildIdentity").get_attribute(
                    "data-build-ref"
                )
                == "browser-proof.bin-deadbeef",
                "visible version/build identity drift",
            ),
        )

        page.locator("#systemStatusButton").click()
        checked(
            "system-status-restores-recognizable-read-only-terrain",
            lambda: require(
                page.locator("#systemDrawer").is_visible()
                and page.evaluate("window.vexV4.systemStatus.state")
                == "healthy"
                and page.evaluate("window.vexV4.systemStatus.trackCount")
                == 3
                and page.evaluate("window.vexV4.systemStatus.sourceCount")
                == 1
                and page.locator("#systemStatusBadge").is_hidden()
                and "System looks healthy"
                in page.locator("#systemSummary").inner_text()
                and "3 tracks · 1 folders"
                in page.locator("#systemLibraryStatus").inner_text(),
                "System status projection drift",
            ),
        )
        checked(
            "system-status-exposes-unmigrated-diagnostics-boundary-without-fake-controls",
            lambda: require(
                "not yet owned by Desktop V4"
                in page.locator(".system-status-boundary").inner_text()
                and page.locator("#runDiagnosticsBtn").count() == 0
                and page.locator("#loadRepairPlanBtn").count() == 0,
                "System status invented unmigrated diagnostics or repair controls",
            ),
        )
        page.locator("#systemDrawerClose").click()
        checked(
            "system-status-closes-without-changing-product-location",
            lambda: require(
                page.locator("#systemDrawer").is_hidden()
                and page.evaluate(
                    "window.vexV4.shellNavigation.currentProjection.view"
                )
                == "home",
                "System drawer crossed shell navigation ownership",
            ),
        )

        checked(
            "browser-loads-read-only-library-contract",
            lambda: require(
                any(
                    row["method"] == "GET"
                    and row["path"] == "/api/library"
                    for row in report["requests"]
                )
                and page.evaluate("window.vexV4.catalog.length") == 3,
                f"library contract not consumed: {report['requests']}",
            ),
        )

        checked(
            "v3-shell-opens-on-library-home-over-real-v4-library-truth",
            lambda: require(
                page.evaluate(
                    "window.vexV4.shellNavigation.currentProjection.view"
                )
                == "home"
                and page.locator("#homeView").is_visible()
                and page.locator("#librarySidebar").is_visible()
                and page.locator("#homeSongCount").inner_text() == "3"
                and page.locator("#homeAlbumCount").inner_text() == "2"
                and page.locator("#homeArtistCount").inner_text() == "2"
                and page.locator("#homeSourceCount").inner_text() == "1"
                and page.locator(
                    "#recentSongs [data-home-track-ref]"
                ).count()
                == 3,
                "recognizable Library Home did not project current V4 truth",
            ),
        )

        page.locator("#libraryNavToggle").click()
        checked(
            "v3-sidebar-collapse-is-shell-presentation-state",
            lambda: require(
                page.locator("#librarySidebar").is_hidden()
                and page.evaluate(
                    "window.vexV4.shellNavigation.currentProjection.sidebarOpen"
                )
                is False,
                "sidebar collapse crossed shell ownership",
            ),
        )
        page.locator("#libraryNavToggle").click()

        page.locator(
            '#librarySidebar [data-shell-view="songs"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='songs'"
        )
        checked(
            "songs-terrain-enters-library-through-shell-navigation",
            lambda: require(
                page.locator("#trackChoices").is_visible()
                and page.evaluate(
                    "window.vexV4.libraryView.mode"
                )
                == "songs",
                "Songs terrain did not enter current Library projection",
            ),
        )
        checked(
            "dynamic-library-metadata-renders-as-data",
            lambda: require(
                "VexStream Test Tone One"
                in page.locator("#trackChoices").inner_text()
                and "VexStream QA"
                in page.locator("#trackChoices").inner_text(),
                "catalog metadata did not render",
            ),
        )
        page.locator("#selectSongs").click()
        page.locator(
            '[data-track-ref="media.vexstream.test-tone.1"] [data-track-select]'
        ).check()
        page.locator(
            '[data-track-ref="media.vexstream.test-tone.3"] [data-track-select]'
        ).check()
        checked(
            "songs-selection-is-ephemeral-experience-state-over-library-truth",
            lambda: require(
                page.evaluate(
                    "window.vexV4.librarySelection.currentProjection.selectedRefs.join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.test-tone.3"
                )
                and page.locator("#librarySelectionBar").is_visible()
                and page.locator("#librarySelectionCount").inner_text()
                == "2 selected"
                and page.evaluate("window.vexV4.catalog.length") == 3,
                "Songs selection drifted into Library truth",
            ),
        )
        page.locator("#selectVisibleSongs").click()
        checked(
            "select-visible-follows-the-current-songs-projection",
            lambda: require(
                page.evaluate(
                    "window.vexV4.librarySelection.currentProjection.count"
                )
                == 3,
                "Select visible did not consume current Library projection",
            ),
        )
        page.locator("#clearSongSelection").click()
        page.locator("#selectSongs").click()
        checked(
            "done-clears-and-closes-song-selection",
            lambda: require(
                page.evaluate(
                    "window.vexV4.librarySelection.currentProjection.active"
                )
                is False
                and page.evaluate(
                    "window.vexV4.librarySelection.currentProjection.count"
                )
                == 0
                and page.locator("#librarySelectionBar").is_hidden(),
                "Selection did not close cleanly",
            ),
        )

        checked(
            "source-config-loads-separately-from-library-catalog",
            lambda: require(
                page.evaluate("window.vexV4.sourceConfig.sources.join(',')")
                == "/music"
                and page.locator('[data-source-path="/music"]').count() == 1,
                "source configuration did not render independently",
            ),
        )

        page.locator("#librarySearch").fill("Three")
        checked(
            "library-search-is-a-view-projection-not-catalog-mutation",
            lambda: require(
                page.locator("#trackChoices [data-track-ref]").count() == 1
                and page.locator(
                    '[data-track-ref="media.vexstream.test-tone.3"]'
                ).count()
                == 1
                and page.evaluate("window.vexV4.catalog.length") == 3,
                "search rewrote Library truth",
            ),
        )
        page.locator("#librarySearch").fill("")

        page.locator("#libraryGenre").select_option(label="Ambient")
        checked(
            "genre-filter-is-local-view-state",
            lambda: require(
                page.locator("#trackChoices [data-track-ref]").count() == 1
                and page.locator(
                    '[data-track-ref="media.vexstream.test-tone.3"]'
                ).count()
                == 1
                and page.evaluate("window.vexV4.catalog.length") == 3,
                "genre filter rewrote catalog",
            ),
        )
        page.locator("#libraryGenre").select_option("all")

        page.locator("#librarySort").select_option("recent")
        checked(
            "library-sort-changes-render-order-without-changing-catalog-order",
            lambda: require(
                page.locator("#trackChoices [data-track-ref]").first.get_attribute(
                    "data-track-ref"
                )
                == "media.vexstream.test-tone.2"
                and page.evaluate(
                    "window.vexV4.catalog.map(x=>x.ref).join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.test-tone.2,"
                    "media.vexstream.test-tone.3"
                ),
                "sort mutated catalog or rendered wrong order",
            ),
        )

        page.locator(
            '#librarySidebar [data-shell-view="albums"]'
        ).click()
        checked(
            "album-browse-is-derived-from-current-library-view",
            lambda: require(
                page.evaluate(
                    "window.vexV4.shellNavigation.currentProjection.view"
                )
                == "albums"
                and page.locator(
                    "#trackChoices [data-library-group]"
                ).count()
                == 2,
                "album grouping drift",
            ),
        )
        page.locator(
            '#trackChoices [data-library-group="v4 practicum"] '
            '[data-library-group-action="open"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.libraryDetail?.kind"
            "==='album'"
        )
        checked(
            "album-detail-restores-v3-drill-in-terrain-over-v4-library-truth",
            lambda: require(
                page.locator("#libraryGroupDetail").is_visible()
                and page.locator("#libraryGroupTitle").inner_text()
                == "V4 Practicum"
                and page.locator(
                    "#libraryGroupTracks [data-track-ref]"
                ).count()
                == 2,
                "Album detail terrain drift",
            ),
        )
        page.locator("#libraryGroupBack").click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.libraryDetail===null"
        )

        page.locator(
            '#librarySidebar [data-shell-view="artists"]'
        ).click()
        checked(
            "artist-browse-is-derived-without-new-library-owner",
            lambda: require(
                page.evaluate(
                    "window.vexV4.shellNavigation.currentProjection.view"
                )
                == "artists"
                and page.locator(
                    "#trackChoices [data-library-group]"
                ).count()
                == 2,
                "artist grouping drift",
            ),
        )
        page.locator(
            '#trackChoices [data-library-group="vexstream qa"] '
            '[data-library-group-action="open"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.libraryDetail?.kind"
            "==='artist'"
        )
        checked(
            "artist-detail-restores-v3-drill-in-terrain-over-v4-library-truth",
            lambda: require(
                page.locator("#libraryGroupDetail").is_visible()
                and page.locator("#libraryGroupTitle").inner_text()
                == "VexStream QA"
                and page.locator(
                    "#libraryGroupTracks [data-track-ref]"
                ).count()
                == 2,
                "Artist detail terrain drift",
            ),
        )
        page.locator("#libraryGroupBack").click()

        page.locator("#brandHome").click()
        checked(
            "brand-returns-to-recognizable-library-home",
            lambda: require(
                page.evaluate(
                    "window.vexV4.shellNavigation.currentProjection.view"
                )
                == "home"
                and page.locator("#homeView").is_visible(),
                "brand Home route drift",
            ),
        )
        page.locator(
            '#librarySidebar [data-shell-view="songs"]'
        ).click()
        page.locator("#librarySort").select_option("title")

        page.locator("#discoverTopTab").click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='discover'"
        )
        checked(
            "discover-opens-on-recognizable-this-pc-route",
            lambda: require(
                page.evaluate(
                    "window.vexV4.discoverNavigation.currentProjection.mode"
                )
                == "pc"
                and page.locator("#sourceDiscover").is_visible()
                and page.locator("#youtubeImportURL").is_hidden()
                and page.locator("#exploreCandidates").is_hidden(),
                "Discover did not preserve the This PC route",
            ),
        )

        page.locator("#sourceDiscover").click()
        page.wait_for_function(
            "document.querySelectorAll('[data-source-candidate]').length===2"
        )
        checked(
            "folder-discovery-remains-temporary-until-include",
            lambda: require(
                page.evaluate(
                    "window.vexV4.sourceConfig.sources.join(',')"
                )
                == "/music"
                and page.locator(
                    '[data-source-candidate="/candidate"]'
                ).count()
                == 1,
                "discovery mutated source configuration",
            ),
        )

        page.locator(
            '[data-source-candidate="/ignore-me"] '
            '[data-source-candidate-action="ignore"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.sourceConfig.ignored.includes('/ignore-me')"
        )
        checked(
            "ignoring-discovered-folder-is-explicit-config-action",
            lambda: require(
                page.locator(
                    '[data-source-candidate="/ignore-me"]'
                ).count()
                == 0
                and page.evaluate(
                    "window.vexV4.sourceConfig.ignored.includes('/ignore-me')"
                )
                is True,
                "ignore action drift",
            ),
        )

        page.locator(
            '[data-source-candidate="/candidate"] '
            '[data-source-candidate-action="include"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.sourceConfig.sources.includes('/candidate')"
        )
        checked(
            "including-discovered-folder-is-separate-admission",
            lambda: require(
                page.locator('[data-source-path="/candidate"]').count() == 1
                and page.evaluate("window.vexV4.catalog.length") == 3,
                "Include did not currentize config/library independently",
            ),
        )

        page.locator('[data-discover-mode="youtube"]').click()
        page.wait_for_function(
            "window.vexV4.discoverNavigation.currentProjection.mode==='youtube'"
        )
        direct_import_starts = sum(
            1
            for row in report["requests"]
            if row["method"] == "POST"
            and row["path"] == "/api/import/start"
        )
        page.locator("#youtubeImportURL").fill(
            "https://www.youtube.com/watch?v=candidate.one"
        )
        page.locator("#youtubeInspect").click()
        page.wait_for_function("window.vexV4.importReview!==null")
        checked(
            "youtube-link-route-reviews-before-any-download-admission",
            lambda: require(
                page.locator("#importReview").is_visible()
                and page.locator("#importReviewSourceTitle").inner_text()
                == "Candidate One"
                and sum(
                    1
                    for row in report["requests"]
                    if row["method"] == "POST"
                    and row["path"] == "/api/import/start"
                )
                == direct_import_starts,
                "direct YouTube review crossed into acquisition",
            ),
        )
        page.locator("#importReviewClose").click()
        checked(
            "youtube-route-keeps-import-activity-in-its-own-terrain",
            lambda: require(
                page.locator("#importJobs").is_visible()
                and page.locator("#sourceDiscover").is_hidden()
                and page.locator("#exploreCandidates").is_hidden(),
                "YouTube terrain composition drift",
            ),
        )
        page.locator('[data-discover-mode="pc"]').click()

        before_picker_requests = len(report["requests"])
        page.locator("#sourceAdd").click()
        page.wait_for_function(
            "window.vexV4.sourceConfig.sources.includes('/picked')"
        )
        picker_requests = report["requests"][before_picker_requests:]
        checked(
            "native-folder-pick-is-followed-by-explicit-source-admission",
            lambda: require(
                any(
                    row["path"] == "/api/source/choose"
                    and row["method"] == "POST"
                    for row in picker_requests
                )
                and any(
                    row["path"] == "/api/source/add"
                    and row["method"] == "POST"
                    for row in picker_requests
                )
                and page.locator('[data-source-path="/picked"]').count() == 1,
                f"picker/admission sequence drift: {picker_requests}",
            ),
        )

        page.locator(
            '#librarySidebar [data-shell-view="playlists"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='playlists'"
        )

        checked(
            "browser-loads-durable-playlist-contract",
            lambda: require(
                any(
                    row["method"] == "GET"
                    and row["path"] == "/api/playlists"
                    for row in report["requests"]
                )
                and page.evaluate("window.vexV4.playlists.length") == 1
                and page.evaluate(
                    "window.vexV4.playlists[0].trackRefs.join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.missing,"
                    "media.vexstream.test-tone.3"
                ),
                f"playlist contract not consumed: {report['requests']}",
            ),
        )
        checked(
            "playlist-availability-is-derived-without-rewriting-durable-refs",
            lambda: require(
                page.evaluate(
                    "window.vexV4.resolvedPlaylists[0].playableRefs.join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.test-tone.3"
                )
                and page.evaluate(
                    "window.vexV4.resolvedPlaylists[0].missingRefs.join(',')"
                )
                == "media.vexstream.missing"
                and page.locator(
                    '[data-playlist-id="road-trip"] '
                    '[data-available="false"]'
                ).count()
                == 1,
                "playlist availability projection drift",
            ),
        )
        checked(
            "stable-player-and-control-identities",
            lambda: require(
                page.locator(
                    '[data-vex-screen="vexstream.desktop.library"]'
                ).count()
                == 1
                and page.locator(
                    '[data-vex-control="vexstream.queue.next"]'
                ).count()
                == 1,
                "semantic identity missing",
            ),
        )
        journey = page.evaluate("window.vexV4.journey")
        checked(
            "feature-owned-player-journey-is-addressable",
            lambda: require(
                journey["ref"]
                == "journey.vexstream.desktop.player.queue-basics"
                and len(journey["steps"]) == 6,
                "journey drift",
            ),
        )

        page.locator(
            '#librarySidebar [data-shell-view="songs"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='songs'"
        )

        page.locator(
            '[data-track-ref="media.vexstream.test-tone.1"] '
            '[data-track-action="edit"]'
        ).click()
        checked(
            "track-editor-opens-current-projected-metadata",
            lambda: require(
                page.locator("#libraryEditor").is_visible()
                and page.locator("#editTitle").input_value()
                == "VexStream Test Tone One",
                "track editor did not bind current Library projection",
            ),
        )

        page.locator("#editTitle").fill("Edited Test Tone One")
        page.locator("#editGenre").fill("Edited Genre")
        page.locator("#saveTrackDisplay").click()
        page.wait_for_function(
            "window.vexV4.catalog.find("
            "x=>x.ref==='media.vexstream.test-tone.1').title"
            "==='Edited Test Tone One'"
        )
        checked(
            "display-edit-is-reversible-overlay-not-source-metadata-rewrite",
            lambda: require(
                page.locator(
                    '[data-track-ref="media.vexstream.test-tone.1"] strong'
                ).inner_text()
                == "Edited Test Tone One"
                and page.evaluate(
                    "window.vexV4.catalog.find("
                    "x=>x.ref==='media.vexstream.test-tone.1').sourceTitle"
                )
                == "VexStream Test Tone One",
                "overlay/source distinction drift",
            ),
        )

        page.locator("#editPlaybackStart").fill("0.2")
        page.locator("#editPlaybackEnd").fill("1.0")
        page.locator("#savePlaybackWindow").click()
        page.wait_for_function(
            "window.vexV4.catalog.find("
            "x=>x.ref==='media.vexstream.test-tone.1')"
            ".playbackStartSeconds===0.2"
        )
        checked(
            "playback-window-edit-is-separate-from-display-overlay",
            lambda: require(
                page.evaluate(
                    "window.vexV4.catalog.find("
                    "x=>x.ref==='media.vexstream.test-tone.1')"
                    ".playbackEndSeconds"
                )
                == 1
                and page.evaluate(
                    "window.vexV4.catalog.find("
                    "x=>x.ref==='media.vexstream.test-tone.1').title"
                )
                == "Edited Test Tone One",
                "playback window crossed metadata ownership",
            ),
        )

        page.locator("#restoreTrackDisplay").click()
        page.wait_for_function(
            "window.vexV4.catalog.find("
            "x=>x.ref==='media.vexstream.test-tone.1').title"
            "==='VexStream Test Tone One'"
        )
        page.locator("#restorePlaybackWindow").click()
        page.wait_for_function(
            "window.vexV4.catalog.find("
            "x=>x.ref==='media.vexstream.test-tone.1')"
            ".playbackStartSeconds===0"
        )
        checked(
            "track-overlays-and-playback-window-restore-independently",
            lambda: require(
                page.evaluate(
                    "window.vexV4.catalog.find("
                    "x=>x.ref==='media.vexstream.test-tone.1').genre"
                )
                == "Test"
                and page.evaluate(
                    "window.vexV4.catalog.find("
                    "x=>x.ref==='media.vexstream.test-tone.1')"
                    ".playbackEndSeconds"
                )
                is None,
                "reversible Library edits did not restore",
            ),
        )
        page.locator("#libraryEditorClose").click()

        page.locator(
            '#librarySidebar [data-shell-view="playlists"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='playlists'"
        )

        page.locator("#newPlaylistName").fill("Focus Mix")
        page.locator("#createPlaylist").click()
        page.wait_for_function(
            "window.vexV4.playlists.some(x=>x.name==='Focus Mix')"
        )
        focus_id = page.evaluate(
            "window.vexV4.playlists.find(x=>x.name==='Focus Mix').id"
        )
        checked(
            "empty-playlist-can-be-formed-without-media-mutation",
            lambda: require(
                len(page.evaluate(
                    "window.vexV4.playlists.find("
                    "x=>x.name==='Focus Mix').trackRefs"
                ))
                == 0
                and page.evaluate("window.vexV4.catalog.length") == 3,
                "playlist creation mutated Library",
            ),
        )

        page.locator(
            '#librarySidebar [data-shell-view="songs"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='songs'"
        )

        first_card = page.locator(
            '[data-track-ref="media.vexstream.test-tone.1"]'
        )
        first_card.locator("[data-track-playlist-target]").select_option(
            focus_id
        )
        first_card.locator('[data-track-action="playlist-add"]').click()
        page.wait_for_function(
            "(id)=>window.vexV4.playlists.find(x=>x.id===id)"
            ".trackRefs.includes('media.vexstream.test-tone.1')",
            arg=focus_id,
        )

        third_card = page.locator(
            '[data-track-ref="media.vexstream.test-tone.3"]'
        )
        third_card.locator("[data-track-playlist-target]").select_option(
            focus_id
        )
        third_card.locator('[data-track-action="playlist-add"]').click()
        page.wait_for_function(
            "(id)=>window.vexV4.playlists.find(x=>x.id===id)"
            ".trackRefs.length===2",
            arg=focus_id,
        )

        page.locator(
            '#librarySidebar [data-shell-view="playlists"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='playlists'"
        )

        focus_card = page.locator(
            f'[data-playlist-id="{focus_id}"]'
        )
        focus_card.locator('[data-playlist-action="open"]').click()
        page.wait_for_function(
            "(id)=>window.vexV4.playlistNavigation.currentProjection"
            ".openPlaylistID===id",
            arg=focus_id,
        )
        checked(
            "playlist-browse-opens-recognizable-detail-terrain",
            lambda: require(
                page.locator("#playlistOverview").is_hidden()
                and page.locator("#playlistDetail").is_visible()
                and page.locator("#playlistDetailName").input_value()
                == "Focus Mix"
                and page.locator("#playlistDetailEntries li").count() == 2,
                "Playlist detail terrain drift",
            ),
        )

        page.locator("#playlistDetailAddSongs").click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='songs' "
            "&& window.vexV4.librarySelection.currentProjection.active===true"
        )
        checked(
            "playlist-add-songs-hands-off-to-existing-songs-selection",
            lambda: require(
                page.locator("#bulkPlaylistTarget").input_value() == focus_id
                and page.locator("#librarySelectionBar").is_visible(),
                "Playlist add-songs handoff did not preserve target",
            ),
        )
        page.locator(
            '[data-track-ref="media.vexstream.test-tone.2"] [data-track-select]'
        ).check()
        page.locator("#bulkPlaylistSelected").click()
        page.wait_for_function(
            "(id)=>window.vexV4.shellNavigation.currentProjection.view"
            "==='playlists' "
            "&& window.vexV4.playlistNavigation.currentProjection"
            ".openPlaylistID===id "
            "&& window.vexV4.playlists.find(x=>x.id===id)"
            ".trackRefs.length===3",
            arg=focus_id,
        )

        page.locator(
            '#playlistDetailEntries [data-playlist-index="1"] '
            '[data-playlist-entry-action="up"]'
        ).click()
        page.wait_for_function(
            "(id)=>window.vexV4.playlists.find(x=>x.id===id)"
            ".trackRefs[0]==='media.vexstream.test-tone.3'",
            arg=focus_id,
        )

        page.locator("#playlistDetailName").fill("Focus Renamed")
        page.locator("#playlistDetailRename").click()
        page.wait_for_function(
            "(id)=>window.vexV4.playlists.find(x=>x.id===id).name"
            "==='Focus Renamed'",
            arg=focus_id,
        )

        page.locator(
            '#playlistDetailEntries [data-playlist-index="2"] '
            '[data-playlist-entry-action="remove"]'
        ).click()
        page.wait_for_function(
            "(id)=>window.vexV4.playlists.find(x=>x.id===id)"
            ".trackRefs.length===2",
            arg=focus_id,
        )
        checked(
            "playlist-detail-crud-keeps-durable-order-and-current-availability-separate",
            lambda: require(
                page.evaluate(
                    "(id)=>window.vexV4.playlists.find(x=>x.id===id)"
                    ".trackRefs[0]",
                    focus_id,
                )
                == "media.vexstream.test-tone.3"
                and page.evaluate("window.vexV4.catalog.length") == 3,
                "playlist detail CRUD crossed into Library",
            ),
        )

        page.locator("#playlistDetailDelete").click()
        page.wait_for_function(
            "(id)=>!window.vexV4.playlists.some(x=>x.id===id) "
            "&& window.vexV4.playlistNavigation.currentProjection"
            ".openPlaylistID===''",
            arg=focus_id,
        )

        page.locator(
            '#librarySidebar [data-shell-view="songs"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='songs'"
        )

        page.locator(
            '[data-track-ref="media.vexstream.test-tone.2"] '
            '[data-track-action="trash"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.catalog.length===2 "
            "&& window.vexV4.trash.trash.length===1"
        )
        checked(
            "move-to-trash-changes-custody-and-current-library-projection",
            lambda: require(
                page.evaluate(
                    "window.vexV4.trash.trash[0].track.id"
                )
                == "media.vexstream.test-tone.2"
                and page.evaluate(
                    "'trashPath' in window.vexV4.trash.trash[0]"
                )
                is False,
                "Trash browser projection leaked custody path or lost track",
            ),
        )

        page.locator(
            '#librarySidebar [data-shell-view="deleted"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='deleted'"
        )

        page.locator(
            '#trashItems [data-trash-action="restore"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.catalog.length===3 "
            "&& window.vexV4.trash.trash.length===0"
        )
        checked(
            "trash-restore-rejoins-library-without-new-track-identity",
            lambda: require(
                page.evaluate(
                    "window.vexV4.catalog.some("
                    "x=>x.ref==='media.vexstream.test-tone.2')"
                )
                is True,
                "restored track did not rejoin Library",
            ),
        )

        page.locator(
            '#librarySidebar [data-shell-view="songs"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='songs'"
        )

        play = page.locator(
            '[data-vex-control="vexstream.playback.play-pause"]'
        )
        nxt = page.locator('[data-vex-control="vexstream.queue.next"]')
        prev = page.locator(
            '[data-vex-control="vexstream.queue.previous"]'
        )
        shuf = page.locator(
            '[data-vex-control="vexstream.queue.shuffle"]'
        )
        rep = page.locator(
            '[data-vex-control="vexstream.queue.repeat"]'
        )

        checked(
            "top-level-play-is-enabled-before-queue-exists",
            lambda: require(
                play.is_enabled()
                and play.get_attribute("data-text-ref") == "playback.play",
                "explicit seed-context Play missing",
            ),
        )
        first_media_path = "/media/media.vexstream.test-tone.1"
        with page.expect_request(
            lambda request: urlparse(request.url).path == first_media_path,
            timeout=5000,
        ):
            play.click()
        page.wait_for_function(
            "window.vexV4.queue.currentProjection.state.kind==='positioned' "
            "&& window.vexV4.media.paused===false"
        )
        checked(
            "play-seeds-library-context-and-starts-first",
            lambda: require(
                page.evaluate(
                    "window.vexV4.queue.currentProjection.state.items.length"
                )
                == 3
                and page.evaluate("window.vexV4.player.currentItem.ref")
                == "media.vexstream.test-tone.1"
                and play.get_attribute("data-text-ref")
                == "playback.pause",
                "Play composition drift",
            ),
        )
        checked(
            "playback-uses-opaque-library-media-url",
            lambda: require(
                page.evaluate("window.vexV4.media.currentSrc")
                == (
                    "http://vexstream.local/media/"
                    "media.vexstream.test-tone.1"
                )
                and any(
                    row["path"] == first_media_path
                    for row in report["requests"]
                ),
                f"opaque media route not used: {report['requests']}",
            ),
        )

        nxt.click()
        page.wait_for_function(
            "window.vexV4.player.currentItem.ref"
            "==='media.vexstream.test-tone.2'"
        )
        checked(
            "next-advances-queue-and-keeps-playback-active",
            lambda: require(
                page.evaluate("window.vexV4.media.paused") is False
                and play.get_attribute("data-text-ref")
                == "playback.pause",
                "Next composition drift",
            ),
        )

        page.locator("#queueMiniBtn").click()
        page.wait_for_function(
            "document.querySelector('#queueDrawer').classList.contains('open')"
        )
        checked(
            "expanded-queue-keeps-rich-current-detail-left-and-timeline-right",
            lambda: require(
                page.locator("#queueCurrentPane").get_attribute("data-track-ref")
                == "media.vexstream.test-tone.2"
                and "A deliberately descriptive current-track fixture"
                in page.locator("#queueCurrentPane").inner_text()
                and "A deliberately descriptive current-track fixture"
                not in page.locator("#queueList li.current").inner_text()
                and page.evaluate(
                    """() => {
                      const left=document.querySelector('#queueCurrentPane')
                        .getBoundingClientRect().width;
                      const right=document.querySelector('#queueTimelinePane')
                        .getBoundingClientRect().width;
                      return left>0 && right>=left*2;
                    }"""
                ),
                "expanded Queue lost its Now Playing/timeline split",
            ),
        )
        page.locator("#queueMiniBtn").click()

        page.evaluate("window.vexV4.media.currentTime=0")
        prev.click()
        page.wait_for_function(
            "window.vexV4.player.currentItem.ref"
            "==='media.vexstream.test-tone.1'"
        )
        checked(
            "previous-near-start-moves-to-prior-queue-item",
            lambda: require(
                page.evaluate("window.vexV4.player.currentItem.ref")
                == "media.vexstream.test-tone.1",
                "Previous composition drift",
            ),
        )

        shuf.click()
        checked(
            "shuffle-selects-and-preserves-current-item",
            lambda: require(
                shuf.get_attribute("aria-pressed") == "true"
                and page.evaluate(
                    "window.vexV4.queue.currentProjection.state.items"
                    ".join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.test-tone.3,"
                    "media.vexstream.test-tone.2"
                ),
                "shuffle upcoming law drift",
            ),
        )

        page.locator("#queueMiniBtn").click()
        page.wait_for_function(
            "document.querySelector('#queueDrawer').classList.contains('open')"
        )
        page.locator("#shuffleQueueUpcoming").click()
        checked(
            "drawer-shuffle-upcoming-reshuffles-suffix-without-toggling-mode",
            lambda: require(
                page.evaluate(
                    "window.vexV4.queue.currentProjection.state.items.join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.test-tone.2,"
                    "media.vexstream.test-tone.3"
                )
                and page.evaluate(
                    "window.vexV4.queue.currentProjection.state.shuffle"
                )
                is True
                and shuf.get_attribute("aria-pressed") == "true",
                "drawer Shuffle upcoming drift",
            ),
        )
        page.locator("#queueMiniBtn").click()
        rep.click()
        checked(
            "repeat-is-queue-truth-not-playback-truth",
            lambda: require(
                rep.get_attribute("aria-pressed") == "true"
                and page.evaluate(
                    "window.vexV4.queue.currentProjection.state.repeat"
                )
                is True
                and page.evaluate(
                    "window.vexV4.playback.currentProjection.state.repeat"
                )
                is None,
                "repeat ownership drift",
            ),
        )

        third_track_play=page.locator(
            '[data-track-ref="media.vexstream.test-tone.3"] '
            '[data-track-action="play"]'
        )
        third_track_play.click()
        page.wait_for_function(
            "window.vexV4.player.currentItem.ref"
            "==='media.vexstream.test-tone.3'"
        )
        checked(
            "fresh-shuffled-context-makes-the-whole-library-upcoming",
            lambda: require(
                page.evaluate(
                    "window.vexV4.queue.currentProjection.state.currentIndex"
                )
                == 0
                and page.evaluate(
                    "window.vexV4.queue.currentProjection.state.items.join(',')"
                )
                == (
                    "media.vexstream.test-tone.3,"
                    "media.vexstream.test-tone.2,"
                    "media.vexstream.test-tone.1"
                )
                and page.locator("#queueDrawerNextCount").inner_text()
                == "2 next",
                "fresh shuffled context retained a false sorted-history prefix",
            ),
        )

        shuf.click()
        third_track_play.click()
        page.wait_for_function(
            "window.vexV4.queue.currentProjection.state.currentIndex===2"
            " && window.vexV4.queue.currentProjection.state.shuffle===false"
        )
        shuf.click()
        nxt.click()
        page.wait_for_function(
            "window.vexV4.player.currentItem.ref"
            "==='media.vexstream.test-tone.1'"
        )
        checked(
            "repeat-wraps-next-from-end-to-first",
            lambda: require(
                page.evaluate("window.vexV4.player.currentItem.ref")
                == "media.vexstream.test-tone.1",
                "repeat wrap drift",
            ),
        )

        page.locator(
            '[data-track-ref="media.vexstream.test-tone.3"] '
            '[data-track-action="play-next"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.queue.currentProjection.state.items.length===4"
        )
        checked(
            "play-next-inserts-directly-after-current",
            lambda: require(
                page.evaluate(
                    "window.vexV4.queue.currentProjection.state.items"
                    ".join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.test-tone.3,"
                    "media.vexstream.test-tone.2,"
                    "media.vexstream.test-tone.3"
                ),
                "play-next payload drift",
            ),
        )

        page.locator(
            '[data-track-ref="media.vexstream.test-tone.2"] '
            '[data-track-action="append"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.queue.currentProjection.state.items.length===5"
        )
        checked(
            "add-to-queue-appends-without-changing-current",
            lambda: require(
                page.evaluate("window.vexV4.player.currentItem.ref")
                == "media.vexstream.test-tone.1"
                and page.evaluate(
                    "window.vexV4.queue.currentProjection.state.items.at(-1)"
                )
                == "media.vexstream.test-tone.2",
                "append payload drift",
            ),
        )

        page.locator(
            '#queueList li[data-queue-index="4"] '
            '[data-queue-action="up"]'
        ).click()
        checked(
            "reorder-preserves-current-item-identity",
            lambda: require(
                page.evaluate("window.vexV4.player.currentItem.ref")
                == "media.vexstream.test-tone.1"
                and page.evaluate(
                    "window.vexV4.queue.currentProjection.state.items"
                    ".join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.test-tone.3,"
                    "media.vexstream.test-tone.2,"
                    "media.vexstream.test-tone.2,"
                    "media.vexstream.test-tone.3"
                ),
                "reorder/current preservation drift",
            ),
        )

        page.locator(
            '#queueList li[data-queue-index="1"] '
            '[data-queue-action="play"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.player.currentItem.ref"
            "==='media.vexstream.test-tone.3'"
        )
        checked(
            "play-queue-item-selects-then-realizes-playback",
            lambda: require(
                page.evaluate("window.vexV4.media.paused") is False,
                "queue play-now did not realize playback",
            ),
        )

        page.locator(
            '[data-vex-control="vexstream.queue.clear-after-current"]'
        ).click()
        checked(
            "clear-after-current-preserves-history-and-current",
            lambda: require(
                page.evaluate(
                    "window.vexV4.queue.currentProjection.state.items"
                    ".join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.test-tone.3"
                )
                and page.evaluate("window.vexV4.player.currentItem.ref")
                == "media.vexstream.test-tone.3",
                "clear-after-current drift",
            ),
        )

        checked(
            "queue-render-marks-one-current-item",
            lambda: require(
                page.locator("#queueList li.current").count() == 1,
                "queue current projection drift",
            ),
        )
        current_before=page.evaluate("window.vexV4.player.currentItem.ref")
        checked(
            "current-queue-item-can-be-repositioned-but-not-removed",
            lambda: require(
                page.locator(
                    "#queueList li.current "
                    "[data-queue-action='remove']"
                ).is_disabled()
                and page.locator(
                    "#queueList li.current "
                    "[data-queue-action='up']"
                ).is_enabled()
                and page.locator("#queueList li.current").get_attribute(
                    "draggable"
                )
                == "true",
                "current Queue row did not expose reorder while retaining remove guard",
            ),
        )
        page.locator(
            "#queueList li.current [data-queue-action='up']"
        ).click()
        checked(
            "moving-current-queue-item-preserves-playing-identity",
            lambda: require(
                page.evaluate("window.vexV4.player.currentItem.ref")
                == current_before
                and page.evaluate(
                    "window.vexV4.queue.currentProjection.state.currentIndex"
                )
                == 0,
                "moving current Queue row changed Playback identity",
            ),
        )
        page.locator(
            '#librarySidebar [data-shell-view="playlists"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='playlists'"
        )

        page.locator(
            '[data-playlist-id="road-trip"] '
            '[data-playlist-action="play"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.player.currentItem.ref"
            "==='media.vexstream.test-tone.1'"
        )
        checked(
            "play-playlist-composes-durable-playlist-through-current-library-into-queue",
            lambda: require(
                page.evaluate(
                    "window.vexV4.queue.currentProjection.state.items"
                    ".join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.test-tone.3"
                )
                and page.evaluate(
                    "window.vexV4.playlists[0].trackRefs.join(',')"
                )
                == (
                    "media.vexstream.test-tone.1,"
                    "media.vexstream.missing,"
                    "media.vexstream.test-tone.3"
                )
                and page.evaluate("window.vexV4.media.paused") is False,
                "Playlist -> Library availability -> Queue composition drift",
            ),
        )

        page.locator(
            '#librarySidebar [data-shell-view="songs"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='songs'"
        )

        page.locator(
            '[data-track-ref="media.vexstream.test-tone.1"] '
            '[data-track-action="explore"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.exploreSession.currentProjection.kind==='active' "
            "&& window.vexV4.shellNavigation.currentProjection.view==='discover' "
            "&& window.vexV4.discoverNavigation.currentProjection.mode==='explore'"
        )
        checked(
            "explore-from-library-track-enters-the-recognizable-explore-route",
            lambda: require(
                page.locator("#exploreCandidates").is_visible()
                and page.locator("#sourceDiscover").is_hidden()
                and page.locator("#youtubeImportURL").is_hidden(),
                "Explore intent did not select the Explore terrain",
            ),
        )
        checked(
            "explore-from-library-track-forms-temporary-session",
            lambda: require(
                page.evaluate(
                    "window.vexV4.exploreSession.currentProjection.frame.candidates.length"
                )
                == 2
                and any(
                    row["method"] == "POST"
                    and row["path"] == "/api/discovery/radio"
                    for row in report["requests"]
                ),
                "Explore session did not form from Library track",
            ),
        )
        checked(
            "explore-renders-current-library-relationship-without-mutating-library",
            lambda: require(
                "Candidate Two" in page.locator("#exploreCandidates").inner_text()
                and "In library" in page.locator("#exploreCandidates").inner_text()
                and page.evaluate("window.vexV4.catalog.length") == 3,
                "Explore candidate relationship projection drift",
            ),
        )

        import_starts_before_review = sum(
            1
            for row in report["requests"]
            if row["method"] == "POST"
            and row["path"] == "/api/import/start"
        )
        page.locator(
            '[data-explore-index="0"] [data-explore-action="import"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.importReview!==null "
            "&& window.vexV4.discoverNavigation.currentProjection.mode==='youtube'"
        )
        checked(
            "explore-import-review-enters-youtube-route-without-starting-download",
            lambda: require(
                page.locator("#youtubeImportURL").is_hidden()
                and page.locator("#importReview").is_visible()
                and page.locator("#exploreCandidates").is_hidden(),
                "Explore import review did not replace the YouTube entry terrain",
            ),
        )
        checked(
            "explore-import-review-is-read-only-until-human-start",
            lambda: require(
                page.locator("#importReview").is_visible()
                and "Candidate One"
                in page.locator("#importReviewSourceTitle").inner_text()
                and "1 possible library match"
                in page.locator("#importDuplicateSummary").inner_text()
                and "Possible duplicate already in your Library"
                in page.locator("#importDuplicateDetails").inner_text()
                and "VexStream Test Tone Two"
                in page.locator("#importDuplicateDetails").inner_text()
                and page.locator(
                    "#importDuplicateDetails [data-import-duplicate-preview]"
                ).count()
                == 1
                and page.locator("#importDestination").input_value()
                == "/music"
                and page.locator("#importMode").input_value()
                == "split"
                and sum(
                    1
                    for row in report["requests"]
                    if row["method"] == "POST"
                    and row["path"] == "/api/import/start"
                )
                == import_starts_before_review,
                "Import review accidentally admitted acquisition",
            ),
        )
        checked(
            "import-review-preserves-two-provider-sections-for-human-editing",
            lambda: require(
                page.locator("[data-import-section]").count() == 2
                and page.locator(
                    '[data-import-section="1"] '
                    '[data-import-section-title]'
                ).input_value()
                == "Candidate Part One",
                "Import review section projection drift",
            ),
        )

        page.locator(
            '[data-explore-index="0"] [data-explore-action="preview"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.exploreSession.currentProjection.previewIndex===0 "
            "&& window.vexV4.media.paused===true"
        )
        checked(
            "explore-preview-pauses-local-music-through-playback-boundary",
            lambda: require(
                page.evaluate("window.vexV4.media.paused") is True
                and page.locator("#explorePreview").is_visible()
                and "candidate.one"
                in page.locator("#explorePreview").get_attribute("src"),
                "Explore preview did not pause local Playback",
            ),
        )

        page.locator(
            '[data-explore-index="0"] [data-explore-action="branch"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.exploreSession.currentProjection.frameIndex===1"
        )
        checked(
            "explore-branch-replaces-forward-history-without-resuming-local-music",
            lambda: require(
                page.evaluate(
                    "window.vexV4.exploreSession.currentProjection.frame.seed.title"
                )
                == "Candidate One"
                and page.evaluate("window.vexV4.media.paused") is True,
                "Explore branch/current Playback separation drift",
            ),
        )

        page.locator("#exploreBack").click()
        page.wait_for_function(
            "window.vexV4.exploreSession.currentProjection.frameIndex===0"
        )
        checked(
            "explore-back-keeps-preview-resume-held",
            lambda: require(
                page.evaluate("window.vexV4.media.paused") is True
                and page.evaluate(
                    "window.vexV4.exploreSession.currentProjection.frame.seed.title"
                )
                == "VexStream Test Tone One",
                "Explore Back resumed or lost prior frame",
            ),
        )

        page.locator("#exploreClear").click()
        page.wait_for_function(
            "window.vexV4.exploreSession.currentProjection.kind==='empty' "
            "&& window.vexV4.media.paused===false"
        )
        checked(
            "explore-clear-resumes-previous-local-playback-through-current-control",
            lambda: require(
                page.evaluate("window.vexV4.media.paused") is False
                and page.locator("#explorePreview").is_hidden(),
                "Explore Clear did not restore local Playback",
            ),
        )

        page.locator("#importStart").click()
        page.wait_for_function(
            "window.vexV4.importReview===null "
            "&& window.vexV4.importJobs.active.length===1"
        )
        checked(
            "reviewed-import-start-is-the-only-acquisition-admission",
            lambda: require(
                import_job_state["started"] is True
                and import_job_state["startBody"]["destination"] == "/music"
                and import_job_state["startBody"]["mode"] == "split"
                and page.evaluate(
                    "window.vexV4.importJobs.active[0].id"
                )
                == "job-import-1"
                and page.locator(
                    '[data-import-job-id="job-import-1"]'
                ).count()
                == 1,
                "reviewed Import did not become durable job activity",
            ),
        )
        checked(
            "import-start-does-not-rewrite-current-library-projection",
            lambda: require(
                page.evaluate("window.vexV4.catalog.length") == 3,
                "Import start mutated browser Library truth before commit",
            ),
        )

        page.wait_for_timeout(180)
        checked(
            "queue-changes-persist-through-v4-server-session-boundary",
            lambda: require(
                len(player_session_state["writes"]) > 0
                and player_session_state["session"]["schemaVersion"]
                == "vexstream.desktop-player-session/v1"
                and player_session_state["session"]["currentRef"]
                == player_session_state["session"]["items"][
                    player_session_state["session"]["currentIndex"]
                ],
                f"player session was not persisted exactly: {player_session_state}",
            ),
        )

        saved_session = dict(player_session_state["session"])
        saved_items = ",".join(saved_session["items"])
        saved_current = saved_session["currentRef"]
        expected_title = next(
            row["title"] for row in TRACKS if row["id"] == saved_current
        )

        restart = browser.new_page(viewport={"width": 1000, "height": 840})
        restart.on(
            "pageerror",
            lambda error: report["pageErrors"].append(
                "restart: " + str(error)
            ),
        )
        restart.on(
            "console",
            lambda message: report["console"].append(
                {
                    "type": message.type,
                    "text": "restart: " + message.text,
                }
            ),
        )
        restart.route("http://vexstream.local/**", route_request)
        restart.route(
            "https://www.youtube.com/**",
            lambda route: route.fulfill(
                status=200,
                content_type="text/html",
                body="<html><body>preview fixture</body></html>",
            ),
        )
        restart.set_content(html, wait_until="load")
        restart.wait_for_function("window.vexV4?.ready===true")

        checked(
            "clean-restart-restores-exact-queue-context-without-autoplay",
            lambda: require(
                restart.evaluate(
                    "window.vexV4.sessionRestoration.kind"
                )
                == "restored"
                and restart.evaluate(
                    "window.vexV4.queue.currentProjection.state.items.join(',')"
                )
                == saved_items
                and restart.evaluate(
                    "window.vexV4.player.currentItem.ref"
                )
                == saved_current
                and restart.evaluate("window.vexV4.media.paused") is True
                and restart.evaluate(
                    "window.vexV4.media.getAttribute('src')"
                )
                is None
                and restart.locator("#trackTitle").inner_text()
                == expected_title
                and restart.locator("#sessionState").inner_text()
                == "Restored",
                "clean restart did not restore exact stopped listening context",
            ),
        )

        checked(
            "restored-current-stays-a-play-intention-not-provider-success",
            lambda: require(
                restart.locator("#playPause").get_attribute("data-text-ref")
                == "playback.play"
                and restart.evaluate(
                    "window.vexV4.playback.currentProjection.state.kind"
                )
                == "no-media",
                "restoration crossed into Playback realization",
            ),
        )
        restart.close()

        page.locator(
            '#librarySidebar [data-shell-view="songs"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='songs'"
        )

        page.locator(
            '[data-track-ref="media.vexstream.test-tone.2"] '
            '[data-track-action="trash"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.trash.trash.length===1"
        )
        page.locator(
            '#librarySidebar [data-shell-view="deleted"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='deleted'"
        )

        page.locator(
            '#trashItems [data-trash-action="purge"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.trash.trash.length===0 "
            "&& window.vexV4.trash.deletedHistory.length===1"
        )
        checked(
            "permanent-trash-delete-is-explicit-and-preserves-deleted-history",
            lambda: require(
                page.evaluate(
                    "window.vexV4.trash.deletedHistory[0].track.id"
                )
                == "media.vexstream.test-tone.2"
                and page.evaluate("window.vexV4.catalog.length") == 2,
                "permanent Trash delete drift",
            ),
        )

        page.locator("#discoverTopTab").click()
        page.wait_for_function(
            "window.vexV4.shellNavigation.currentProjection.view==='discover'"
        )

        page.locator(
            '[data-source-path="/music"] [data-source-action="remove"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.catalog.length===0 "
            "&& window.vexV4.queue.currentProjection.state.kind==='empty'"
        )
        checked(
            "removing-current-library-source-clears-stale-queue-and-media",
            lambda: require(
                "/music" not in source_state["sources"]
                and page.evaluate("window.vexV4.catalog.length") == 0
                and page.evaluate(
                    "window.vexV4.queue.currentProjection.state.kind"
                )
                == "empty"
                and page.evaluate("window.vexV4.media.paused") is True
                and page.evaluate(
                    "window.vexV4.media.getAttribute('src')"
                )
                is None
                and page.locator("#sessionState").inner_text()
                == "Held · previous current unavailable",
                "source removal left stale playback truth",
            ),
        )

        page.locator("#exitVexStream").click()
        page.wait_for_function(
            "document.querySelector('#buildIdentity').textContent"
            ".includes('has exited')"
        )
        checked(
            "explicit-exit-control-sends-confirmed-process-intention",
            lambda: require(
                system_state["exitBody"]
                == {"confirm": "EXIT_VEXSTREAM"}
                and page.locator("#exitVexStream").is_disabled(),
                f"Exit intent drift: {system_state}",
            ),
        )

        checked(
            "browser-finished-without-runtime-errors",
            lambda: require(
                not report["pageErrors"]
                and not [
                    row
                    for row in report["console"]
                    if row["type"] == "error"
                ],
                f"errors: {report}",
            ),
        )
        browser.close()

    report["status"] = "PASS"
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
