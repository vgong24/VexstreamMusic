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
    web=root/"v2"/"web";html=(web/"index.html").read_text(encoding="utf-8");css=(web/"styles.css").read_text(encoding="utf-8");fixture=(root/"tests"/"fixtures"/"vexstream-test-tone.mp3").read_bytes();media=as_data_url("audio/mpeg",fixture);app=bundle_module(web/"app.js",{})
    html=html.replace('<link rel="stylesheet" href="./styles.css">',f"<style>{css}</style>")
    html=html.replace('<script type="module" src="./app.js"></script>',f'<script>window.__VEXSTREAM_V4_MEDIA_URL={json.dumps(media)}</script><script type="module">import {json.dumps(app)};</script>')
    return html

def main()->int:
    parser=argparse.ArgumentParser();parser.add_argument("--root",type=Path,default=Path(__file__).resolve().parents[2]);parser.add_argument("--browser-executable");args=parser.parse_args();root=args.root.resolve();require((root/"tests"/"fixtures"/"vexstream-test-tone.mp3").is_file(),"media fixture missing")
    report={"schemaVersion":"vexstream.v4.playback-browser-regression/v1","checks":[],"pageErrors":[],"console":[]}
    def checked(name,fn):fn();report["checks"].append({"name":name,"status":"PASS"})
    with sync_playwright() as p:
        launch={"headless":True,"args":["--no-sandbox","--autoplay-policy=no-user-gesture-required"]};executable=browser_executable(args.browser_executable)
        if executable:launch["executable_path"]=executable
        browser=p.chromium.launch(**launch);page=browser.new_page(viewport={"width":900,"height":700});page.on("pageerror",lambda e:report["pageErrors"].append(str(e)));page.on("console",lambda m:report["console"].append({"type":m.type,"text":m.text}));page.set_content(assembled_html(root),wait_until="load");page.wait_for_function("window.vexV4?.ready===true")
        checked("real-browser-loads-v4-playback-surface",lambda:require(not report["pageErrors"],f"page errors: {report['pageErrors']}"))
        checked("stable-screen-and-control-identity",lambda:require(page.locator('[data-vex-screen="vexstream.desktop.playback"]').count()==1 and page.locator('[data-vex-control="vexstream.playback.play-pause"]').count()==1,"semantic identity missing"))
        journey=page.evaluate("window.vexV4.journey")
        checked("feature-owned-journey-is-browser-addressable",lambda:require(journey["ref"]=="journey.vexstream.desktop.playback.basic" and len(journey["steps"])==3,"journey drift"))
        button=page.locator('[data-vex-control="vexstream.playback.play-pause"]')
        checked("localized-play-projection-is-initial",lambda:require(button.get_attribute("data-text-ref")=="playback.play" and button.inner_text()=="Play","initial Play projection drift"))
        for index,step in enumerate(journey["steps"]):
            if step["kind"]!="activate":continue
            button.click();expected=step["expectTextRef"]
            page.wait_for_function("expected => document.querySelector('[data-vex-control=\"vexstream.playback.play-pause\"]').dataset.textRef === expected",arg=expected)
            checked(f"journey-step-{index}-projects-{expected}",lambda expected=expected:require(button.get_attribute("data-text-ref")==expected,f"expected {expected}"))
        checked("real-html-media-provider-played-and-paused",lambda:require(page.evaluate("window.vexV4.media.paused") is True and page.evaluate("window.vexV4.media.currentSrc.startsWith('data:audio/mpeg')"),"real media provider did not complete Play/Pause"))
        checked("browser-finished-without-runtime-errors",lambda:require(not report["pageErrors"],f"page errors: {report['pageErrors']}"));browser.close()
    report["status"]="PASS";print(json.dumps(report,indent=2));return 0
if __name__=="__main__":raise SystemExit(main())
