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


test("Playlists list/create terrain remains a source-authored Shell expression when detail is admitted separately",()=>{
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
 assert.ok(screen.elements.some(element=>element.ref==="vexstream.desktop.library.playlists.overview"));
});


test("selected Playlist detail is a gated presentation with rightful fixed child pathways",()=>{
  const lens=JSON.parse(committed);
  const screen=lens.screens[0];
  const shell=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.shell.primary-terrain");
  const detail=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.playlists.detail-open");
  assert.equal(detail.kind,"PRESENTATION");
  assert.equal(detail.subjectRef,"vexstream.desktop.library.playlists.detail");
  assert.equal(detail.owner,"PlaylistNavigation");
  assert.equal(detail.gate.presenceRef,shell.ref);
  assert.equal(detail.gate.choice,3);
  assert.equal(detail.realizes.length,7);
  assert.ok(screen.view.stateBindings.some(binding=>binding.presenceRef===detail.ref&&binding.value==="OPEN"));
  assert.ok(screen.view.stateBindings.some(binding=>binding.presenceRef===detail.ref&&binding.value==="CLOSED"));
  assert.match(screen.view.html,/id="playlistOverview"[^>]*data-vex-structural-ref="vexstream\.desktop\.library\.playlists\.overview"/);
  assert.match(screen.view.html,/id="playlistDetail"[^>]*data-vex-structural-ref="vexstream\.desktop\.library\.playlists\.detail"/);

  const expectedPathways=[
    "pathway.vexstream.playlists.detail.close",
    "pathway.vexstream.player.play-playlist",
    "pathway.vexstream.player.shuffle-playlist",
    "pathway.vexstream.playlists.detail.add-songs",
    "pathway.vexstream.playlists.rename",
    "pathway.vexstream.playlists.delete",
  ];
  for(const ref of expectedPathways)assert.ok(lens.pathways.some(pathway=>pathway.pathwayRef===ref),ref+" is projected");
});

test("exported element kind follows the source interaction type",()=>{
  const lens=JSON.parse(committed);
  const elements=new Map(lens.screens[0].elements.map(element=>[element.ref,element]));
  assert.equal(elements.get("element.vexstream.playlists.rename.name")?.kind,"FIELD");
  assert.equal(elements.get("element.vexstream.playlists.create.name")?.kind,"FIELD");
  assert.equal(elements.get("element.vexstream.library.sort.preset")?.kind,"FIELD");
  assert.equal(elements.get("element.vexstream.playlists.rename")?.kind,"BUTTON");
  assert.equal(elements.get("element.vexstream.playback.play-pause")?.kind,"BUTTON");
});


test("Playlist runtime populations project typed cross-platform identity formation without live instances",()=>{
  const lens=JSON.parse(committed);
  const screen=lens.screens[0];
  const cardPopulation=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.playlists.cards");
  const entryPopulation=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.playlists.detail.entries");
  const cardTemplate=screen.realization.templates.find(entry=>entry.ref==="template.vexstream.playlists.card");
  const entryTemplate=screen.realization.templates.find(entry=>entry.ref==="template.vexstream.playlists.detail-entry");

  assert.equal(cardPopulation.kind,"POPULATION");
  assert.equal(cardPopulation.itemKey,"playlist.id");
  assert.equal(cardTemplate.instanceIdentity.kind,"RUNTIME_INSTANCE");
  assert.equal(cardTemplate.instanceIdentity.continuity,"STABLE_INSTANCE");
  assert.deepEqual(cardTemplate.instanceIdentity.partSources,["playlist.id"]);

  assert.equal(entryPopulation.kind,"POPULATION");
  assert.equal(entryPopulation.gate.presenceRef,"presence.vexstream.playlists.detail-open");
  assert.equal(entryTemplate.instanceIdentity.continuity,"CURRENT_OCCURRENCE");
  assert.deepEqual(
    entryTemplate.instanceIdentity.partSources,
    ["playlist.id","entry.ref","occurrenceWithinPlaylist"]
  );
  assert.ok(entryTemplate.slots.every(slot=>
    slot.depth===0||slot.instanceIdentity?.kind==="DERIVED_CHILD"
  ));
  assert.ok(screen.realization.held.some(value=>/does not yet own a durable per-entry ID/.test(value)));
});


test("Deleted terrain and Trash population reuse the generic expression and runtime identity grammar",()=>{
  const lens=JSON.parse(committed);
  const screen=lens.screens[0];
  const shell=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.shell.primary-terrain");
  const trashPopulation=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.trash.items");
  const trashTemplate=screen.realization.templates.find(entry=>entry.ref==="template.vexstream.trash.item");

  assert.ok(shell.choices.some(choice=>choice.targetRef==="vexstream.desktop.library.panel.deleted"));
  assert.ok(screen.view.stateBindings.some(binding=>
    binding.presenceRef===shell.ref&&binding.value==="vexstream.desktop.library.panel.deleted"
  ));
  assert.match(
    screen.view.html,
    /data-shell-panel="deleted"[^>]*data-vex-structural-ref="vexstream\.desktop\.library\.panel\.deleted"/
  );
  assert.ok(screen.elements.some(element=>element.ref==="element.vexstream.trash.refresh"));

  assert.equal(trashPopulation.kind,"POPULATION");
  assert.equal(trashPopulation.itemKey,"item.id");
  assert.equal(trashPopulation.gate.presenceRef,shell.ref);
  assert.equal(trashPopulation.gate.choice,4);

  assert.equal(trashTemplate.instanceIdentity.kind,"RUNTIME_INSTANCE");
  assert.equal(trashTemplate.instanceIdentity.continuity,"STABLE_INSTANCE");
  assert.deepEqual(trashTemplate.instanceIdentity.partSources,["item.id"]);
  assert.ok(trashTemplate.slots.filter(slot=>slot.depth>0).every(slot=>
    slot.instanceIdentity?.kind==="DERIVED_CHILD"
  ));
  assert.ok(screen.realization.held.some(value=>/Trash selection, purge confirmation and purge recovery/.test(value)));
});


test("source-authored selections name the controls that choose each projected child",()=>{
  const lens=JSON.parse(committed);
  const screen=lens.screens[0];
  const shell=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.shell.primary-terrain");
  const library=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.library.mode");
  const discover=screen.realization.presence.find(entry=>entry.ref==="presence.vexstream.discover.mode");

  assert.deepEqual(shell.choices.map(choice=>choice.selectorRefs),[
    ["element.vexstream.shell.view.home"],
    [
      "element.vexstream.shell.view.songs",
      "element.vexstream.shell.view.albums",
      "element.vexstream.shell.view.artists",
    ],
    ["element.vexstream.shell.view.discover"],
    ["element.vexstream.shell.view.playlists"],
    ["element.vexstream.shell.view.deleted"],
  ]);
  assert.deepEqual(library.choices.map(choice=>choice.selectorRefs),[
    ["element.vexstream.shell.view.songs","element.vexstream.library.mode.songs"],
    ["element.vexstream.shell.view.albums","element.vexstream.library.mode.albums"],
    ["element.vexstream.shell.view.artists","element.vexstream.library.mode.artists"],
  ]);
  assert.deepEqual(discover.choices.map(choice=>choice.selectorRefs),[
    ["element.vexstream.discover.mode.pc"],
    ["element.vexstream.discover.mode.youtube"],
    ["element.vexstream.discover.mode.explore"],
  ]);

  const refs=new Set(screen.elements.map(element=>element.ref));
  for(const selection of [shell,library,discover]){
    for(const choice of selection.choices){
      for(const selectorRef of choice.selectorRefs) assert.ok(refs.has(selectorRef),selectorRef+" is source-addressable");
    }
  }
});
