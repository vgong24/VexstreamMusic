// [VXG RealForever]
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { ShellNavigationIntent } from "../web/features/shell/navigation.js";
import { LibraryClientIntent } from "../web/features/library/client.js";
import { LibraryProjectionIntent } from "../web/features/library/projection.js";
import { TrackSelectionIntent } from "../web/features/library/selection.js";
import { PlayerCompositionIntent } from "../web/features/player/composition.js";
import { DiscoverNavigationIntent } from "../web/features/discover/navigation.js";
import { PlaybackTimelineIntent } from "../web/features/playback/timeline.js";
import { PlaybackVolumeIntent } from "../web/features/playback/volume.js";
import { QueuePresentationIntent } from "../web/features/queue/presentation.js";
import { QueueConnectionIntent } from "../web/features/queue/connection.js";
import { SourceClientIntent } from "../web/features/sources/client.js";
import { ImportClientIntent } from "../web/features/imports/client.js";
import { PlaylistClientIntent } from "../web/features/playlists/client.js";
import { PlaylistNavigationIntent } from "../web/features/playlists/navigation.js";

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
  Object.freeze({id:"songLayoutButton",identity:LibraryProjectionIntent.OpenSongLayout}),
  Object.freeze({id:"playAllSongs",identity:PlayerCompositionIntent.PlayLibraryView}),
  Object.freeze({id:"selectSongs",identity:TrackSelectionIntent.ToggleMode}),
  Object.freeze({id:"sourceAdd",identity:SourceClientIntent.AddFolder}),
  Object.freeze({id:"sourceDiscover",identity:SourceClientIntent.Discover}),
  Object.freeze({id:"sourceDiscoverDeep",identity:SourceClientIntent.DiscoverDeep}),
  Object.freeze({id:"clearIgnoredSources",identity:SourceClientIntent.ClearIgnored}),
  Object.freeze({id:"youtubeImportURL",identity:ImportClientIntent.SourceURL}),
  Object.freeze({id:"youtubeInspect",identity:ImportClientIntent.Inspect}),
  Object.freeze({id:"importRefresh",identity:ImportClientIntent.RefreshJobs}),
  Object.freeze({id:"librarySort",identity:LibraryProjectionIntent.SortPreset}),
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
    "vexstream.library.song-layout.open",
    "vexstream.player.play-library-view",
    "vexstream.library.selection.toggle",
    "vexstream.sources.add-folder",
    "vexstream.sources.discover",
    "vexstream.sources.discover-deep",
    "vexstream.sources.ignored.clear",
    "vexstream.import.source-url",
    "vexstream.import.inspect",
    "vexstream.import.jobs.refresh",
    "vexstream.library.sort.preset",
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
  assert.match(app,/songLayoutButton\.addEventListener\("click",\(\)=>\{songLayoutPanel\.hidden=false;\}\);/);
  assert.match(app,/playAllSongs\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{\s*const visible=currentLibraryView\(\);\s*if\(visible\.length\)await player\.playContext\(/);
  assert.match(app,/selectSongs\.addEventListener\("click",\(\)=>\{\s*const wasActive=librarySelection\.currentProjection\.active;\s*librarySelection\.toggleMode\(\);/);
  assert.match(app,/sourceAdd\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{\s*const path=await chooseSourceFolder\(\);\s*const mutation=await addSource\(path\);/);
  assert.match(app,/sourceDiscover\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{\s*sourceDiscoveryState=await discoverSources\(\{deep:false\}\);/);
  assert.match(app,/sourceDiscoverDeep\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{\s*sourceDiscoveryState=await discoverSources\(\{deep:true\}\);/);
  assert.match(app,/clearIgnoredSourcesButton\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{\s*const mutation=await clearIgnoredSources\(\);/);
  assert.match(app,/youtubeInspect\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{\s*await loadImportReview\(youtubeImportURL\.value\);/);
  assert.match(app,/youtubeImportURL\.addEventListener\("keydown",event=>\{\s*if\(event\.key!=="Enter"\)return;/);
  assert.match(app,/importRefresh\.addEventListener\("click",\(\)=>void guarded\(\(\)=>refreshImportJobs\(\)\)\);/);
  assert.match(app,/librarySort\.addEventListener\("change",\(\)=>\{\s*if\(librarySort\.value==="custom"\)return;/);
  assert.match(app,/shuffleQueueUpcoming\.addEventListener\("click",\(\)=>\{\s*player\.shuffleUpcoming\(\);/);
});

test("ignored folders toggle keeps source identity without inventing an app handler",()=>{
 assert.equal(SourceClientIntent.ToggleIgnored,"vexstream.sources.ignored.toggle");
 assert.match(html,/<summary data-vex-action="vexstream\.sources\.ignored\.toggle">Ignored folders<\/summary>/);
});


test("Playlists terrain create controls keep source identity without rerouting behavior",()=>{
  assert.equal(PlaylistClientIntent.CreateName,"vexstream.playlists.create.name");
  assert.equal(PlaylistClientIntent.Create,"vexstream.playlists.create");
  assert.match(html,/<input id="newPlaylistName"[^>]*data-vex-action="vexstream\.playlists\.create\.name"/);
  assert.match(html,/<button id="createPlaylist"[^>]*data-vex-action="vexstream\.playlists\.create"/);
  assert.match(app,/createPlaylistButton\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{\s*const name=newPlaylistName\.value\.trim\(\);\s*if\(!name\)return;\s*await createPlaylist\(name,\[\]\);/);
});


test("selected Playlist detail controls keep truthful source identity without rerouting behavior",()=>{
  const expectedDetail=[
    ["playlistDetailBack",PlaylistNavigationIntent.CloseDetail],
    ["playlistDetailName",PlaylistClientIntent.RenameName],
    ["playlistDetailPlay",PlayerCompositionIntent.PlayPlaylist],
    ["playlistDetailShuffle",PlayerCompositionIntent.ShufflePlaylist],
    ["playlistDetailAddSongs",PlaylistNavigationIntent.AddSongs],
    ["playlistDetailRename",PlaylistClientIntent.Rename],
    ["playlistDetailDelete",PlaylistClientIntent.Delete],
  ];
  for(const [id,identity] of expectedDetail){
    const tag=html.match(new RegExp("<[^>]*\\bid=\\\""+id+"\\\"[^>]*>"))?.[0];
    assert.ok(tag,id+" remains in the lived shell");
    assert.match(tag,new RegExp('data-vex-action="'+identity.replaceAll(".","\\.")+'"'));
  }
  assert.match(app,/playlistDetailBack\.addEventListener\("click",\(\)=>\{\s*playlistNavigation\.close\(\);/);
  assert.match(app,/playlistDetailPlay\.addEventListener\("click",\(\)=>\{\s*const resolved=currentPlaylistDetail\(\);\s*if\(resolved\?\.playableRefs\.length\)\{\s*void guarded\(\(\)=>player\.playContext\(resolved\.playableRefs,0\)\);/);
  assert.match(app,/playlistDetailShuffle\.addEventListener\("click",\(\)=>\{\s*const resolved=currentPlaylistDetail\(\);\s*const shuffled=shuffleRefs/);
  assert.match(app,/playlistDetailAddSongs\.addEventListener\("click",\(\)=>\{\s*const resolved=currentPlaylistDetail\(\);[\s\S]*playlistSelectionTargetID=resolved\.playlist\.id;[\s\S]*librarySelection\.activate\(\);[\s\S]*shellNavigation\.select\(ShellView\.SONGS\);/);
  assert.match(app,/playlistDetailRename\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{[\s\S]*await renamePlaylist\(resolved\.playlist\.id,playlistDetailName\.value\);/);
  assert.match(app,/playlistDetailDelete\.addEventListener\("click",\(\)=>void guarded\(async\(\)=>\{[\s\S]*await deletePlaylist\(resolved\.playlist\.id\);/);
});
