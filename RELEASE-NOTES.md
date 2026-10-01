# VexStream Music 2.0.0 release notes

## Songs filter no longer truncates playback

- Search and genre filtering continue to control what the Songs view displays.
- Playing a filtered result now uses the full sorted library as the playback timeline.
- Shuffle keeps the selected song current and shuffles every other library song into future playback.
- Play all and Shuffle all ignore the display filter and operate on the full library.
- Add to queue remains an intentional one-track action; bulk queue remains explicit-selection scoped.

## Source-first recovery

2.0.0 is built from the recovered exact 1.1.5 source rather than modifying a stripped executable. The startup session comparison, `/health` version, embedded UI badge, diagnostics, bridge, and artifact filenames are therefore all generated from the same 2.0.0 source version.

## Stronger playback qualification

A new real-media server regression launches the actual Go server, scans a real synthetic MP3 fixture, verifies exact full-file bytes, and verifies `206 Partial Content` byte-range behavior. This complements—not replaces—the deterministic Chromium state-machine suite and target-host listening acceptance.
