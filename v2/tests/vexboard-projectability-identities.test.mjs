// [VXG RealForever]
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { ShellNavigationIntent } from "../web/features/shell/navigation.js";
import { LibraryClientIntent } from "../web/features/library/client.js";
import { LibraryProjectionIntent } from "../web/features/library/projection.js";
import { PlayerCompositionIntent } from "../web/features/player/composition.js";
import { DiscoverNavigationIntent } from "../web/features/discover/navigation.js";
import { PlaybackTimelineIntent } from "../web/features/playback/timeline.js";
import { PlaybackVolumeIntent } from "../web/features/playback/volume.js";
import { QueuePresentationIntent } from "../web/features/queue/presentation.js";
import { QueueConnectionIntent } from "../web/features/queue/connection.js";

const HERE=path.dirname(fileURLToPath(import.meta.url));
const V2=path.resolve(HERE,"..");
const html=fs.readFileSync(path.join(V2,"web","index.html"),"utf8");
const app=fs.readFileSync(path.join(V2,"web","app.js"),"utf8");

const expected=Object.freeze([
  Object.freeze({id:"libraryNavToggle",identity:ShellNavigationIntent.ToggleSidebar}),
  Object.freeze({id:"rescanLibraryBtn",identity:LibraryClientIntent.Rescan}),
  Object.freeze({id:"shuffleAllBtn",identity:PlayerCompositionIntent.ShuffleLibrary}),
  Object.freeze({id:"librarySearch",identity:LibraryProjectionIntent.SearchQuery}),
  Object.freeze({id:"librarySearchExecute",identity:LibraryProjectionIntent.SearchExecute}),
  Object.freeze({id:"libraryGenre",identity:LibraryProjectionIntent.FilterGenre}),
  Object.freeze({id:"goDiscoverAdd",identity:DiscoverNavigationIntent.OpenAddMusic}),
  Object.freeze({id:"seek",identity:PlaybackTimelineIntent.Seek}),
  Object.freeze({id:"volume",identity:PlaybackVolumeIntent.Set}),
  Object.freeze({id:"queueMiniBtn",identity:QueuePresentationIntent.ToggleDrawer}),
  Object.freeze({id:"queueDrawerHandle",identity:QueuePresentationIntent.ToggleDrawer}),
  Object.freeze({id:"shuffleQueueUpcoming",identity:QueueConnectionIntent.ShuffleUpcoming}),
]);

test("all default-visible runtime-wired projectability gaps have source-owned identity",()=>{
  assert.deepEqual(expected.map(row=>row.identity),[
    "vexstream.shell.sidebar.toggle",
    "vexstream.library.rescan",
    "vexstream.player.shuffle-library",
    "vexstream.library.search.query",
    "vexstream.library.search.execute",
    "vexstream.library.filter.genre",
    "vexstream.discover.add.open",
    "vexstream.playback.timeline.seek",
    "vexstream.playback.volume.set",
    "vexstream.queue.drawer.toggle",
    "vexstream.queue.drawer.toggle",
    "vexstream.queue.shuffle-upcoming",
  ]);
  for(const row of expected){
    const tag=new RegExp("<[^>]*\\bid=\""+row.id+"\"[^>]*>");
    const match=html.match(tag);
    assert.ok(match,row.id+" remains in the lived shell");
    assert.match(match[0],new RegExp('data-vex-action="'+row.identity.replaceAll(".","\\.")+'"'));
  }
});

test("two Queue drawer affordances intentionally share one presentation identity",()=>{
  assert.equal(QueuePresentationIntent.ToggleDrawer,"vexstream.queue.drawer.toggle");
  const matches=[...html.matchAll(/data-vex-action="vexstream\.queue\.drawer\.toggle"/g)];
  assert.equal(matches.length,2);
});

test("identity alignment does not replace or reroute existing runtime handlers",()=>{
  assert.match(app,/libraryNavToggle\.addEventListener\("click",\(\)=>\{shellNavigation\.toggleSidebar\(\);\}\);/);
  assert.match(app,/rescanLibraryBtn\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{\s*await rescanLibrary\(\);\s*await refreshCatalog\(\);/);
  assert.match(app,/shuffleAllBtn\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{\s*const shuffled=shuffleRefs\(\[\.\.\.refs\]\);\s*if\(shuffled\.length\)await player\.playContext\(shuffled,0\);/);
  assert.match(app,/librarySearch\.addEventListener\("input",\(\)=>\{\s*libraryView=Object\.freeze\(\{\.\.\.libraryView,query:librarySearch\.value\}\);/);
  assert.match(app,/librarySearchExecute\.addEventListener\("click",executeLibrarySearch\);/);
  assert.match(app,/libraryGenre\.addEventListener\("change",\(\)=>\{\s*libraryView=Object\.freeze\(\{\.\.\.libraryView,genre:libraryGenre\.value\}\);/);
  assert.match(app,/goDiscoverAdd\.addEventListener\("click",\(\)=>\{\s*shellNavigation\.select\(ShellView\.DISCOVER\);\s*discoverNavigation\.select\(DiscoverMode\.PC\);/);
  assert.match(app,/seek\.addEventListener\("input",\(\)=>\{\s*timeline\.seekNormalized\(Number\(seek\.value\)\/1000\);/);
  assert.match(app,/volumeInput\.addEventListener\("input",\(\)=>\{\s*playbackVolume\.setVolume\(Number\(volumeInput\.value\)\);/);
  assert.match(app,/queueDrawerHandle\.addEventListener\("click",toggleQueueDrawer\);/);
  assert.match(app,/queueMiniBtn\.addEventListener\("click",toggleQueueDrawer\);/);
  assert.match(app,/shuffleQueueUpcoming\.addEventListener\("click",\(\)=>\{\s*player\.shuffleUpcoming\(\);/);
});
