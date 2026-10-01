#!/usr/bin/env python3
"""VexStream Music YouTube import bridge.

Network/provider work is intentionally isolated here so the Go player remains a
dependency-free local music player. yt-dlp is imported only for search/inspect/import.
"""
from __future__ import annotations

from pathlib import Path
import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
import traceback
from datetime import datetime, timezone

VERSION = "2.1.0"
CATALOG_SCHEMA = "vexmedia.music-library/v1"


PROVIDER = "youtube"
REQUEST_SPACING_SECONDS = 2.0
PROVIDER_ATTEMPTS = 3

class ImportCancelled(Exception):
    pass

def check_cancel(cancel_path: Path|None):
    if cancel_path and cancel_path.exists():
        raise ImportCancelled("Import cancelled by user")

def configure_utf8_stdio():
    """Keep redirected Windows pipes Unicode-safe."""
    for stream in (sys.stdout, sys.stderr):
        try:
            stream.reconfigure(encoding="utf-8", errors="backslashreplace")
        except Exception:
            pass

configure_utf8_stdio()

def best_thumbnail(info: dict) -> str:
    direct=info.get("thumbnail")
    if isinstance(direct,str) and direct.strip():
        return direct.strip()
    for row in reversed(info.get("thumbnails") or []):
        if isinstance(row,dict):
            url=row.get("url")
            if isinstance(url,str) and url.strip():
                return url.strip()
    return ""

def classify_provider_error(exc: Exception, *, stage: str) -> dict:
    technical=f"{type(exc).__name__}: {exc}"
    low=technical.lower()
    problem={
        "schemaVersion":"vexstream.provider-problem/v1",
        "appVersion":VERSION,
        "provider":PROVIDER,
        "stage":stage,
        "code":"YOUTUBE_PROVIDER_ERROR",
        "title":"YouTube could not complete this request.",
        "message":"The provider returned an unexpected error.",
        "retryable":True,
        "suggestedAction":"Retry once. If it repeats, copy the diagnostic packet for your assistant.",
        "technical":technical,
        "observedAt":datetime.now(timezone.utc).isoformat(),
    }
    if "429" in low or "too many requests" in low or "rate limit" in low:
        problem.update(
            code="YOUTUBE_RATE_LIMIT",
            title="YouTube asked VexStream to slow down.",
            message="The provider is temporarily rate-limiting metadata requests.",
            suggestedAction="Wait a few minutes before retrying. VexStream already spaces requests and uses conservative retries.",
        )
    elif "403" in low or "forbidden" in low:
        problem.update(
            code="YOUTUBE_ACCESS_REJECTED",
            title="YouTube rejected this request.",
            message="The provider did not allow this public metadata request from the current connection/runtime.",
            suggestedAction="Retry later or choose another public source. VexStream does not import browser cookies or bypass access controls.",
        )
    elif "private video" in low or "video is private" in low or "unavailable" in low or "not available" in low:
        problem.update(
            code="YOUTUBE_SOURCE_UNAVAILABLE",
            title="This YouTube source is unavailable.",
            message="The source may be private, removed, region-restricted, or otherwise unavailable to the current public session.",
            retryable=False,
            suggestedAction="Choose another publicly available source.",
        )
    elif "sign in" in low or "login" in low or "age-restricted" in low:
        problem.update(
            code="YOUTUBE_AUTH_REQUIRED",
            title="This source requires access VexStream does not have.",
            message="The source appears to require sign-in, age/account access, or another authenticated session.",
            retryable=False,
            suggestedAction="Use a public source. VexStream does not copy browser sessions or bypass provider access controls.",
        )
    elif "timed out" in low or "timeout" in low or "connection" in low:
        problem.update(
            code="NETWORK_TEMPORARY",
            title="The network/provider request timed out.",
            message="The request did not complete in time.",
            suggestedAction="Check connectivity and retry. Very long videos can take longer to inspect.",
        )
    elif "ffmpeg" in low:
        problem.update(
            code="FFMPEG_REQUIRED",
            title="The local audio converter is not ready.",
            message="VexStream needs FFmpeg to create MP3 output.",
            retryable=False,
            suggestedAction="Run YouTube import setup/repair, then retry.",
        )
    elif "charmap" in low or "unicodeencodeerror" in low or "codec can't encode character" in low:
        problem.update(
            code="TEXT_ENCODING_ERROR",
            title="VexStream hit a text-encoding problem.",
            message="The source metadata contains Unicode text that the runtime could not safely emit.",
            suggestedAction="Update VexStream. This build forces UTF-8 across Windows bridge pipes.",
        )
    return problem

def retry_delay(problem: dict, attempt: int) -> float:
    if problem.get("code")=="YOUTUBE_RATE_LIMIT":
        return 8.0 if attempt == 1 else 15.0
    return 2.0 if attempt == 1 else 5.0

def provider_call(stage: str, fn):
    last=None
    for attempt in range(1,PROVIDER_ATTEMPTS+1):
        try:
            if attempt > 1:
                time.sleep(retry_delay(last or {}, attempt-1))
            return fn()
        except ImportCancelled:
            raise
        except Exception as exc:
            last=classify_provider_error(exc,stage=stage)
            if not last.get("retryable") or attempt>=PROVIDER_ATTEMPTS:
                raise
    raise RuntimeError("provider retry loop exhausted")

def provider_opts(**extra):
    opts={
        "quiet":True,
        "no_warnings":True,
        "retries":3,
        "extractor_retries":3,
        "fragment_retries":3,
        "sleep_interval_requests":REQUEST_SPACING_SECONDS,
    }
    opts.update(extra)
    return opts

def emit_problem(exc: Exception, stage: str):
    print(json.dumps(
        {"type":"error","problem":classify_provider_error(exc,stage=stage)},
        ensure_ascii=False
    ),flush=True)

WINDOWS_RESERVED = {
    "CON","PRN","AUX","NUL",
    *(f"COM{i}" for i in range(1,10)),
    *(f"LPT{i}" for i in range(1,10)),
}
TRANSLATION = str.maketrans({
    "<":"‹", ">":"›", ":":" -", '"':"'", "/":"／", "\\":"＼",
    "|":"｜", "?":"？", "*":"＊",
})

def safe_component(text: str, fallback: str = "track", max_len: int = 155) -> str:
    text=(text or "").strip()
    text=re.sub(r"[\x00-\x1f\x7f]","",text).translate(TRANSLATION)
    text=re.sub(r"\s+"," ",text).strip(" .")
    if not text or text in {".",".."}: text=fallback
    if text.upper() in WINDOWS_RESERVED: text="_"+text
    return (text[:max_len].rstrip(" .") or fallback)

def emit(kind: str, **fields):
    print(json.dumps({"type":kind, **fields}, ensure_ascii=False), flush=True)

def require_yt():
    try:
        import yt_dlp
        return yt_dlp
    except Exception as exc:
        raise RuntimeError(
            "YouTube import runtime is not ready. Install yt-dlp[default,curl-cffi] first."
        ) from exc

def require_ffmpeg() -> str:
    found=shutil.which("ffmpeg")
    if found: return found
    candidates=[]
    local=os.environ.get("LOCALAPPDATA")
    if local:
        candidates += [
            str(Path(local)/"Microsoft/WinGet/Links/ffmpeg.exe"),
            str(Path(local)/"Microsoft/WinGet/Packages"),
        ]
    for candidate in candidates:
        p=Path(candidate)
        if p.is_file(): return str(p)
        if p.is_dir():
            hits=list(p.rglob("ffmpeg.exe"))
            if hits: return str(hits[0])
    raise RuntimeError("FFmpeg was not found. VexStream can inspect/search, but importing MP3 requires FFmpeg.")

def compact_info(info: dict) -> dict:
    url=info.get("webpage_url") or info.get("original_url") or info.get("url")
    return {
        "id":info.get("id"),
        "title":info.get("title") or info.get("id") or "Untitled",
        "channel":info.get("channel") or info.get("uploader") or "",
        "channelId":info.get("channel_id") or info.get("uploader_id") or "",
        "duration":info.get("duration"),
        "url":url,
        "thumbnail":best_thumbnail(info),
    }

def source_audio_quality(info: dict) -> dict:
    """Return a bounded, inspection-only projection of the best audio candidate.

    This is intentionally *not* a universal quality score. Different codecs are
    not ranked against one another merely from bitrate. The projection gives the
    UI enough evidence to tell the user what can and cannot be compared before
    downloading another copy.
    """
    formats=info.get("formats") or []
    audio=[]
    for row in formats:
        if not isinstance(row,dict):
            continue
        acodec=str(row.get("acodec") or "").strip()
        vcodec=str(row.get("vcodec") or "").strip()
        if not acodec or acodec=="none":
            continue
        # Prefer audio-only candidates because that is what the import path asks
        # yt-dlp to retrieve. If none exist we still retain audio-bearing formats.
        audio.append((0 if vcodec in {"", "none"} else 1,row))
    if not audio:
        return {}
    def number(v):
        try: return float(v or 0)
        except (TypeError,ValueError): return 0.0
    def score(item):
        video_penalty,row=item
        return (-video_penalty,number(row.get("abr") or row.get("tbr")),number(row.get("asr")),number(row.get("audio_channels")))
    _,best=max(audio,key=score)
    bitrate=number(best.get("abr") or best.get("tbr"))
    sample_rate=number(best.get("asr"))
    channels=number(best.get("audio_channels"))
    return {
        "codec":str(best.get("acodec") or "").strip(),
        "container":str(best.get("ext") or "").strip(),
        "bitrateKbps":round(bitrate,1) if bitrate>0 else 0,
        "sampleRateHz":int(round(sample_rate)) if sample_rate>0 else 0,
        "channels":int(round(channels)) if channels>0 else 0,
        "basis":"youtube.inspect.best-audio-candidate",
    }

def cmd_search(query: str, limit: int):
    yt_dlp=require_yt()
    def perform():
        with yt_dlp.YoutubeDL(provider_opts(extract_flat="in_playlist",skip_download=True)) as ydl:
            return ydl.extract_info(f"ytsearch{limit}:{query}",download=False)
    data=provider_call("search",perform)
    rows=[]
    for entry in (data or {}).get("entries") or []:
        if not entry: continue
        row=compact_info(entry)
        if not row["url"] and row["id"]:
            row["url"]=f"https://www.youtube.com/watch?v={row['id']}"
        rows.append(row)
    print(json.dumps({"results":rows},ensure_ascii=False))

def normalize_sections(info: dict) -> list[dict]:
    rows=[]
    for idx,ch in enumerate(info.get("chapters") or [],start=1):
        start=float(ch.get("start_time") or 0)
        end=ch.get("end_time")
        if end is None: continue
        end=float(end)
        if end <= start: continue
        rows.append({
            "index":idx,
            "start":start,
            "end":end,
            "title":str(ch.get("title") or f"Track {idx}").strip(),
            "artist":"",
            "selected":True,
        })
    return rows

def cmd_inspect(url: str):
    yt_dlp=require_yt()
    def perform():
        with yt_dlp.YoutubeDL(provider_opts(skip_download=True,noplaylist=True)) as ydl:
            return ydl.extract_info(url,download=False)
    info=provider_call("inspect",perform)
    sections=normalize_sections(info)
    out={
        **compact_info(info),
        "description":info.get("description") or "",
        "tags":info.get("tags") or [],
        "categories":info.get("categories") or [],
        "artist":info.get("artist") or "",
        "album":info.get("album") or "",
        "genre":info.get("genre") or "",
        "uploadDate":info.get("upload_date") or "",
        "sections":sections,
        "hasSections":len(sections)>=2,
        "sourceQuality":source_audio_quality(info),
    }
    print(json.dumps(out,ensure_ascii=False))

def ffmpeg_mp3(source: Path, dest: Path, *, start: float|None=None, end: float|None=None,
               title: str, artist: str, album: str, genre: str, comment: str,
               cancel_path: Path|None=None):
    ff=require_ffmpeg()
    dest.parent.mkdir(parents=True,exist_ok=True)
    cmd=[ff,"-hide_banner","-loglevel","error","-y"]
    if start is not None:
        cmd += ["-ss",f"{start:.3f}"]
    cmd += ["-i",str(source)]
    if end is not None and start is not None:
        cmd += ["-t",f"{max(0,end-start):.3f}"]
    cmd += ["-vn","-codec:a","libmp3lame","-q:a","2","-id3v2_version","3",
            "-metadata",f"title={title}",
            "-metadata",f"artist={artist}",
            "-metadata",f"album={album}",
            "-metadata",f"comment={comment}",
            "-metadata",f"encoded_by=VexStream Music {VERSION}"]
    if genre: cmd += ["-metadata",f"genre={genre}"]
    cmd.append(str(dest))
    check_cancel(cancel_path)
    p=subprocess.Popen(cmd,stdout=subprocess.DEVNULL,stderr=subprocess.PIPE,text=True)
    while p.poll() is None:
        if cancel_path and cancel_path.exists():
            p.terminate()
            try: p.wait(timeout=2)
            except subprocess.TimeoutExpired: p.kill()
            raise ImportCancelled("Import cancelled while creating MP3 output")
        time.sleep(.2)
    stderr=(p.stderr.read() if p.stderr else "").strip()
    if p.returncode:
        raise RuntimeError("FFmpeg import failed: "+(stderr or f"status {p.returncode}"))

def upsert_catalog(music_dir: Path, entry: dict):
    store=music_dir/"_vexmedia"
    store.mkdir(parents=True,exist_ok=True)
    path=store/"library.jsonl"
    rows=[]
    if path.is_file():
        for line in path.read_text(encoding="utf-8",errors="replace").splitlines():
            if not line.strip(): continue
            try: rows.append(json.loads(line))
            except json.JSONDecodeError: pass
    key=(entry.get("provider"),entry.get("providerId"))
    replaced=False
    for i,row in enumerate(rows):
        if (row.get("provider"),row.get("providerId"))==key:
            rows[i]=entry;replaced=True;break
    if not replaced: rows.append(entry)
    tmp=path.with_suffix(".jsonl.tmp")
    tmp.write_text("".join(json.dumps(r,ensure_ascii=False,separators=(",",":"))+"\n" for r in rows),encoding="utf-8")
    tmp.replace(path)

def catalog_entry(*, provider_id: str, filename: str, bytes_: int, meta: dict) -> dict:
    return {
        "schemaVersion":CATALOG_SCHEMA,
        "toolVersion":VERSION,
        "indexedAt":datetime.now(timezone.utc).isoformat(),
        "provider":"youtube",
        "providerId":provider_id,
        "filename":filename,
        "relativePath":filename,
        "bytes":bytes_,
        "embeddedTagStatus":"PASS",
        "metadata":meta,
    }

def source_meta(info: dict, url: str, *, title: str, artist: str, album: str, genre: str,
                section: dict|None=None, user_overrides: dict|None=None) -> dict:
    channel=info.get("channel") or info.get("uploader") or ""
    m={
        "title":title,
        "artist":artist,
        "album":album,
        "genre":genre,
        "artistBasis":"user.override" if user_overrides and user_overrides.get("artist") else ("provider.music.artist" if info.get("artist") else "unresolved"),
        "albumBasis":"user.collection" if user_overrides and user_overrides.get("collection") else ("provider.music.album" if info.get("album") else "source.collection"),
        "genreBasis":"user.override" if user_overrides and user_overrides.get("genre") else ("provider.music.genre" if info.get("genre") else "youtube.category"),
        "channel":channel,
        "channelId":info.get("channel_id") or "",
        "channelUrl":info.get("channel_url") or info.get("uploader_url") or "",
        "uploader":info.get("uploader") or "",
        "uploaderId":info.get("uploader_id") or "",
        "categories":info.get("categories") or [],
        "tags":info.get("tags") or [],
        "description":str(info.get("description") or "")[:20000],
        "descriptionTruncated":len(str(info.get("description") or ""))>20000,
        "duration":(section["end"]-section["start"]) if section else info.get("duration"),
        "uploadDate":info.get("upload_date") or "",
        "releaseDate":info.get("release_date") or "",
        "releaseYear":info.get("release_year"),
        "language":info.get("language") or "",
        "sourceUrl":info.get("webpage_url") or url,
        "thumbnailUrl":info.get("thumbnail") or "",
        "sourceVideoId":info.get("id"),
        "sourceVideoTitle":info.get("title"),
        "sourceCreator":channel,
        "userOverrides":user_overrides or {},
    }
    if section:
        m.update({
            "sourceSectionIndex":section["index"],
            "sourceSectionTitle":section["sourceTitle"],
            "sourceStart":section["start"],
            "sourceEnd":section["end"],
        })
    return m

def cmd_import(plan_path: Path):
    plan=json.loads(plan_path.read_text(encoding="utf-8"))
    url=plan["url"]
    final_destination=Path(plan["destination"]).expanduser().resolve()
    staging=Path(plan["stagingDirectory"]).expanduser().resolve()
    cancel_path=Path(plan.get("cancelPath") or (staging/".cancel"))
    outputs_dir=staging/"outputs"
    outputs_dir.mkdir(parents=True,exist_ok=True)
    check_cancel(cancel_path)
    yt_dlp=require_yt()
    emit("progress",stage="Inspecting source",progress=3)
    def inspect_source():
        with yt_dlp.YoutubeDL(provider_opts(skip_download=True,noplaylist=True)) as ydl:
            return ydl.extract_info(url,download=False)
    info=provider_call("import.inspect",inspect_source)
    check_cancel(cancel_path)

    video_id=str(info.get("id") or "youtube")
    channel=info.get("channel") or info.get("uploader") or "YouTube creator"
    default_artist=info.get("artist") or ""
    artist_override=(plan.get("artistOverride") or "").strip()
    artist=artist_override or default_artist
    collection=(plan.get("collection") or "").strip() or info.get("album") or info.get("title") or f"YouTube • {channel}"
    genre=(plan.get("genre") or "").strip() or info.get("genre") or ((info.get("categories") or [""])[0] if info.get("categories") else "")
    mode=plan.get("mode") or "whole"

    with tempfile.TemporaryDirectory(prefix="source-",dir=staging) as td:
        td=Path(td)
        outtmpl=str(td/"source.%(ext)s")
        emit("progress",stage="Downloading source once",progress=10)
        def progress_hook(_):
            check_cancel(cancel_path)
        opts=provider_opts(
            noplaylist=True,
            format="bestaudio/best",
            outtmpl=outtmpl,
            overwrites=True,
            progress_hooks=[progress_hook],
        )
        check_cancel(cancel_path)
        time.sleep(REQUEST_SPACING_SECONDS)
        check_cancel(cancel_path)
        def download_source():
            with yt_dlp.YoutubeDL(opts) as ydl:
                downloaded=ydl.extract_info(url,download=True)
                return downloaded,Path(ydl.prepare_filename(downloaded))
        downloaded,src=provider_call("import.download",download_source)
        check_cancel(cancel_path)
        if not src.is_file():
            files=[p for p in td.iterdir() if p.is_file()]
            if not files: raise RuntimeError("Download completed but source audio file was not found.")
            src=max(files,key=lambda p:p.stat().st_size)

        outputs=[]
        user_overrides={"artist":artist_override,"collection":plan.get("collection") or "","genre":plan.get("genre") or ""}

        if mode in {"whole","both"}:
            title=(plan.get("wholeTitle") or info.get("title") or video_id).strip()
            name=f"{safe_component(title)} [{video_id}].mp3"
            dest=outputs_dir/name
            emit("progress",stage="Creating whole-track MP3",progress=45)
            ffmpeg_mp3(src,dest,title=title,artist=artist,album=collection,genre=genre,
                       comment=f"Source: {info.get('webpage_url') or url} | YouTube creator: {channel}",
                       cancel_path=cancel_path)
            meta=source_meta(info,url,title=title,artist=artist,album=collection,genre=genre,user_overrides=user_overrides)
            entry=catalog_entry(provider_id=video_id,filename=dest.name,bytes_=dest.stat().st_size,meta=meta)
            outputs.append({"relativePath":dest.relative_to(staging).as_posix(),"filename":dest.name,"catalogEntry":entry})

        sections=plan.get("sections") or []
        selected=[s for s in sections if s.get("selected",True)]
        if mode in {"split","both"} and selected:
            total=len(selected)
            for n,s in enumerate(selected,start=1):
                title=(s.get("title") or s.get("sourceTitle") or f"Track {n}").strip()
                section_artist=(s.get("artist") or "").strip() or artist
                start=float(s.get("start") or 0);end=float(s.get("end") or start)
                sid=f"{video_id}-s{int(s.get('index') or n):02d}"
                dest=outputs_dir/f"{safe_component(title)} [{sid}].mp3"
                emit("progress",stage=f"Creating track {n}/{total}: {title}",progress=50+int((n/total)*45))
                ffmpeg_mp3(src,dest,start=start,end=end,title=title,artist=section_artist,album=collection,genre=genre,
                           comment=f"Source: {info.get('webpage_url') or url} | section {start:.2f}-{end:.2f}s | YouTube creator: {channel}",
                           cancel_path=cancel_path)
                section_for_meta={
                    "index":int(s.get("index") or n),
                    "sourceTitle":s.get("sourceTitle") or title,
                    "start":start,"end":end,
                }
                meta=source_meta(info,url,title=title,artist=section_artist,album=collection,genre=genre,section=section_for_meta,user_overrides=user_overrides)
                entry=catalog_entry(provider_id=sid,filename=dest.name,bytes_=dest.stat().st_size,meta=meta)
                outputs.append({"relativePath":dest.relative_to(staging).as_posix(),"filename":dest.name,"catalogEntry":entry})

    check_cancel(cancel_path)
    manifest={"schemaVersion":"vexstream.import-stage/v1","destination":str(final_destination),"outputs":outputs}
    tmp=staging/"manifest.json.tmp"
    tmp.write_text(json.dumps(manifest,ensure_ascii=False,indent=2),encoding="utf-8")
    tmp.replace(staging/"manifest.json")
    emit("progress",stage="Prepared — waiting for you to add it",progress=100)
    print(json.dumps({"ok":True,"outputs":outputs,"staged":True},ensure_ascii=False))

def self_test():
    assert safe_component('A | B: "C"?') == "A ｜ B - 'C'？"
    fake={"chapters":[
        {"start_time":0,"end_time":10,"title":"A"},
        {"start_time":10,"end_time":20,"title":"B"},
    ]}
    rows=normalize_sections(fake)
    assert len(rows)==2 and rows[1]["start"]==10 and rows[1]["title"]=="B"
    text=json.dumps({"title":"Music 🎵 日本語 한국어"},ensure_ascii=False)
    assert "🎵" in text
    problem=classify_provider_error(RuntimeError("HTTP Error 429: Too Many Requests"),stage="inspect")
    assert problem["code"]=="YOUTUBE_RATE_LIMIT" and problem["retryable"] is True
    unavailable=classify_provider_error(RuntimeError("ERROR: [youtube] sample: This video is not available"),stage="inspect")
    assert unavailable["code"]=="YOUTUBE_SOURCE_UNAVAILABLE" and unavailable["retryable"] is False
    quality=source_audio_quality({"formats":[
        {"acodec":"none","vcodec":"avc1","tbr":800},
        {"acodec":"opus","vcodec":"none","abr":129.4,"asr":48000,"audio_channels":2,"ext":"webm"},
        {"acodec":"mp4a.40.2","vcodec":"none","abr":128,"asr":44100,"audio_channels":2,"ext":"m4a"},
    ]})
    assert quality["codec"]=="opus" and quality["bitrateKbps"]==129.4 and quality["sampleRateHz"]==48000
    print(json.dumps({"ok":True,"tests":6,"unicode":"🎵"},ensure_ascii=False))

def main():
    p=argparse.ArgumentParser()
    sub=p.add_subparsers(dest="command",required=True)
    s=sub.add_parser("search");s.add_argument("query");s.add_argument("--limit",type=int,default=8)
    i=sub.add_parser("inspect");i.add_argument("url")
    imp=sub.add_parser("import");imp.add_argument("plan")
    sub.add_parser("self-test")
    a=p.parse_args()
    if a.command=="search":cmd_search(a.query,a.limit)
    elif a.command=="inspect":cmd_inspect(a.url)
    elif a.command=="import":cmd_import(Path(a.plan))
    else:self_test()

if __name__=="__main__":
    try:
        main()
    except ImportCancelled as exc:
        emit("cancelled",message=str(exc))
        raise SystemExit(2)
    except Exception as exc:
        stage=sys.argv[1] if len(sys.argv)>1 else "bridge"
        emit_problem(exc,stage)
        traceback.print_exc(file=sys.stderr)
        raise
