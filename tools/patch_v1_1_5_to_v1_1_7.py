#!/usr/bin/env python3
from __future__ import annotations
import argparse, hashlib
from pathlib import Path

SOURCE_SHA256 = "e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f"
SOURCE_HTML_SHA256 = "fadc33d2b16d3a9582cc84bcb701395876dfa7cc20d48e0d7b0ed7a339ab65fd"
OLD_VERSION = b"1.1.5"
NEW_VERSION = b"1.1.7"
HTML_START = b'<html lang="en">'
HTML_END = b'</html>'
LAUNCH_GUARD_115 = bytes.fromhex("4883fb05752c8138312e312e752480780435751e")
LAUNCH_GUARD_117 = bytes.fromhex("4883fb05752c8138312e312e752480780437751e")
EXPECTED_LAUNCH_GUARD_OFFSET = 0x2CD120
EXPECTED_OUTSIDE_UI_DIFF_OFFSETS = {0x2CD131, 0x39C751, 0x3BDFCA}
OLD_PLAY_LIBRARY = "function playLibraryTrack(id){const context=currentView==='home'?recentTracks(8):sortedSongTracks(),i=context.findIndex(t=>t.id===id);if(i>=0){setQueue(context,i,true);return}playTrack(id)}"
NEW_PLAY_LIBRARY = "function playLibraryTrack(id){let c=currentView==='home'?recentTracks(8):currentView==='songs'?sortTrackList(state.tracks):sortedSongTracks(),i=c.findIndex(t=>t.id===id);if(i>=0){if(currentView==='songs'&&state.shuffle)c=[c[i],...c.slice(0,i),...c.slice(i+1)],i=0;setQueue(c,i,true);return}playTrack(id)}"
OLD_TOGGLE_PLAYBACK = "function togglePlayback(){if(!audio.paused){audio.pause();return}const t=currentTrack();if(!t){const context=sortedSongTracks();if(context.length)setQueue(context,0,true);return}if(audio.dataset.trackId!==t.id||!audio.currentSrc){playIndex(state.queueIndex);return}audio.play().catch(()=>{})}"
NEW_TOGGLE_PLAYBACK = "function togglePlayback(){if(!audio.paused){audio.pause();return}const t=currentTrack();if(!t){const c=currentView==='songs'?sortTrackList(state.tracks):sortedSongTracks();if(c.length)setQueue(c,0,true);return}if(audio.dataset.trackId!==t.id||!audio.currentSrc){playIndex(state.queueIndex);return}audio.play().catch(()=>{})}"
OLD_SHUFFLE_ALL = "bindClick('shuffleAllBtn',()=>setQueue(shuf(sortedSongTracks()),0,true));"
NEW_SHUFFLE_ALL = "bindClick('shuffleAllBtn',()=>setQueue(shuf(sortTrackList(state.tracks)),0,true));"
OLD_PLAY_ALL = "bindClick('playAllBtn',()=>setQueue(sortedSongTracks(),0,true));"
NEW_PLAY_ALL = "bindClick('playAllBtn',()=>setQueue(sortTrackList(state.tracks),0,true));"
OLD_UI_VERSION = '<span class="version">1.1.5</span>'
NEW_UI_VERSION = '<span class="version">1.1.7</span>'

def sha256(data): return hashlib.sha256(data).hexdigest()

def extract_html(data):
    if data.count(HTML_START) != 1 or data.count(HTML_END) != 1:
        raise SystemExit("expected exactly one embedded HTML document")
    start = data.index(HTML_START)
    return start, data.index(HTML_END,start)+len(HTML_END), data[start:data.index(HTML_END,start)+len(HTML_END)]

def trim_inert_leading_whitespace(html, bytes_to_remove):
    head, marker, tail = html.partition("<script>")
    if not marker: raise SystemExit("embedded UI is missing expected <script> marker")
    remaining=bytes_to_remove; out=[]
    for line in head.splitlines(keepends=True):
        if remaining and line.startswith(" "):
            removable=len(line)-len(line.lstrip(" ")); take=min(removable,remaining)
            line=line[take:]; remaining-=take
        out.append(line)
    if remaining: raise SystemExit(f"not enough inert pre-script indentation to remove {bytes_to_remove} bytes")
    return "".join(out)+marker+tail

def patch_html(html_bytes):
    if sha256(html_bytes) != SOURCE_HTML_SHA256:
        raise SystemExit("embedded HTML digest does not match the qualified 1.1.5 baseline")
    html=html_bytes.decode("utf-8")
    replacements=[
      (OLD_PLAY_LIBRARY,NEW_PLAY_LIBRARY,"playLibraryTrack"),
      (OLD_TOGGLE_PLAYBACK,NEW_TOGGLE_PLAYBACK,"togglePlayback"),
      (OLD_SHUFFLE_ALL,NEW_SHUFFLE_ALL,"Shuffle all binding"),
      (OLD_PLAY_ALL,NEW_PLAY_ALL,"Play all binding"),
      (OLD_UI_VERSION,NEW_UI_VERSION,"UI version badge"),
    ]
    patched=html
    for old,new,label in replacements:
        if patched.count(old)!=1: raise SystemExit(f"{label} baseline not found exactly once")
        patched=patched.replace(old,new,1)
    delta=len(patched.encode("utf-8"))-len(html_bytes)
    if delta<0: patched=patched.replace("</html>",(" "*-delta)+"</html>",1)
    elif delta>0: patched=trim_inert_leading_whitespace(patched,delta)
    out=patched.encode("utf-8")
    if len(out)!=len(html_bytes): raise SystemExit("embedded HTML length changed")
    for old,new,label in replacements[:-1]:
        if old.encode() in out or new.encode() not in out: raise SystemExit(f"{label} patch incomplete")
    return out

def patch_exe(source,output):
    data=source.read_bytes()
    if sha256(data)!=SOURCE_SHA256: raise SystemExit("source SHA-256 mismatch")
    if data.count(OLD_VERSION)!=3: raise SystemExit("expected exactly three contiguous 1.1.5 version markers")
    if data.count(LAUNCH_GUARD_115)!=1 or data.find(LAUNCH_GUARD_115)!=EXPECTED_LAUNCH_GUARD_OFFSET:
        raise SystemExit("compiled 1.1.5 launch guard missing, duplicated, or moved")
    start,end,html=extract_html(data)
    out=data[:start]+patch_html(html)+data[end:]
    if out.count(OLD_VERSION)!=2: raise SystemExit("expected two non-UI 1.1.5 markers after UI patch")
    out=out.replace(OLD_VERSION,NEW_VERSION)
    if out.count(LAUNCH_GUARD_115)!=1: raise SystemExit("qualified launch guard no longer unique")
    out=out.replace(LAUNCH_GUARD_115,LAUNCH_GUARD_117,1)
    if len(out)!=len(data) or not out.startswith(b"MZ"): raise SystemExit("binary structure changed")
    if out.count(OLD_VERSION)!=0 or out.count(NEW_VERSION)<3: raise SystemExit("contiguous version patch incomplete")
    if out.count(LAUNCH_GUARD_115)!=0 or out.count(LAUNCH_GUARD_117)!=1: raise SystemExit("launch guard patch incomplete")
    diffs={i for i,(a,z) in enumerate(zip(data,out)) if a!=z and not(start<=i<end)}
    if diffs!=EXPECTED_OUTSIDE_UI_DIFF_OFFSETS: raise SystemExit(f"unexpected non-UI byte diffs: {[hex(i) for i in sorted(diffs)]}")
    if any(data[i]!=ord("5") or out[i]!=ord("7") for i in diffs): raise SystemExit("non-UI diff was not 5->7")
    output.write_bytes(out)
    return sha256(data),sha256(out)

def main():
    p=argparse.ArgumentParser()
    p.add_argument("source",type=Path); p.add_argument("output",type=Path); a=p.parse_args()
    src,dst=patch_exe(a.source,a.output)
    print(f"source_sha256={src}\noutput_sha256={dst}\noutput={a.output}")
    return 0

if __name__=="__main__": raise SystemExit(main())
