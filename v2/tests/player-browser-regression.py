#!/usr/bin/env python3
from __future__ import annotations
import argparse, base64, json, os, re, shutil
from pathlib import Path
from playwright.sync_api import sync_playwright

def require(condition: bool,message: str)->None:
    if not condition: raise AssertionError(message)

def browser_executable(explicit: str|None)->str|None:
    if explicit:return explicit
    if os.environ.get("VEXSTREAM_CHROMIUM"):return os.environ["VEXSTREAM_CHROMIUM"]
    for name in ("chromium","chromium-browser","google-chrome","chrome"):
        found=shutil.which(name)
        if found:return found
    return None

def as_data_url(mime: str,data: bytes)->str:
    return f"data:{mime};base64,{base64.b64encode(data).decode('ascii')}"

def bundle_module(path: Path,cache: dict[Path,str])->str:
    path=path.resolve()
    if path in cache:return cache[path]
    source=path.read_text(encoding="utf-8")
    pattern=re.compile(r'(?P<prefix>\bfrom\s+|\bimport\s+)(?P<quote>["\'])(?P<spec>\.[^"\']+)(?P=quote)')
    def replace(match: re.Match[str])->str:
        target=(path.parent/match.group("spec")).resolve()
        url=bundle_module(target,cache)
        return f'{match.group("prefix")}{match.group("quote")}{url}{match.group("quote")}'
    source=pattern.sub(replace,source)
    url=as_data_url("text/javascript;charset=utf-8",source.encode("utf-8"));cache[path]=url;return url

def assembled_html(root: Path)->str:
    web=root/"v2"/"web"
    html=(web/"index.html").read_text(encoding="utf-8")
    css=(web/"styles.css").read_text(encoding="utf-8")
    fixture=(root/"tests"/"fixtures"/"vexstream-test-tone.mp3").read_bytes()
    media=as_data_url("audio/mpeg",fixture)
    app=bundle_module(web/"app.js",{})
    html=html.replace('<link rel="stylesheet" href="./styles.css">',f"<style>{css}</style>")
    inject=f'<script>window.__VEXSTREAM_V4_MEDIA_URL={json.dumps(media)};window.__VEXSTREAM_V4_SHUFFLE=(values)=>[...values].reverse()</script>'
    html=html.replace('<script type="module" src="./app.js"></script>',inject+f'<script type="module">import {json.dumps(app)};</script>')
    return html

def main()->int:
    parser=argparse.ArgumentParser()
    parser.add_argument("--root",type=Path,default=Path(__file__).resolve().parents[2])
    parser.add_argument("--browser-executable")
    args=parser.parse_args()
    root=args.root.resolve()
    require((root/"tests"/"fixtures"/"vexstream-test-tone.mp3").is_file(),"media fixture missing")
    report={"schemaVersion":"vexstream.v4.player-browser-regression/v1","checks":[],"pageErrors":[],"console":[]}
    def checked(name,fn):fn();report["checks"].append({"name":name,"status":"PASS"})
    with sync_playwright() as p:
        launch={"headless":True,"args":["--no-sandbox","--autoplay-policy=no-user-gesture-required"]}
        executable=browser_executable(args.browser_executable)
        if executable:launch["executable_path"]=executable
        browser=p.chromium.launch(**launch)
        page=browser.new_page(viewport={"width":1000,"height":840})
        page.on("pageerror",lambda e:report["pageErrors"].append(str(e)))
        page.on("console",lambda m:report["console"].append({"type":m.type,"text":m.text}))
        page.set_content(assembled_html(root),wait_until="load")
        page.wait_for_function("window.vexV4?.ready===true")
        checked("real-browser-loads-v4-player-surface",lambda:require(not report["pageErrors"],f"page errors: {report['pageErrors']}"))
        checked("stable-player-and-control-identities",lambda:require(page.locator('[data-vex-screen="vexstream.desktop.player-practicum"]').count()==1 and page.locator('[data-vex-control="vexstream.queue.next"]').count()==1,"semantic identity missing"))
        journey=page.evaluate("window.vexV4.journey")
        checked("feature-owned-player-journey-is-addressable",lambda:require(journey["ref"]=="journey.vexstream.desktop.player.queue-basics" and len(journey["steps"])==6,"journey drift"))

        play=page.locator('[data-vex-control="vexstream.playback.play-pause"]')
        nxt=page.locator('[data-vex-control="vexstream.queue.next"]')
        prev=page.locator('[data-vex-control="vexstream.queue.previous"]')
        shuf=page.locator('[data-vex-control="vexstream.queue.shuffle"]')
        rep=page.locator('[data-vex-control="vexstream.queue.repeat"]')

        checked("top-level-play-is-enabled-before-queue-exists",lambda:require(play.is_enabled() and play.get_attribute("data-text-ref")=="playback.play","explicit seed-context Play missing"))
        play.click()
        page.wait_for_function("window.vexV4.queue.currentProjection.state.kind==='positioned' && window.vexV4.media.paused===false")
        checked("play-seeds-three-track-context-and-starts-first",lambda:require(page.evaluate("window.vexV4.queue.currentProjection.state.items.length") == 3 and page.evaluate("window.vexV4.player.currentItem.ref")=="media.vexstream.test-tone.1" and play.get_attribute("data-text-ref")=="playback.pause","Play composition drift"))

        nxt.click()
        page.wait_for_function("window.vexV4.player.currentItem.ref==='media.vexstream.test-tone.2'")
        checked("next-advances-queue-and-keeps-playback-active",lambda:require(page.evaluate("window.vexV4.media.paused") is False and play.get_attribute("data-text-ref")=="playback.pause","Next composition drift"))

        page.evaluate("window.vexV4.media.currentTime=0")
        prev.click()
        page.wait_for_function("window.vexV4.player.currentItem.ref==='media.vexstream.test-tone.1'")
        checked("previous-near-start-moves-to-prior-queue-item",lambda:require(page.evaluate("window.vexV4.player.currentItem.ref")=="media.vexstream.test-tone.1","Previous composition drift"))

        shuf.click()
        checked("shuffle-selects-and-preserves-current-item",lambda:require(shuf.get_attribute("aria-pressed")=="true" and page.evaluate("window.vexV4.queue.currentProjection.state.items.join(',')")=="media.vexstream.test-tone.1,media.vexstream.test-tone.3,media.vexstream.test-tone.2","shuffle upcoming law drift"))

        rep.click()
        checked("repeat-is-queue-truth-not-playback-truth",lambda:require(rep.get_attribute("aria-pressed")=="true" and page.evaluate("window.vexV4.queue.currentProjection.state.repeat") is True and page.evaluate("window.vexV4.playback.currentProjection.state.repeat") is None,"repeat ownership drift"))

        page.locator('[data-track-ref="media.vexstream.test-tone.3"]').click()
        page.wait_for_function("window.vexV4.player.currentItem.ref==='media.vexstream.test-tone.3'")
        nxt.click()
        page.wait_for_function("window.vexV4.player.currentItem.ref==='media.vexstream.test-tone.1'")
        checked("repeat-wraps-next-from-end-to-first",lambda:require(page.evaluate("window.vexV4.player.currentItem.ref")=="media.vexstream.test-tone.1","repeat wrap drift"))
        checked("queue-render-marks-one-current-item",lambda:require(page.locator("#queueList li.current").count()==1,"queue current projection drift"))
        checked("browser-finished-without-runtime-errors",lambda:require(not report["pageErrors"] and not [x for x in report["console"] if x["type"]=="error"],f"errors: {report}"))
        browser.close()

    report["status"]="PASS"
    print(json.dumps(report,indent=2))
    return 0

if __name__=="__main__":raise SystemExit(main())
