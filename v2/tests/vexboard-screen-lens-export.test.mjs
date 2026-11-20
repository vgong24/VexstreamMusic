// [VXG RealForever]
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { buildVexStreamVexBoardLens, readCurrentSources, stableJSON } from "../projection/vexboard-screen-lens.mjs";
import { PhoneConnectionIntent } from "../web/features/phone/connection.js";
import { SystemStatusIntent } from "../web/features/system/status.js";

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

test("every source-stable control/action is addressable in the exported inert view and Lens tree",()=>{
  const live=fs.readFileSync(path.join(V2,"web","index.html"),"utf8");
  const lens=JSON.parse(committed);
  const view=lens.screens[0].view;
  const identities=[...live.matchAll(/data-vex-(?:control|action)="([^"]+)"/g)].map(match=>match[1]);
  assert.equal(new Set(identities).size,identities.length,"stable source identities must be unique within the bounded shell");
  const elementRefs=new Set(lens.screens[0].elements.map(element=>element.ref));
  for(const identity of identities){
    assert.match(view.html,new RegExp('data-vex-ref="element\\.'+identity.replace(/[.*+?^$()|[\\]{}]/g,"\\$&")+'"'));
    assert.ok(elementRefs.has("element."+identity),identity+" is addressable in the Lens tree");
  }
});

test("phone and System shell controls expose full admitted pathways rather than silent pixels",()=>{
  const lens=JSON.parse(committed);
  const phone=lens.screens[0].elements.find(element=>element.ref==="element."+PhoneConnectionIntent.Open);
  const system=lens.screens[0].elements.find(element=>element.ref==="element."+SystemStatusIntent.Open);
  assert.equal(phone?.pathwayRef,"pathway.vexstream.phone.connection.open");
  assert.equal(system?.pathwayRef,"pathway.vexstream.system.status.open");
  assert.ok(lens.pathways.find(pathway=>pathway.pathwayRef===phone.pathwayRef)?.nodes.some(node=>node.role==="CONNECTION"));
  assert.ok(lens.pathways.find(pathway=>pathway.pathwayRef===system.pathwayRef)?.nodes.some(node=>node.role==="PROJECTION"));
});
