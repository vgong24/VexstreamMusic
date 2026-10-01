# VexStream Music 2.0.1 release notes

## Windows YouTube import setup no longer fails on an already-installed FFmpeg package

2.0.0 could surface a large `powershell.exe failed` message when WinGet reported that `Gyan.FFmpeg` was already installed and had no newer version available. The setup path also depended too heavily on the current process PATH to rediscover FFmpeg.

2.0.1 repairs that boundary:

- Python/yt-dlp provisioning and FFmpeg provisioning are separate effects.
- Windows FFmpeg discovery includes WinGet's nested `Gyan.FFmpeg_*` package directories, not only PATH/WinGet Links.
- A WinGet no-upgrade result is not promoted into loss of YouTube search readiness.
- If FFmpeg is still genuinely unavailable, setup returns a partial-ready state instead of a misleading provider failure.
- `Download & add` checks `importReady` first and holds the download at setup rather than issuing an import request that must fail with `FFMPEG_REQUIRED`.
- The Discover UI now explicitly says when search is ready but MP3 download still needs FFmpeg.

## Preserved behavior

The source-native 2.0.0 playback correction remains unchanged: Songs filtering is visual, Songs playback uses the full library, and explicit queue actions remain scoped to user intent.
