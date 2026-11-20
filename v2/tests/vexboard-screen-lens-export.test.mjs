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

test("every source-stable relationship is addressable in the exported inert view and Lens tree",()=>{
  const live=fs.readFileSync(path.join(V2,"web","index.html"),"utf8");
  const lens=JSON.parse(committed);
  const view=lens.screens[0].view;
  const relationPatterns=[
    ["data-vex-control",(value)=>value],
    ["data-vex-action",(value)=>value],
    ["data-shell-view",(value)=>"vexstream.shell.view."+value],
    ["data-library-mode",(value)=>"vexstream.library.mode."+value],
    ["data-discover-mode",(value)=>"vexstream.discover.mode."+value],
    ["data-phone-action",(value)=>"vexstream.phone.action."+value],
    ["data-system-action",(value)=>"vexstream.system.action."+value],
    ["data-track-action",(value)=>"vexstream.track.action."+value],
  ];
  const identities=[];
  for(const tag of live.matchAll(/<[A-Za-z][^>]*>/g)){
    for(const [attribute,identityFor] of relationPatterns){
      const match=tag[0].match(new RegExp("\\b"+attribute+'="([^"]+)"'));
      if(!match)continue;
      identities.push(identityFor(match[1]));
      break;
    }
  }
  const elementRefs=new Set(lens.screens[0].elements.map(element=>element.ref));
  for(const identity of new Set(identities)){
    assert.ok(view.html.includes('data-vex-ref="element.'+identity+'"'),identity+" is anchored in the inert view");
    assert.ok(elementRefs.has("element."+identity),identity+" is addressable in the Lens tree");
  }
  assert.ok((view.html.match(/data-vex-ref="element\.vexstream\.shell\.view\.home"/g)??[]).length>=3,
    "repeated Home affordances share one relationship ref instead of inventing separate semantics");
  assert.ok(elementRefs.has("element.vexstream.player.play-next"),
    "a stronger data-vex-action identity wins over the same node's generic data-track-action");
  assert.equal(elementRefs.has("element.vexstream.track.action.play-next"),false,
    "one physical control does not receive two competing Lens identities");
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


test("Library alternate terrain uses source-authored nested selections without Product JavaScript",()=>{
  const lens=JSON.parse(committed);
  const screen=lens.screens[0];
  const shell=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.shell.primary-terrain");
  const mode=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.library.mode");
  assert.equal(shell.kind,"SELECTION");
  assert.equal(shell.exhaustive,false);
  assert.equal(shell.defaultRef,"vexstream.desktop.library.panel.home");
  const shellTargets=shell.choices.map(choice=>choice.targetRef);
  assert.ok(shellTargets.includes("vexstream.desktop.library.panel.home"));
  assert.ok(shellTargets.includes("vexstream.desktop.library.panel.library"));
  assert.equal(new Set(shellTargets).size,shellTargets.length,"Shell terrain refs stay unique as additional terrains are admitted");
  assert.equal(mode.kind,"SELECTION");
  assert.deepEqual(mode.choices.map(choice=>choice.targetRef),[
    "vexstream.desktop.library.mode-state.songs",
    "vexstream.desktop.library.mode-state.albums",
    "vexstream.desktop.library.mode-state.artists",
  ]);
  assert.equal(mode.gate.presenceRef,shell.ref);
  assert.equal(screen.view.scriptPolicy,"NONE");
  assert.ok(screen.view.stateBindings.some(binding=>binding.presenceRef===mode.ref&&binding.value==="vexstream.desktop.library.mode-state.albums"));
  assert.match(screen.view.html,/data-shell-panel="library"[^>]*data-vex-structural-ref="vexstream.desktop.library.panel.library"/);
  assert.match(screen.view.html,/id="trackChoices"[^>]*data-vex-structural-ref="vexstream.desktop.library.track-choices"/);
});

test("Discover alternate terrain uses source-authored nested selection without Product JavaScript",()=>{
 const lens=JSON.parse(committed);
 const screen=lens.screens[0];
 const shell=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.shell.primary-terrain");
 const discover=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.discover.mode");
 assert.equal(screen.view.scriptPolicy,"NONE");
 assert.ok(shell.choices.some(choice=>choice.targetRef==="vexstream.desktop.library.panel.discover"));
 assert.equal(discover.kind,"SELECTION");
 assert.equal(discover.owner,"DiscoverNavigation");
 assert.equal(discover.defaultRef,"vexstream.desktop.library.discover-panel.pc");
 assert.equal(discover.exhaustive,true);
 assert.equal(discover.gate.presenceRef,shell.ref);
 assert.equal(discover.gate.choice,2);
 assert.deepEqual(
  discover.choices.map(choice=>choice.targetRef),
  [
   "vexstream.desktop.library.discover-panel.pc",
   "vexstream.desktop.library.discover-panel.youtube",
   "vexstream.desktop.library.discover-panel.explore"
  ]
 );
 assert.ok(screen.view.stateBindings.some(binding=>binding.presenceRef===discover.ref&&binding.value==="vexstream.desktop.library.discover-panel.youtube"));
 assert.ok(screen.view.stateBindings.some(binding=>binding.presenceRef===discover.ref&&binding.value==="vexstream.desktop.library.discover-panel.explore"));
});


test("Playlists list/create terrain is a source-authored Shell expression while detail remains held",()=>{
 const lens=JSON.parse(committed);
 const screen=lens.screens[0];
 const shell=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.shell.primary-terrain");
 assert.equal(screen.view.scriptPolicy,"NONE");
 assert.ok(shell.choices.some(choice=>choice.targetRef==="vexstream.desktop.library.panel.playlists"));
 assert.ok(screen.view.stateBindings.some(binding=>
  binding.presenceRef===shell.ref&&binding.value==="vexstream.desktop.library.panel.playlists"
 ));
 assert.match(screen.view.html,/data-shell-panel="playlists"[^>]*data-vex-structural-ref="vexstream\.desktop\.library\.panel\.playlists"/);
 assert.ok(screen.elements.some(element=>element.ref==="element.vexstream.playlists.create.name"));
 assert.ok(screen.elements.some(element=>element.ref==="element.vexstream.playlists.create"));
 assert.ok(screen.realization.held.some(value=>/Playlist detail/.test(value)));
});
