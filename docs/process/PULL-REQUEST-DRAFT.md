# Pull request projection — VexStream Music 2.0.0 source-native recovery

`[VXG RealForever]`

## Summary

Promote the exact recovered 1.1.5 buildable source lineage into the public VexStream Music repository, form 2.0.0 from source, repair the Songs-filter playback-scope defect, and add a real server/media regression gate.

## Source baseline

```text
VexStreamMusic-1.1.5-Source.zip
sha256=15d54909bd9227d4e0ef67f4879503d7e7fdf80366781f8ec2723a0d375c1243

reproduced historical Windows EXE
sha256=e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f
```

## 2.0.0 change

- search/genre filters remain Songs display lenses;
- Songs playback resolves against the full sorted library;
- shuffle keeps the selected row current and includes every other library track exactly once;
- Play all and Shuffle all operate on the full library;
- Add to queue remains single-track; bulk queue remains explicit-selection scoped;
- full 1.1.5 product behavior remains source-derived rather than reconstructed from the stripped EXE;
- real HTTP/media proof is added beside the deterministic Chromium fixture suite.

## Preserved boundaries

No VexLife federation, network exposure, friend sharing, AI music generation, YouTube publication, or unrelated interaction redesign is introduced by 2.0.0.
