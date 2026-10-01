#!/usr/bin/env python3
"""Headless UI regression proof for VexStream Music.

The test executes the actual embedded HTML/JavaScript in Chromium with a local,
deterministic API/audio fixture. It intentionally performs no network or media-file
mutation. This catches browser-runtime failures that syntax and string checks miss.
"""
from __future__ import annotations

import argparse
import json
import os
import platform
import importlib.metadata
from pathlib import Path
import shutil
import sys
from typing import Any

from playwright.sync_api import Page, sync_playwright

VERSION = "2.0.0"

TRACKS: list[dict[str, Any]] = [
    {
        "id": "t1",
        "path": "/music/t1.mp3",
        "filename": "t1.mp3",
        "title": "Track 1",
        "artist": "Artist 1",
        "album": "Album",
        "genre": "Test",
        "duration": 10,
        "source": "/music",
        "addedAt": "2026-09-29T00:00:01Z",
    },
    {
        "id": "t2",
        "path": "/music/t2.mp3",
        "filename": "t2.mp3",
        "title": "Track 2",
        "artist": "Artist 2",
        "album": "Album",
        "genre": "Test",
        "duration": 11,
        "source": "/music",
        "addedAt": "2026-09-29T00:00:02Z",
    },
    {
        "id": "t3",
        "path": "/music/t3.mp3",
        "filename": "t3.mp3",
        "title": "Track 3",
        "artist": "Artist 3",
        "album": "Album",
        "genre": "Test",
        "duration": 12,
        "source": "/music",
        "addedAt": "2026-09-29T00:00:03Z",
    },
    {
        "id": "t4",
        "path": "/music/t4.mp3",
        "filename": "t4.mp3",
        "title": "Track 4",
        "artist": "Artist 4",
        "album": "Album",
        "genre": "Test",
        "duration": 13,
        "source": "/music",
        "addedAt": "2026-09-29T00:00:04Z",
    },
]


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def snapshot(page: Page) -> dict[str, Any]:
    return page.evaluate(
        """() => ({
          queue: [...state.queue],
          queueIndex: state.queueIndex,
          current: state.queue[state.queueIndex] || null,
          src: audio.currentSrc,
          trackId: audio.dataset.trackId || null,
          paused: audio.paused,
          currentTime: audio.currentTime,
          title: nowTitle.textContent,
          meta: nowMeta.textContent,
          playText: playBtn.textContent,
          queueOpen: queueDrawer.classList.contains('open'),
          queueCountText: queueDrawerCount.textContent,
          queueNextText: queueDrawerNextCount.textContent,
          queueMiniText: queueMiniCount.textContent
        })"""
    )


def fixture_prelude() -> str:
    return f"""<script>
window.__mockTracks={json.dumps(TRACKS)};
Object.defineProperty(window,'localStorage',{{configurable:true,value:(()=>{{
  const m=new Map();
  return {{
    getItem:k=>m.has(k)?m.get(k):null,
    setItem:(k,v)=>m.set(k,String(v)),
    removeItem:k=>m.delete(k),
    clear:()=>m.clear()
  }};
}})()}});
window.__mockRequests=[];
window.fetch=async function(path,opts={{}}){{
  const p=new URL(String(path),'http://mock.local').pathname;
  window.__mockRequests.push({{path:p,method:opts.method||'GET',body:opts.body||null}});
  let d={{}};
  if(p==='/api/library') d={{tracks:window.__mockTracks,sources:[{{path:'/music',name:'music',trackCount:4}}],candidates:[],ignored:[]}};
  else if(p==='/api/library/trash') d={{trash:[],deletedHistory:[]}};
  else if(p==='/api/playlists') d={{playlists:[{{id:'p1',name:'Favorites',trackIds:[]}}]}};
  else if(p==='/api/import/jobs') d={{jobs:[],active:[],ready:[],completed:[]}};
  else if(p==='/api/import/status') d=window.__mockImportStatus||{{ready:true,searchReady:true,ffmpegReady:true,importReady:true,detail:'mock'}};
  else if(p==='/api/system/preflight') d={{issues:[],liveInstances:[],scan:{{running:false}},library:{{tracks:4,sources:1}},importRuntime:{{ready:true}}}};
  else if(p==='/api/diagnostics/session') d={{listening:true,sessionId:'mock',eventCount:0,sessionStartedAt:new Date().toISOString()}};
  else if(p==='/api/diagnostics') d={{}};
  else d={{ok:true}};
  return {{ok:true,status:200,statusText:'OK',json:async()=>d,blob:async()=>new Blob([JSON.stringify(d)])}};
}};
for(const id of ['audio','duplicatePreviewAudio','cropPreviewAudio']){{
  const el=document.getElementById(id); if(!el) continue;
  const st={{paused:true,currentTime:0,duration:120,src:'',readyState:4,volume:1,playCalls:0,pauseCalls:0}};
  Object.defineProperties(el,{{
    paused:{{configurable:true,get:()=>st.paused}},
    currentTime:{{configurable:true,get:()=>st.currentTime,set:v=>{{st.currentTime=Number(v)||0;}}}},
    duration:{{configurable:true,get:()=>st.duration}},
    currentSrc:{{configurable:true,get:()=>st.src}},
    readyState:{{configurable:true,get:()=>st.readyState}},
    src:{{configurable:true,get:()=>st.src,set:v=>{{st.src=String(v);st.currentTime=0;queueMicrotask(()=>el.dispatchEvent(new Event('loadedmetadata')));}}}},
    volume:{{configurable:true,get:()=>st.volume,set:v=>{{st.volume=Number(v);}}}}
  }});
  el.play=()=>{{st.playCalls++;st.paused=false;el.dispatchEvent(new Event('play'));return Promise.resolve();}};
  el.pause=()=>{{st.pauseCalls++;st.paused=true;el.dispatchEvent(new Event('pause'));}};
  el.load=()=>{{}};
  const origRemove=el.removeAttribute.bind(el);
  el.removeAttribute=(name)=>{{if(name==='src')st.src='';return origRemove(name)}};
  el.__state=st;
}}
</script>"""


def browser_executable(explicit: str | None) -> str | None:
    if explicit:
        return explicit
    env = os.environ.get("VEXSTREAM_CHROMIUM")
    if env:
        return env
    for name in ("chromium", "chromium-browser", "google-chrome", "chrome"):
        found = shutil.which(name)
        if found:
            return found
    return None


def run(html_path: Path, executable: str | None) -> dict[str, Any]:
    html = html_path.read_text(encoding="utf-8")
    marker = html.rfind("<script>")
    require(marker >= 0, "main application script was not found")
    html = html[:marker] + fixture_prelude() + html[marker:]

    report: dict[str, Any] = {
        "schemaVersion": "vexstream.ui-runtime-regression/v1",
        "appVersion": VERSION,
        "html": str(html_path),
        "checks": [],
        "console": [],
        "pageErrors": [],
        "environment": {
            "python": platform.python_version(),
            "playwright": importlib.metadata.version("playwright"),
            "browser": None,
        },
    }

    def checked(name: str, fn) -> None:
        fn()
        report["checks"].append({"name": name, "status": "PASS"})

    with sync_playwright() as p:
        launch: dict[str, Any] = {"headless": True, "args": ["--no-sandbox"]}
        if executable:
            launch["executable_path"] = executable
        browser = p.chromium.launch(**launch)
        report["environment"]["browser"] = browser.version
        page = browser.new_page(viewport={"width": 1280, "height": 720})
        page.on("console", lambda m: report["console"].append({"type": m.type, "text": m.text}))
        page.on("pageerror", lambda e: report["pageErrors"].append(str(e)))
        page.set_content(html, wait_until="load")
        page.wait_for_function("state.tracks.length===4")

        checked(
            "initial-render-has-no-browser-runtime-error",
            lambda: require(not report["pageErrors"], f"page errors during initial render: {report['pageErrors']}"),
        )
        checked(
            "version-badge-is-current",
            lambda: require(page.locator("header .version").inner_text().strip() == VERSION, "version badge drift"),
        )

        def import_readiness_split_contract() -> None:
            page.evaluate("window.__mockImportStatus={ready:true,searchReady:true,ffmpegReady:false,importReady:false,detail:'Search works; FFmpeg pending'}")
            page.evaluate("refreshImportStatus()")
            page.wait_for_timeout(50)
            text = page.locator("#ytRuntime").inner_text()
            require("YouTube search is ready." in text, f"partial readiness not projected: {text}")
            require(page.locator("#setupYt").count() == 1, "finish-setup action missing for partial readiness")
            require(page.locator("#setupYt").inner_text().strip() == "Finish import setup", "partial readiness action label drift")
            page.evaluate("window.__mockImportStatus={ready:true,searchReady:true,ffmpegReady:true,importReady:true,detail:'All ready'}")
            page.evaluate("refreshImportStatus()")
            page.wait_for_timeout(50)
            require("YouTube import is ready." in page.locator("#ytRuntime").inner_text(), "full import readiness not projected")

        checked("youtube-runtime-distinguishes-search-ready-from-import-ready", import_readiness_split_contract)

        def adaptive_navigation_contract() -> None:
            page.click("#libraryNavToggle")
            require(page.locator("#libraryShell").evaluate("e => e.classList.contains('nav-collapsed')"), "desktop navigation did not collapse")
            page.click("#libraryNavToggle")
            require(not page.locator("#libraryShell").evaluate("e => e.classList.contains('nav-collapsed')"), "desktop navigation did not expand")
            page.set_viewport_size({"width": 500, "height": 760})
            page.wait_for_timeout(50)
            page.click("#libraryNavToggle")
            page.wait_for_timeout(220)
            require(page.locator("#librarySidebar").evaluate("e => e.classList.contains('drawer-open')"), "narrow navigation did not become an overlay drawer")
            require(page.locator("#librarySidebar").bounding_box()["x"] >= -1, f"navigation drawer did not finish on-screen: {page.locator('#librarySidebar').bounding_box()}")
            require(not page.locator("#navBackdrop").evaluate("e => e.classList.contains('hidden')"), "narrow navigation backdrop missing")
            page.locator("#librarySidebar button[data-view='albums']").click()
            require(not page.locator("#librarySidebar").evaluate("e => e.classList.contains('drawer-open')"), "navigation drawer did not close after destination selection")
            require(not page.locator("#albumsView").evaluate("e => e.classList.contains('hidden')"), "drawer destination did not open")
            page.set_viewport_size({"width": 1280, "height": 720})
            page.wait_for_timeout(50)

        checked("adaptive-navigation-collapses-and-becomes-narrow-overlay", adaptive_navigation_contract)

        page.click("button[data-view='songs']")
        page.wait_for_selector("#songsTable tbody tr")
        rows = page.locator("#songsTable tbody tr")
        require(rows.count() == 4, "expected four deterministic song rows")

        def song_layout_fields_are_configurable() -> None:
            genre_header = page.locator("#songsTable th.col-genre")
            require(genre_header.evaluate("e => getComputedStyle(e).display === 'none'"), "Genre should be hidden by default in 1.0 desktop layout")
            page.click("#songLayoutBtn")
            require(page.locator("#modalTitle").inner_text() == "Song layout", "song layout dialog did not open")
            genre_toggle = page.locator(".song-layout-field").filter(has_text="Genre").locator("input")
            genre_toggle.check()
            require(not genre_header.evaluate("e => getComputedStyle(e).display === 'none'"), "Genre field could not be enabled")
            slider = page.locator(".layout-range input[type=range]")
            slider.fill("58")
            require(page.evaluate("() => getComputedStyle(document.documentElement).getPropertyValue('--song-title-width').trim()") == "58%", "title-space preference did not apply")
            genre_toggle.uncheck()
            slider.fill("46")
            page.click("#modalCloseBtn")

        checked("song-fields-and-title-space-are-user-configurable", song_layout_fields_are_configurable)

        def normalized_library_search_contract() -> None:
            page.evaluate("""() => {
              const t=state.tracks.find(x=>x.id==='t4');
              window.__savedSearchTrack={...t};
              Object.assign(t,{title:"Ain't It Fun",artist:'Paramore',album:'Paramore Greatest Hits'});
              search.value=''; render(); showView('songs');
            }""")
            for query in ["aint", "ain't", "ain", "param aint"]:
                page.fill("#search", query)
                page.wait_for_timeout(20)
                text=page.locator("#songsTable").inner_text()
                require("Ain't It Fun" in text, f"normalized search failed to rediscover apostrophe/token variant {query!r}: {text}")
            page.evaluate("""() => {
              const i=state.tracks.findIndex(x=>x.id==='t4');
              state.tracks[i]=window.__savedSearchTrack;
              search.value=''; render(); showView('songs');
            }""")

        checked("library-search-normalizes-apostrophes-and-token-prefixes", normalized_library_search_contract)

        def whole_source_duplicate_advisory_contract() -> None:
            page.evaluate("""() => {
              topScreen('discover'); discoverPane('youtube');
              inspected={
                id:'different-youtube-id',
                url:'https://www.youtube.com/watch?v=different',
                title:"Paramore: Ain't It Fun [OFFICIAL VIDEO]",
                channel:'Paramore', duration:228, album:'', artist:'', genre:'Music', categories:['Music'],
                sections:[], hasSections:false,
                sourceQuality:{codec:'opus',container:'webm',bitrateKbps:130,sampleRateHz:48000,basis:'youtube.inspect.best-audio-candidate'}
              };
              inspectedDuplicates=[{kind:'whole',title:inspected.title,duration:228,sourceQuality:inspected.sourceQuality,matches:[{
                trackId:'t4',title:"Ain't It Fun",artist:'Paramore',album:'Paramore Greatest Hits',duration:297,
                score:67,confidence:'possible',relation:'SAME_SONG_DIFFERENT_EDIT_POSSIBLE',
                reasons:['same normalized song title','source creator matches existing artist','duration differs by 69s; may be a different edit'],
                localQuality:{codec:'mp3',container:'mp3',bitrateKbps:192,basis:'local.file-size-duration-estimate'},
                qualityComparison:{state:'NOT_DIRECTLY_COMPARABLE',summary:'Different codecs or containers; bitrate alone cannot prove which copy sounds better.'}
              }]}];
              renderImportPlan();
            }""")
            require(page.locator("#ytPlan .duplicate-card").count() == 1, "whole-track source without chapters hid duplicate advisory")
            text=page.locator("#ytPlan .duplicate-card").inner_text()
            require("Possible duplicate already in your library" in text and "Likely same song" in text, f"duplicate identity warning missing: {text}")
            require("Existing:" in text and "Source:" in text and "Quality check:" in text, f"duplicate quality comparison missing: {text}")
            page.evaluate("topScreen('library'); showView('songs'); search.value=''; render()")

        checked("whole-track-import-shows-duplicate-and-quality-advisory", whole_source_duplicate_advisory_contract)

        def existing_library_artist_suggestion_contract() -> None:
            page.evaluate("""() => {
              topScreen('discover'); discoverPane('youtube');
              inspected={
                id:'missy-source',
                url:'https://www.youtube.com/watch?v=missy',
                title:'Missy Elliott - Lose Control (feat. Ciara & Fat Man Scoop) [Official Music Video]',
                channel:'Missy Elliott', duration:247, album:'', artist:'', genre:'Music', categories:['Music'],
                sections:[], hasSections:false, sourceQuality:{} ,
                artistSuggestion:{artist:'Missy Elliott',confidence:'HIGH',basis:'existing-library-artist-title-prefix+source-creator-corroboration',matchingTrackCount:3,channelCorroborated:true}
              };
              inspectedDuplicates=[];
              renderImportPlan();
            }""")
            require(page.locator("#importArtist").input_value() == "Missy Elliott", "existing-library artist suggestion did not prefill override")
            text=page.locator("#importArtistSuggestion").inner_text()
            require("Matched existing library artist:" in text and "Missy Elliott" in text and "YouTube creator" in text, f"artist suggestion explanation missing: {text}")
            require(not page.locator("#importArtist").is_disabled(), "artist suggestion field became disabled")
            page.locator("#importArtist").evaluate("e => { e.value='Missy Elliott & Ciara'; e.dispatchEvent(new Event('input',{bubbles:true})); }")
            require(page.locator("#importArtist").input_value() == "Missy Elliott & Ciara", "artist suggestion must remain user-editable")
            page.evaluate("topScreen('library'); showView('songs'); search.value=''; render()")

        checked("youtube-import-prefills-high-confidence-existing-library-artist-suggestion", existing_library_artist_suggestion_contract)

        def narrow_song_projection_and_player_fit() -> None:
            page.set_viewport_size({"width": 500, "height": 760})
            page.wait_for_timeout(50)
            dims=page.evaluate("() => ({sw:document.documentElement.scrollWidth,w:innerWidth})")
            require(dims["sw"] <= dims["w"] + 1, f"narrow library overflows horizontally: {dims}")
            require(page.locator("#songsTable thead").evaluate("e => getComputedStyle(e).display === 'none'"), "narrow projection still exposes desktop table header")
            first = page.locator("#songsTable tbody tr").first
            box=first.bounding_box(); require(box is not None and box["width"] > 450, f"narrow track card did not reclaim available width: {box}")
            meta = first.locator(".song-mobile-meta")
            require(meta.evaluate("e => getComputedStyle(e).display !== 'none'") and "Artist 1" in meta.inner_text() and "Album" in meta.inner_text(), "compact track projection lost essential metadata")
            for selector in ["#prevBtn","#playBtn","#nextBtn","#shuffleToggle","#repeatBtn","#queueMiniBtn"]:
                ctl=page.locator(selector).bounding_box(); require(ctl is not None and ctl["x"] >= 0 and ctl["x"] + ctl["width"] <= 501, f"compact player control escaped viewport: {selector} {ctl}")
            page.set_viewport_size({"width": 1280, "height": 720})
            page.wait_for_timeout(50)

        checked("narrow-window-uses-track-cards-with-player-controls-in-bounds", narrow_song_projection_and_player_fit)

        def intent_first_track_actions() -> None:
            page.locator("#songsTable tbody tr").first.locator("button[aria-label='Track actions']").click()
            require(page.locator(".track-action-quick button").count() == 3, "common playback intents are not grouped as three quick actions")
            require(page.locator(".track-action-row").count() >= 5, "track actions are not presented as contextual rows")
            require(page.locator(".crop-glyph").count() == 1, "playback crop row lacks visual window glyph")
            require(page.locator("#modalActions button").count() == 1 and "Deleted" in page.locator("#modalActions").inner_text(), "track actions should have one separated destructive footer action and no duplicate Close")
            page.locator("select[aria-label='Add track to playlist']").select_option("p1")
            page.wait_for_function("document.getElementById('quickPlaylistFeedback').textContent.includes('Favorites')")
            req=page.evaluate("() => __mockRequests.filter(x=>x.path==='/api/playlists/add').at(-1)")
            require(req is not None and 't1' in (req.get('body') or ''), f"inline playlist selection did not issue the bound add request: {req}")
            page.click("#modalCloseBtn")

        checked("track-actions-are-contextual-and-playlist-selection-is-inline", intent_first_track_actions)

        def artist_cards_are_navigation_first() -> None:
            page.evaluate("""() => {
              window.__artistFixture=Array.from({length:22},(_,i)=>({
                id:`artist-${String(i+1).padStart(2,'0')}`,
                path:`/music/artist-${i+1}.mp3`,
                filename:`artist-${i+1}.mp3`,
                title:`Artist Track ${String(i+1).padStart(2,'0')}`,
                artist:'Focus Artist',
                album:`Album ${String(i+1).padStart(2,'0')}`,
                genre:i%2?'Rock':'Alternative',
                duration:180+i,
                source:'/music',
                addedAt:`2026-09-29T00:${String(i).padStart(2,'0')}:00Z`
              }));
              state.tracks=window.__artistFixture;
              openArtistName=''; artistAlbumsExpanded=false; artistAlbumsShowAll=false; artistSongQuery=''; artistAlbumFilter='';
              sortStack=[{key:'title',dir:'asc'}];
              search.value=''; genreFilter.dataset.preferred='all'; genreFilter.value='all';
              render(); showView('artists');
            }""")
            page.wait_for_selector("#artistGrid .artist-card")
            require(page.locator("#artistGrid .artist-card").count() == 1, "artist grouping did not collapse fixture into one artist card")
            require(page.locator("#artistGrid button").count() == 0, "artist browse cards still expose play/shuffle controls")
            box = page.locator("#artistGrid .artist-card").bounding_box()
            require(box is not None and abs(box["width"] - box["height"]) < 3, f"artist browse card is not a square visual surface: {box}")
            require("22 tracks" in page.locator("#artistGrid .artist-card-meta").inner_text(), "artist card lost track/album summary")
            page.locator("#artistGrid .artist-card").click()
            page.wait_for_timeout(30)
            require(not page.locator("#artistDetail").evaluate("e => e.classList.contains('hidden')"), "artist detail did not open")
            require(page.locator("#artistGrid").evaluate("e => e.classList.contains('hidden')"), "artist grid did not yield to artist detail terrain")
            require("Focus Artist" in page.locator("#artistDetail .artist-hero-copy").inner_text(), "artist hero lost identity")

        checked("artist-grid-cards-are-navigation-first", artist_cards_are_navigation_first)

        def artist_scoped_search_and_sort() -> None:
            require(page.locator("#artistSongsTable tbody tr").count() == 22, "artist detail did not expose all scoped songs")
            page.locator("#artistSongSearch").fill("Track 03")
            page.wait_for_timeout(30)
            require(page.locator("#artistSongsTable tbody tr").count() == 1, "artist mini-search did not filter the already-scoped song terrain")
            require("Artist Track 03" in page.locator("#artistSongsTable tbody tr").first.inner_text(), "artist mini-search returned the wrong song")
            page.locator("#artistSongSearch").fill("")
            page.wait_for_timeout(30)
            page.locator("#artistSongsTable th.col-title .sort-head").click()
            page.wait_for_timeout(30)
            first_title=page.locator("#artistSongsTable tbody tr").first.locator("strong.clip2").inner_text()
            require(first_title == "Artist Track 22", f"artist song sort did not reverse title order: {first_title}")
            require("Title Z→A" in " ".join(page.locator("#artistSortSummary .sort-chip").all_inner_texts()), "artist sort summary did not reflect active sort")
            page.locator("#artistSongsTable th.col-title .sort-head").click()
            page.wait_for_timeout(20)

        checked("artist-detail-searches-and-sorts-scoped-songs", artist_scoped_search_and_sort)

        def artist_album_shelf_contract() -> None:
            require(page.locator("#artistAlbumsBody").evaluate("e => e.classList.contains('hidden')"), "artist album shelf should start collapsed")
            require("22" in page.locator("#artistAlbumsToggle").inner_text(), "artist album count is missing")
            page.locator("#artistAlbumsToggle").click()
            page.wait_for_timeout(30)
            require(not page.locator("#artistAlbumsBody").evaluate("e => e.classList.contains('hidden')"), "artist album shelf did not expand")
            limit=page.evaluate("artistAlbumShelfLimit()")
            visible=page.locator("#artistAlbumsBody .artist-album-card").count()
            require(visible == min(22, limit), f"artist album shelf ignored its three-row bound: visible={visible}, limit={limit}")
            if limit < 22:
                require(page.locator("#artistAlbumsBody .artist-shelf-more button").count() == 1, "bounded shelf did not offer Show all")
            first_album=page.locator("#artistAlbumsBody .artist-album-card").first
            first_album.click()
            page.wait_for_timeout(40)
            require("Album 01" in page.locator("#artistAlbumFilter").inner_text(), "album card did not become an artist-song filter")
            require(page.locator("#artistSongsTable tbody tr").count() == 1 and "Artist Track 01" in page.locator("#artistSongsTable tbody tr").first.inner_text(), "artist album filtering did not map into the song terrain")
            page.locator("#artistAlbumFilter button").click()
            page.wait_for_timeout(30)
            require(page.locator("#artistSongsTable tbody tr").count() == 22, "clearing artist album filter did not restore scoped songs")
            page.evaluate("""() => {
              state.tracks=window.__mockTracks; openArtistName=''; artistAlbumsExpanded=false; artistAlbumsShowAll=false; artistSongQuery=''; artistAlbumFilter=''; sortStack=[{key:'title',dir:'asc'}]; render(); showView('songs');
            }""")
            page.wait_for_selector("#songsTable tbody tr")
            require(page.locator("#songsTable tbody tr").count() == 4, "artist fixture cleanup did not restore baseline library")

        checked("artist-album-shelf-is-collapsible-bounded-and-filters-songs", artist_album_shelf_contract)

        def album_reverse_artist_lattice() -> None:
            page.evaluate("showView('albums')")
            page.wait_for_selector("#albumGrid .album-open")
            require(page.locator("#albumGrid .album-open").count() == 1, "baseline shared album should render as one collection")
            require("4 artists" in page.locator("#albumGrid .album-open .small.muted").inner_text(), "multi-artist album browse summary did not expose collaborator count")
            page.locator("#albumGrid .album-open").click()
            page.wait_for_timeout(30)
            require(not page.locator("#albumDetail").evaluate("e => e.classList.contains('hidden')"), "album detail did not open")
            relation_texts = page.locator("#albumDetail .artist-relation-chip").all_inner_texts()
            require(relation_texts == ["Artist 1", "Artist 2", "Artist 3", "Artist 4"], f"album artist relations were not complete/stable: {relation_texts}")
            require("Artists / collaborators" in page.locator("#albumDetail .album-artist-relations").inner_text(), "multi-artist album did not communicate collaborator relation")
            require(page.locator("#albumDetail .album-track-artist").count() == 4, "album track rows do not expose reverse artist navigation")
            page.get_by_role("button", name="Open artist Artist 3").first.click()
            page.wait_for_timeout(40)
            route = page.evaluate("() => ({view:currentView, artist:openArtistName, album:openAlbumName})")
            require(route["view"] == "artists" and route["artist"] == "Artist 3", f"album relation did not navigate to the artist terrain: {route}")
            require(page.locator("#artistSongsTable tbody tr").count() == 1 and "Track 3" in page.locator("#artistSongsTable tbody tr").first.inner_text(), "reverse lattice arrived at the wrong artist works")
            page.evaluate("showView('albums')")
            page.wait_for_timeout(30)
            require(not page.locator("#albumDetail").evaluate("e => e.classList.contains('hidden')"), "returning to Albums lost the prior collection context")
            page.evaluate("closeAlbum(); showView('songs')")

        checked("album-detail-reaches-every-credited-artist-and-preserves-lattice-context", album_reverse_artist_lattice)

        def repeated_scope_click_returns_to_scope_root_only() -> None:
            page.evaluate("showView('albums')")
            page.wait_for_selector("#albumGrid .album-open")
            page.locator("#albumGrid .album-open").first.click()
            page.wait_for_timeout(30)
            require(page.evaluate("openAlbumName") == "Album", "album detail did not establish nested scope")
            page.locator("#librarySidebar button[data-view='albums']").click()
            page.wait_for_timeout(20)
            state1=page.evaluate("() => ({view:currentView,album:openAlbumName})")
            require(state1 == {"view":"albums","album":""}, f"Albums click did not return to album scope root: {state1}")
            page.locator("#librarySidebar button[data-view='albums']").click()
            page.wait_for_timeout(20)
            state1b=page.evaluate("() => ({view:currentView,album:openAlbumName})")
            require(state1b == {"view":"albums","album":""}, f"Albums root click escaped its own scope: {state1b}")

            page.evaluate("openArtistName=''; artistAlbumsExpanded=false; artistAlbumsShowAll=false; artistSongQuery=''; artistAlbumFilter=''; showView('artists')")
            page.wait_for_selector("#artistGrid .artist-card")
            page.locator("#artistGrid .artist-card").first.click()
            page.wait_for_timeout(30)
            require(bool(page.evaluate("openArtistName")), "artist detail did not establish nested scope")
            page.locator("#librarySidebar button[data-view='artists']").click()
            page.wait_for_timeout(20)
            state2=page.evaluate("() => ({view:currentView,artist:openArtistName})")
            require(state2 == {"view":"artists","artist":""}, f"Artists click did not return to artist scope root: {state2}")
            page.locator("#librarySidebar button[data-view='artists']").click()
            page.wait_for_timeout(20)
            state2b=page.evaluate("() => ({view:currentView,artist:openArtistName})")
            require(state2b == {"view":"artists","artist":""}, f"Artists root click escaped its own scope: {state2b}")
            page.evaluate("showView('songs')")

        checked("repeat-album-and-artist-scope-clicks-return-to-scope-root-only", repeated_scope_click_returns_to_scope_root_only)

        def now_playing_exposes_details_artist_album_and_brand_home() -> None:
            page.evaluate("setQueue(window.__mockTracks,1,true)")
            page.wait_for_timeout(30)
            require(not page.locator("#nowDetailsBtn").is_disabled(), "now-playing details control stayed disabled with a current track")
            require(page.locator("#nowMeta [data-now-artist]").count() == 1, "now-playing artist relation missing")
            require(page.locator("#nowMeta [data-now-album]").count() == 1, "now-playing album relation missing")
            page.locator("#nowDetailsBtn").click()
            require(page.locator("#modalTitle").inner_text().strip() == "Track 2", "now-playing details did not open the current track")
            require(page.locator(".track-action-list").count() == 1, "current-track detail surface is not the established track action/details contract")
            page.click("#modalCloseBtn")

            page.locator("#nowMeta [data-now-artist]").click()
            page.wait_for_timeout(30)
            artist_state=page.evaluate("() => ({view:currentView,artist:openArtistName})")
            require(artist_state == {"view":"artists","artist":"Artist 2"}, f"now-playing artist relation navigated incorrectly: {artist_state}")
            page.locator("#nowMeta [data-now-album]").click()
            page.wait_for_timeout(30)
            album_state=page.evaluate("() => ({view:currentView,album:openAlbumName})")
            require(album_state == {"view":"albums","album":"Album"}, f"now-playing album relation navigated incorrectly: {album_state}")

            page.evaluate("topScreen('discover')")
            require(not page.locator("#discoverScreen").evaluate("e => e.classList.contains('hidden')"), "Discover did not open for brand-home test")
            page.click("#brandHome")
            page.wait_for_timeout(20)
            home_state=page.evaluate("() => ({view:currentView,album:openAlbumName,artist:openArtistName,libraryHidden:libraryScreen.classList.contains('hidden')})")
            require(home_state == {"view":"home","album":"","artist":"","libraryHidden":False}, f"VexStream brand did not return to clean Library home: {home_state}")
            page.evaluate("showView('songs')")

        checked("now-playing-links-to-details-artist-album-and-brand-returns-home", now_playing_exposes_details_artist_album_and_brand_home)

        def filtered_songs_are_display_lens_not_playback_universe() -> None:
            page.evaluate("() => { state.shuffle=false; state.queue=[]; state.queueIndex=-1; search.value='Track 2'; showView('songs'); render(); }")
            page.wait_for_timeout(20)
            visible = page.locator("#songsTable tbody tr")
            require(visible.count() == 1, "fixture search should expose exactly one visible song")
            visible.first.dblclick()
            page.wait_for_timeout(20)
            s = snapshot(page)
            require(s["queue"] == ["t1", "t2", "t3", "t4"], f"filtered row playback truncated full library: {s}")
            require(s["current"] == "t2" and s["queueIndex"] == 1, f"filtered row playback lost selected current: {s}")

            page.evaluate("() => { state.queue=[]; state.queueIndex=-1; state.shuffle=true; search.value='Track 3'; render(); }")
            page.wait_for_timeout(20)
            page.locator("#songsTable tbody tr").first.dblclick()
            page.wait_for_timeout(20)
            shuffled = snapshot(page)
            require(shuffled["current"] == "t3" and shuffled["queueIndex"] == 0, f"shuffle did not keep selected filtered song current: {shuffled}")
            require(len(shuffled["queue"]) == 4 and set(shuffled["queue"]) == {"t1","t2","t3","t4"}, f"shuffle omitted filtered-out library songs: {shuffled}")

            page.evaluate("() => { state.shuffle=false; state.queue=[]; state.queueIndex=-1; search.value='Track 4'; render(); }")
            page.wait_for_timeout(20)
            page.click("#playAllBtn")
            page.wait_for_timeout(20)
            require(snapshot(page)["queue"] == ["t1","t2","t3","t4"], "Play all inherited the Songs filter")

            page.evaluate("() => { state.queue=[]; state.queueIndex=-1; search.value='Track 4'; render(); }")
            page.click("#shuffleAllBtn")
            page.wait_for_timeout(20)
            q = snapshot(page)["queue"]
            require(len(q) == 4 and set(q) == {"t1","t2","t3","t4"}, f"Shuffle all inherited the Songs filter: {q}")

            page.evaluate("() => { state.queue=['t1']; state.queueIndex=0; state.shuffle=false; search.value='Track 4'; render(); }")
            page.wait_for_timeout(20)
            add = page.locator("#songsTable tbody tr").first.locator("button[aria-label='Add to queue']")
            add.click()
            page.wait_for_timeout(20)
            q = snapshot(page)["queue"]
            require(q == ["t1","t4"], f"filtered row Add to queue should append one intentional track only: {q}")

            page.evaluate("() => { search.value=''; state.shuffle=false; state.queue=[]; state.queueIndex=-1; render(); }")

        checked("songs-filter-is-display-lens-not-playback-or-queue-universe", filtered_songs_are_display_lens_not_playback_universe)

        def first_activation() -> None:
            rows.nth(0).dblclick()
            page.wait_for_timeout(30)
            s = snapshot(page)
            require(s["queue"] == ["t1", "t2", "t3", "t4"], f"library activation did not establish visible sequence: {s}")
            require(s["current"] == "t1" and s["queueIndex"] == 0, f"wrong first current track: {s}")
            require(not s["paused"] and s["src"].endswith("/media/t1"), f"first track did not begin: {s}")
            require(s["trackId"] == "t1", f"loaded-track identity missing: {s}")

        checked("double-click-starts-visible-library-sequence", first_activation)

        def midstream_switch() -> None:
            rows.nth(1).dblclick()
            page.wait_for_timeout(30)
            s = snapshot(page)
            require(s["queue"] == ["t1", "t2", "t3", "t4"], f"switch collapsed queue: {s}")
            require(s["current"] == "t2" and s["queueIndex"] == 1, f"wrong switched track: {s}")
            require(not s["paused"] and s["src"].endswith("/media/t2"), f"second track did not begin: {s}")

        checked("double-click-switches-track-midstream", midstream_switch)

        def next_control() -> None:
            page.click("#nextBtn")
            page.wait_for_timeout(20)
            s = snapshot(page)
            require(s["current"] == "t3" and s["src"].endswith("/media/t3") and not s["paused"], f"Next failed: {s}")

        checked("next-control-advances", next_control)

        def previous_control() -> None:
            page.click("#prevBtn")
            page.wait_for_timeout(20)
            s = snapshot(page)
            require(s["current"] == "t2" and s["src"].endswith("/media/t2") and not s["paused"], f"Previous failed: {s}")

        checked("previous-control-returns", previous_control)

        def play_pause_control() -> None:
            page.click("#playBtn")
            require(snapshot(page)["paused"], "Play/Pause did not pause")
            page.click("#playBtn")
            page.wait_for_timeout(20)
            require(not snapshot(page)["paused"], "Play/Pause did not resume")

        checked("play-pause-control-toggles", play_pause_control)

        def queue_timeline_persists_history() -> None:
            page.click("#queueMiniBtn")
            page.wait_for_timeout(30)
            require(page.locator("#queueDrawer").evaluate("e => e.classList.contains('open')"), "queue drawer did not open")
            labels = [" ".join(x.split()).upper() for x in page.locator("#drawerQueueList .queue-section-label").all_inner_texts()[:3]]
            require(labels == ["HISTORY 1", "CURRENT", "UP NEXT 2"], f"timeline sections do not expose history/current/future: {page.locator('#drawerQueueList').inner_text()}")
            history = page.locator("#drawerQueueList .drawer-row.history")
            current = page.locator("#drawerQueueList .drawer-row.current")
            future = page.locator("#drawerQueueList .drawer-row.future")
            require(history.count() == 1 and "Track 1" in history.inner_text(), "played track disappeared from queue history")
            require(current.count() == 1 and "Track 2" in current.inner_text() and "Current" in current.inner_text(), "current queue node is not explicit")
            require(future.count() == 2, "up-next projection is wrong")
            s = snapshot(page)
            require(s["queueCountText"] == "4" and s["queueMiniText"] == "4" and s["queueNextText"] == "2 next", f"queue counts should distinguish total from upcoming: {s}")
            require(page.locator("#songsTable tr.playing .now-playing-chip").count() == 1, "songs view does not expose a current-playing indicator")
            page.click("#queueMiniBtn")
            page.wait_for_timeout(20)
            require(not page.locator("#queueDrawer").evaluate("e => e.classList.contains('open')"), "queue drawer did not close after timeline inspection")

        checked("queue-timeline-preserves-history-current-and-up-next", queue_timeline_persists_history)

        def queue_button_feedback() -> None:
            button = page.locator("#songsTable tbody tr").nth(2).locator("button[aria-label='Add to queue']")
            before = page.evaluate("() => state.queue.length")
            button.click()
            page.wait_for_timeout(20)
            require("Queued" in button.inner_text(), "queue action gave no immediate feedback")
            require(page.evaluate("() => state.queue.length") == before + 1, "queue action did not append exactly one node")
            page.evaluate("() => removeQueue(state.queue.length-1)")

        checked("add-to-queue-action-is-explicit-and-acknowledged", queue_button_feedback)

        def album_drilldown_and_queue_scope() -> None:
            page.click("button[data-view='albums']")
            page.wait_for_selector("#albumGrid .album-open")
            page.locator("#albumGrid [data-album-open-button]").first.click()
            page.wait_for_timeout(20)
            require(not page.locator("#albumDetail").evaluate("e => e.classList.contains('hidden')"), "album detail did not open")
            require(page.locator("#albumGrid").evaluate("e => e.classList.contains('hidden')"), "album grid should yield to nested detail terrain")
            require(page.locator("#albumDetail .album-track-row").count() == 4, "album detail did not expose all collection tracks")
            require(page.locator("#albumDetail .album-track-row.playing .now-playing-chip").count() == 1, "album detail does not identify current playback")
            album_queue = page.locator("#albumDetail .album-detail-hero button.queue-action")
            album_queue.click()
            page.wait_for_timeout(20)
            require(page.evaluate("() => state.queue.length") == 8, "album queue action did not append the entire collection")
            require("queued" in album_queue.inner_text().lower(), "album queue action gave no feedback")
            page.evaluate("() => { state.queue=['t1','t2','t3','t4']; playIndex(1); }")
            page.wait_for_timeout(20)
            page.get_by_role("button", name="← All albums").click()

        checked("album-opens-to-track-terrain-and-queues-at-collection-scope", album_drilldown_and_queue_scope)

        def drag_history_into_future_preserves_current() -> None:
            page.click("button[data-view='songs']")
            if not page.locator("#queueDrawer").evaluate("e => e.classList.contains('open')"):
                page.click("#queueMiniBtn")
            page.wait_for_timeout(20)
            source = page.locator("#drawerQueueList .drawer-row.history[data-queue-index='0']")
            target = page.locator("#drawerQueueList .queue-drop-end")
            require(source.count() == 1, "expected one history node before drag")
            source.drag_to(target)
            page.wait_for_timeout(50)
            s = snapshot(page)
            require(s["queue"] == ["t2", "t3", "t4", "t1"], f"drag did not move history node to future: {s}")
            require(s["current"] == "t2" and s["queueIndex"] == 0 and s["trackId"] == "t2", f"drag lost current playback identity: {s}")
            require(page.locator("#drawerQueueList .drawer-row.current").get_attribute("data-queue-index") == "0", "current node index was not remapped after drag")
            require(page.locator("#drawerQueueList .drawer-row.future").count() == 3, "moved history node did not become future context")
            page.evaluate("() => { state.queue=['t1','t2','t3','t4']; playIndex(1); }")
            page.wait_for_timeout(20)
            page.click("#queueMiniBtn")
            page.wait_for_timeout(20)
            require(not page.locator("#queueDrawer").evaluate("e => e.classList.contains('open')"), "queue drawer did not close after drag proof")

        checked("drag-reorders-timeline-without-losing-current-node", drag_history_into_future_preserves_current)

        def shuffle_visible_order_is_playback_order() -> None:
            page.evaluate("() => { state.shuffle=false; state.queue=['t1','t2','t3','t4']; playIndex(0); }")
            page.click("#shuffleToggle")
            page.wait_for_timeout(20)
            projection = page.evaluate("() => ({queue:[...state.queue], index:state.queueIndex, current:state.queue[state.queueIndex], pressed:shuffleToggle.getAttribute('aria-pressed')})")
            require(projection["pressed"] == "true" and projection["index"] == 0 and projection["current"] == "t1", f"shuffle mode activation broke current node: {projection}")
            expected = projection["queue"][1]
            page.click("#nextBtn")
            page.wait_for_timeout(20)
            s = snapshot(page)
            require(s["current"] == expected and s["queueIndex"] == 1, f"Next skipped the visible shuffled successor: expected {expected}, got {s}")
            page.click("#shuffleToggle")
            page.evaluate("() => { state.queue=['t1','t2','t3','t4']; playIndex(1); }")
            page.wait_for_timeout(20)

        checked("shuffle-reorders-visible-future-and-next-follows-it", shuffle_visible_order_is_playback_order)

        def collapsed_hit_test() -> None:
            hit = page.evaluate(
                """() => {
                  const r=nextBtn.getBoundingClientRect();
                  const e=document.elementFromPoint(r.left+r.width/2,r.top+r.height/2);
                  return {id:e?.id||'', contained:nextBtn===e||nextBtn.contains(e), drawerOpen:queueDrawer.classList.contains('open')};
                }"""
            )
            require(not hit["drawerOpen"], f"drawer unexpectedly open: {hit}")
            require(hit["contained"], f"collapsed queue surface intercepts Next: {hit}")

        checked("collapsed-queue-does-not-cover-player-controls", collapsed_hit_test)

        def source_recovery() -> None:
            page.evaluate(
                """() => {
                  audio.pause();
                  audio.removeAttribute('src');
                  delete audio.dataset.trackId;
                  audio.load();
                }"""
            )
            page.click("#playBtn")
            page.wait_for_timeout(20)
            s = snapshot(page)
            require(s["current"] == "t2", f"current identity changed during source recovery: {s}")
            require(s["src"].endswith("/media/t2") and not s["paused"], f"Play did not recover unloaded current source: {s}")

        checked("play-recovers-current-track-after-unloaded-source", source_recovery)

        def playback_start_crop() -> None:
            page.evaluate(
                """() => {
                  const t=trackById('t1');
                  t.playbackStart=2.5;
                  t.playbackEnd=null;
                  playLibraryTrack('t1');
                }"""
            )
            page.wait_for_timeout(30)
            s = snapshot(page)
            require(s["current"] == "t1", f"crop test loaded wrong track: {s}")
            require(abs(float(s["currentTime"]) - 2.5) < 0.02, f"playback start crop not applied: {s}")
            require("✂" in s["meta"], f"crop awareness not rendered: {s}")

        checked("non-destructive-start-crop-is-applied", playback_start_crop)

        def playback_end_crop() -> None:
            page.evaluate(
                """() => {
                  const t=trackById('t1');
                  t.playbackEnd=3.0;
                  audio.currentTime=2.99;
                  audio.dispatchEvent(new Event('timeupdate'));
                }"""
            )
            page.wait_for_timeout(30)
            s = snapshot(page)
            require(s["current"] == "t2" and s["src"].endswith("/media/t2"), f"crop end did not advance queue: {s}")

        checked("non-destructive-end-crop-advances-queue", playback_end_crop)

        checked(
            "full-interaction-run-has-no-browser-runtime-error",
            lambda: require(not report["pageErrors"], f"page errors after interactions: {report['pageErrors']}"),
        )
        report["finalState"] = snapshot(page)
        report["status"] = "PASS"
        browser.close()
    return report


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--html", type=Path, default=Path(__file__).resolve().parents[1] / "ui" / "index.html")
    parser.add_argument("--browser-executable")
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()

    report_path = args.report
    try:
        report = run(args.html.resolve(), browser_executable(args.browser_executable))
    except Exception as exc:
        report = {
            "schemaVersion": "vexstream.ui-runtime-regression/v1",
            "appVersion": VERSION,
            "html": str(args.html.resolve()),
            "status": "FAIL",
            "error": f"{type(exc).__name__}: {exc}",
        }
        if report_path:
            report_path.parent.mkdir(parents=True, exist_ok=True)
            report_path.write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
        print(json.dumps(report, indent=2, ensure_ascii=False))
        return 1

    if report_path:
        report_path.parent.mkdir(parents=True, exist_ok=True)
        report_path.write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
