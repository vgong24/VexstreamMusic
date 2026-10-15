#!/usr/bin/env python3
from __future__ import annotations

import argparse
import base64
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
        "playbackStart": 0,
    },
    {
        "id": "media.vexstream.test-tone.2",
        "title": "VexStream Test Tone Two",
        "artist": "VexStream QA",
        "album": "V4 Practicum",
        "genre": "Test",
        "year": "2026",
        "duration": 1.25,
        "playbackStart": 0,
    },
    {
        "id": "media.vexstream.test-tone.3",
        "title": "VexStream Test Tone Three",
        "artist": "VexStream QA",
        "album": "V4 Practicum",
        "genre": "Test",
        "year": "2026",
        "duration": 1.25,
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
                body=json.dumps({"tracks": TRACKS}),
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
                body=json.dumps({"playlists": PLAYLISTS}),
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
        # The execution sandbox blocks HTTP top-level navigation by administrator policy.
        # Keep the real product-relative URLs, but host this proof in an in-memory document whose
        # <base> points at the intercepted VexStream origin.
        page.set_content(html, wait_until="load")
        page.wait_for_function("window.vexV4?.ready===true")

        checked(
            "real-browser-loads-v4-player-surface",
            lambda: require(
                not report["pageErrors"],
                f"page errors: {report['pageErrors']}",
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
            "dynamic-library-metadata-renders-as-data",
            lambda: require(
                "VexStream Test Tone One"
                in page.locator("#trackChoices").inner_text()
                and "VexStream QA"
                in page.locator("#trackChoices").inner_text(),
                "catalog metadata did not render",
            ),
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
                    '[data-vex-screen="vexstream.desktop.player-practicum"]'
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
                    row["path"]
                    == "/media/media.vexstream.test-tone.1"
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

        page.locator(
            '[data-track-ref="media.vexstream.test-tone.3"] '
            '[data-track-action="play"]'
        ).click()
        page.wait_for_function(
            "window.vexV4.player.currentItem.ref"
            "==='media.vexstream.test-tone.3'"
        )
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
        checked(
            "current-queue-item-cannot-be-removed-or-reordered",
            lambda: require(
                page.locator(
                    "#queueList li.current "
                    "[data-queue-action='remove']"
                ).is_disabled()
                and page.locator(
                    "#queueList li.current "
                    "[data-queue-action='up']"
                ).is_disabled(),
                "current item mutation guard drift",
            ),
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
