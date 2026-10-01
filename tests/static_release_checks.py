#!/usr/bin/env python3
"""Static and syntax qualification for a VexStream Music source tree."""
from __future__ import annotations

import argparse
import json
from html.parser import HTMLParser
from pathlib import Path
import re
import subprocess
import tempfile

VERSION = "2.1.0"
EXPECTED_EXE = f"VexStreamMusic-{VERSION}.exe"


class IdCollector(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.ids: list[str] = []

    def handle_starttag(self, tag: str, attrs) -> None:
        for key, value in attrs:
            if key == "id" and value:
                self.ids.append(value)


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def extract_main_script(html: str) -> str:
    matches = list(re.finditer(r"<script(?:\s[^>]*)?>(.*?)</script>", html, flags=re.S | re.I))
    require(bool(matches), "no inline application script found")
    return matches[-1].group(1)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--artifact", type=Path)
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    root = args.root.resolve()

    checks: list[dict[str, str]] = []

    def checked(name: str, fn) -> None:
        fn()
        checks.append({"name": name, "status": "PASS"})

    main_go = (root / "main.go").read_text(encoding="utf-8")
    bridge = (root / "media_bridge.py").read_text(encoding="utf-8")
    html = (root / "ui" / "index.html").read_text(encoding="utf-8")

    checked("go-version-binding", lambda: require(f'const version = "{VERSION}"' in main_go, "main.go version mismatch"))
    checked("bridge-version-binding", lambda: require(f'VERSION = "{VERSION}"' in bridge, "media bridge version mismatch"))
    checked("ui-version-binding", lambda: require(f'<span class="version">{VERSION}</span>' in html, "UI version mismatch"))
    checked("embedded-source-binding", lambda: require("//go:embed ui/index.html media_bridge.py" in main_go, "embedded source declaration missing"))
    checked("platform-process-boundary", lambda: require("syscall" not in main_go and (root / "procattr_windows.go").is_file() and (root / "procattr_unix.go").is_file(), "OS-specific process attributes are not isolated from shared source"))
    checked("single-master-multiplatform-distribution-builder-present", lambda: require((root / "tests" / "build_all_platforms.py").is_file() and "Mac Silicon" in (root / "tests" / "build_all_platforms.py").read_text(encoding="utf-8") and "COPY-PASTE-TO-TERMINAL.md" in (root / "tests" / "build_all_platforms.py").read_text(encoding="utf-8"), "one-master-ZIP platform distribution contract missing"))
    checked("mac-youtube-runtime-bootstrap-present", lambda: require("func (a *App) setupDarwinImportTools()" in main_go and "/opt/homebrew/bin/python3" in main_go and 'brew, "install", "ffmpeg"' in main_go and 'runtime.GOOS == "darwin"' in main_go, "macOS YouTube runtime bootstrap contract missing"))
    checked("mac-private-import-venv-present", lambda: require('return filepath.Join(venv, "bin", "python3")' in main_go and "importRuntimePythonPath" in main_go, "macOS app-local import runtime path missing"))
    checked("import-readiness-split-present", lambda: require("ffmpegReady" in main_go and "importReady" in main_go and "YouTube search is ready; MP3 download still needs FFmpeg." in html and "Finish import setup" in html, "search/import readiness split missing"))
    checked("windows-winget-ffmpeg-discovery-present", lambda: require("findWindowsWinGetFFmpeg" in main_go and 'filepath.Join(local, "Microsoft", "WinGet", "Packages")' in main_go and 'strings.EqualFold(info.Name(), "ffmpeg.exe")' in main_go, "WinGet FFmpeg package discovery missing"))
    checked("windows-import-setup-is-direct-and-idempotent", lambda: require('runSetupCommand(15*time.Minute, py, "-m", "pip", "install", "--upgrade", "pip", "yt-dlp[default,curl-cffi]")' in main_go and 'exec.LookPath("winget.exe")' in main_go and 'powershell.exe' not in main_go[main_go.index("func (a *App) setupWindowsImportTools()"):main_go.index("func (a *App) setupDarwinImportTools()")], "Windows import setup still depends on monolithic PowerShell bootstrap"))
    checked("import-start-readiness-preflight-present", lambda: require("async function ensureImportRuntimeReady()" in html and "if(!await ensureImportRuntimeReady())return;" in html and "projectImportRuntimeStatus" in html, "Download & add can still bypass runtime readiness preflight"))

    def duplicate_ids() -> None:
        parser_ = IdCollector()
        parser_.feed(html)
        dupes = sorted({x for x in parser_.ids if parser_.ids.count(x) > 1})
        require(not dupes, f"duplicate DOM ids: {dupes}")

    checked("dom-ids-unique", duplicate_ids)
    checked("library-sequence-activation-present", lambda: require("function playLibraryTrack(id)" in html, "library sequence activation missing"))
    checked("filtered-songs-display-playback-separation-present", lambda: require("function sortedLibraryTracks(){return sortTrackList(state.tracks)}" in html and "currentView==='songs'?sortedLibraryTracks()" in html and "setQueue(shuf(sortedLibraryTracks()),0,true)" in html and "setQueue(sortedLibraryTracks(),0,true)" in html, "Songs filtering still leaks into full-library playback semantics"))
    checked("startup-session-version-source-bound", lambda: require("if v := runningVersion(u); v == version" in main_go and 'map[string]any{"ok": true, "version": version}' in main_go, "startup/session version identity is not source-bound"))
    checked("real-media-server-regression-present", lambda: require((root / "tests" / "media_server_regression.py").is_file() and (root / "tests" / "fixtures" / "vexstream-test-tone.mp3").is_file(), "real media server regression fixture/gate missing"))
    checked("play-source-recovery-present", lambda: require("function togglePlayback()" in html and "bindClick('playBtn',togglePlayback)" in html, "recovery-aware play toggle missing"))
    checked("loaded-track-identity-present", lambda: require("audio.dataset.trackId=t.id" in html, "loaded-track identity missing"))
    checked("render-player-bounds-in-function-scope", lambda: require("function renderPlayer(){const t=currentTrack(),b=playbackBounds" in html, "playback bounds are not function-scoped"))
    checked("collapsed-drawer-contract-present", lambda: require(".queue-drawer.open" in html and "height:36px" in html, "collapsed queue geometry contract missing"))
    checked("crop-contract-present", lambda: require("function playbackBounds" in html and "playbackStart" in html and "playbackEnd" in html, "playback crop contract missing"))
    checked("album-drilldown-terrain-present", lambda: require('id="albumDetail"' in html and "function renderAlbumDetail()" in html and "album-track-row" in html, "album drill-down terrain contract missing"))
    checked("album-reverse-artist-lattice-present", lambda: require("function albumArtistNames" in html and "function albumArtistRelationsMarkup" in html and "data-artist-relation" in html and "Artists / collaborators" in html and "function navigateToArtist" in html, "album-to-artist reverse lattice contract missing"))
    checked("artist-navigation-card-contract-present", lambda: require('id="artistDetail"' in html and "function artistCardHTML" in html and "data-artist-open" in html and "artist-card-name" in html, "artist navigation-card contract missing"))
    checked("artist-search-sort-terrain-present", lambda: require("function renderArtistSongs()" in html and "artistSongSearch" in html and "artistSortSummary" in html and "function playArtistTrack" in html, "artist scoped search/sort song terrain missing"))
    checked("artist-collapsible-album-shelf-present", lambda: require("function renderArtistAlbumShelf()" in html and "artistAlbumShelfLimit" in html and "artistAlbumsExpanded" in html and "data-artist-album-filter" in html, "artist collapsible bounded album shelf missing"))
    checked("queue-timeline-context-present", lambda: require("function queueTimelineRow" in html and "History" in html and "queueDrawerNextCount" in html, "queue history/current/up-next projection missing"))
    checked("queue-reorder-current-preservation-present", lambda: require("function moveQueueTo(from,to)" in html and "from===state.queueIndex" in html and "state.queueIndex=nextCurrent" in html, "queue reorder current-node preservation missing"))
    checked("queue-action-feedback-present", lambda: require("function flashAction" in html and "queueTrackFromButton" in html and "☷＋" in html, "explicit queue action feedback missing"))
    checked("shuffle-visible-order-contract-present", lambda: require("function toggleShuffleMode()" in html and "function next(){if(!state.queue.length)return;if(state.queueIndex+1<state.queue.length)" in html, "shuffle/next visible-order contract missing"))
    checked("adaptive-navigation-drawer-present", lambda: require('id="libraryNavToggle"' in html and "function syncLibraryNav()" in html and "drawer-open" in html, "adaptive navigation drawer contract missing"))
    checked("configurable-song-fields-present", lambda: require("songFieldDefaults" in html and "function openSongLayoutSettings()" in html and "--song-title-width" in html, "configurable song field/title-space contract missing"))
    checked("narrow-track-card-projection-present", lambda: require("song-mobile-meta" in html and "@media(max-width:760px)" in html and "document.documentElement.scrollWidth" not in html, "narrow track-card projection missing"))
    checked("intent-first-track-actions-present", lambda: require("track-action-quick" in html and "track-action-row" in html and "quickAddTrackToPlaylist" in html and "crop-glyph" in html, "intent-first track action contract missing"))
    checked("single-track-action-close-present", lambda: require('id="modalCloseBtn" class="modal-close"' in html and "style=\"margin-right:auto\"" in html, "track-action close/destructive layout contract missing"))
    checked("repeat-scope-navigation-present", lambda: require("function navigateLibraryScope(v)" in html and "v==='albums'&&currentView==='albums'" in html and "v==='artists'&&currentView==='artists'" in html and "goLibraryHome" in html, "repeat-click scope-root navigation contract missing"))
    checked("now-playing-lattice-navigation-present", lambda: require('id="brandHome"' in html and 'id="nowDetailsBtn"' in html and "function currentTrackRelationsMarkup" in html and "data-now-artist" in html and "data-now-album" in html and "function navigateToAlbum" in html, "now-playing details/artist/album navigation contract missing"))
    checked("punctuation-insensitive-token-search-present", lambda: require("function normalizeSearchText" in html and "function librarySearchMatches" in html and "hay.compact.includes(q.compact)" in html and "artistVisibleSongTracks" in html, "normalized token/compact library search contract missing"))
    checked("whole-source-duplicate-warning-present", lambda: require("${duplicateHTML()}${has?" in html and "Possible duplicate already in your library" in html, "whole-source duplicate advisory is still gated behind track sections"))
    checked("duplicate-quality-comparison-present", lambda: require("SourceQuality AudioQuality" in main_go and "QualityComparison" in main_go and "source creator matches existing artist" in main_go and "sourceQuality" in bridge and "source_audio_quality" in bridge, "duplicate identity/quality comparison contract missing"))
    checked("duplicate-logic-regression-test-present", lambda: require((root / "duplicate_logic_test.go").is_file() and "Paramore: Ain't It Fun [OFFICIAL VIDEO]" in (root / "duplicate_logic_test.go").read_text(encoding="utf-8"), "duplicate regression test missing"))
    checked("existing-library-artist-suggestion-present", lambda: require("inferExistingArtistSuggestion" in main_go and "titleStartsWithArtistCredit" in main_go and "artistSuggestion" in main_go and "importArtistSuggestion" in html and "Matched existing library artist" in html and "Missy Elliott - Lose Control" in (root / "duplicate_logic_test.go").read_text(encoding="utf-8"), "existing-library artist suggestion contract missing"))
    checked("discovery-radio-route-present", lambda: require('/api/discovery/radio' in main_go and 'func (a *App) discoveryRadio' in main_go and 'SEARCH_DERIVED' in main_go, "metadata-only discovery route missing"))
    checked("discovery-radio-ui-present", lambda: require('id="radioDiscoverTab"' in html and 'id="radioDiscoverPane"' in html and 'Discover more like this' in html and 'Previewed ≠ downloaded' in html, "radio discovery UI contract missing"))
    checked("discovery-provider-preview-is-visible-youtube-embed", lambda: require('id="radioPreviewFrame"' in html and 'referrerpolicy="strict-origin-when-cross-origin"' in html and 'youtube.com/embed/' in html and 'allow="autoplay; encrypted-media; picture-in-picture"' in html, "visible provider preview contract missing"))
    checked("discovery-session-is-memory-only-and-queue-separate", lambda: require('let discoverySession={frames:[]' in html and 'state.queue' not in html[html.index('function renderDiscoverySession'):html.index('function renderYouTubeResults')] and "localStorage.setItem('vexstream.discovery" not in html, "discovery session leaked into queue/persistence"))
    checked("discovery-library-match-uses-existing-duplicate-semantics", lambda: require('discoveryLibraryMatch' in main_go and 'findDuplicateMatches' in main_go and 'IN_LIBRARY' in main_go and 'NOT_IN_LIBRARY' in main_go, "discovery candidate library-match annotation missing"))
    checked("discovery-go-regression-test-present", lambda: require((root / "discovery_radio_test.go").is_file() and "TestDiscoveryQueriesAreDeterministicAndBounded" in (root / "discovery_radio_test.go").read_text(encoding="utf-8"), "discovery Go regression tests missing"))

    def node_check() -> None:
        script = extract_main_script(html)
        with tempfile.TemporaryDirectory(prefix="vexstream-js-") as td:
            js = Path(td) / "vexstream-ui.js"
            js.write_text(script, encoding="utf-8")
            proc = subprocess.run(["node", "--check", str(js)], capture_output=True, text=True)
            require(proc.returncode == 0, f"node --check failed:\n{proc.stdout}\n{proc.stderr}")

    checked("embedded-javascript-syntax", node_check)

    if args.artifact:
        artifact = args.artifact.resolve()
        checked("versioned-executable-name", lambda: require(artifact.name == EXPECTED_EXE, f"expected {EXPECTED_EXE}, got {artifact.name}"))
        checked("executable-present", lambda: require(artifact.is_file() and artifact.stat().st_size > 0, "executable missing or empty"))

    report = {
        "schemaVersion": "vexstream.static-release-checks/v1",
        "appVersion": VERSION,
        "sourceRoot": str(root),
        "artifact": str(args.artifact.resolve()) if args.artifact else None,
        "checks": checks,
        "status": "PASS",
    }
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
