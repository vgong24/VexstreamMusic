# Pull request projection — VexStream Music 2.2.0

## Purpose

Advance the existing source-native VexStream PR #1 from the working 2.0.1 import-runtime baseline to Explore Session v1 multi-axis refinement.

## 2.2.0 change

- temporary `🌐 Explore from this song` branching from local tracks;
- metadata-first, search-derived candidate frames;
- one visible YouTube provider preview at a time;
- library-match annotations using existing duplicate/provenance logic;
- explicit Add-to-library handoff into the existing import review;
- branch/back/clear lifecycle in browser memory;
- no normal queue mutation from metadata or preview;
- deterministic seam for later AI ranking without making AI required.

## Gate

Keep draft until the target Windows host confirms that provider preview, branch/back, leaving/resuming discovery, exact local-match behavior, and Add-to-library handoff work in ordinary use. Embedded provider availability and advertising remain provider-controlled.
