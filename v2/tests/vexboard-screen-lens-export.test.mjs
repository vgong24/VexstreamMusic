// [VXG RealForever]
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { buildVexStreamVexBoardLens, readCurrentSources, stableJSON } from "../projection/vexboard-screen-lens.mjs";

const HERE=path.dirname(fileURLToPath(import.meta.url));
const V2=path.resolve(HERE,"..");
const lensPath=path.join(V2,"projection","vexboard-screen-lens.json");
const committed=fs.readFileSync(lensPath,"utf8");

test("committed VexBoard Screen Lens regenerates byte-for-byte from current VexStream source",()=>{
  assert.equal(committed,stableJSON(buildVexStreamVexBoardLens(readCurrentSources())));
});

test("exported SOURCE_VIEW derives generic anchors without mutating live VexStream identity",()=>{
  const live=fs.readFileSync(path.join(V2,"web","index.html"),"utf8");
  const lens=JSON.parse(committed);
  const view=lens.screens[0].view;
  assert.doesNotMatch(live,/data-vex-ref=/);
  assert.match(view.html,/data-vex-ref="element\.vexstream\.player\.previous"/);
  assert.match(view.html,/data-vex-ref="element\.vexstream\.playback\.play-pause"/);
  assert.match(view.html,/data-vex-ref="element\.vexstream\.queue\.next"/);
  assert.equal(view.scriptPolicy,"NONE");
  assert.equal(view.anchors.elementAttribute,"data-vex-ref");
});

test("first slice keeps Player composition distinct from Playback and Queue Family ownership",()=>{
  const lens=JSON.parse(committed);
  const previous=lens.pathways.find(row=>row.pathwayRef==="pathway.vexstream.player.previous");
  const playback=lens.pathways.find(row=>row.pathwayRef==="pathway.vexstream.playback.play-pause");
  const next=lens.pathways.find(row=>row.pathwayRef==="pathway.vexstream.queue.next");
  assert.equal(previous.disposition,"DIRECT_REGISTRATION");
  assert.equal(playback.disposition,"SCAE_FAMILY");
  assert.equal(next.disposition,"SCAE_FAMILY");
  assert.ok(previous.nodes.some(node=>node.label==="Player.previous()"));
  assert.ok(playback.nodes.some(node=>node.role==="CONNECTION"));
});
