// [VXG RealForever]
import { PlaybackControl } from "./features/playback/behavior.js";
import { createHTMLMediaPlaybackConnection } from "./features/playback/connection.js";
import { QueueControl, QueueState, currentQueueRef } from "./features/queue/behavior.js";
import { createQueueConnection } from "./features/queue/connection.js";
import { createPlayerComposition } from "./features/player/composition.js";
import { playerJourney } from "./features/player/journey.js";
import {
  loadPlayerSession,
  reconcilePlayerSession,
  savePlayerSession,
} from "./features/player/session.js";
import { loadLibraryCatalog } from "./features/library/client.js";
import {
  restorePlaybackWindow,
  restoreTrackDisplay,
  updatePlaybackWindow,
  updateTrackDisplay,
} from "./features/library/admin.js";
import {
  loadTrash,
  moveTracksToTrash,
  purgeTrashItems,
  recoverTrashPurge,
  restoreTrashItems,
} from "./features/library/trash.js";
import {
  groupLibrary,
  libraryGenres,
  projectLibraryHome,
  projectLibraryView,
} from "./features/library/projection.js";
import {
  addSource,
  chooseSourceFolder,
  clearIgnoredSources,
  discoverSources,
  ignoreSourceCandidate,
  loadSourceConfig,
  removeSource,
  unignoreSource,
} from "./features/sources/client.js";
import {
  addTracksToPlaylist,
  createPlaylist,
  deletePlaylist,
  loadPlaylists,
  removeTracksFromPlaylist,
  renamePlaylist,
  reorderPlaylist,
  resolvePlaylistAvailability,
} from "./features/playlists/client.js";
import { createExploreSession } from "./features/explore/session.js";
import { createExploreComposition } from "./features/explore/composition.js";
import {
  cancelImport,
  discardImport,
  inspectImport,
  loadImportDestinations,
  loadImportJobs,
  recoverImportCommit,
  retryImportCommit,
  startImport,
} from "./features/imports/client.js";
import {
  buildImportPlan,
  formImportReview,
} from "./features/imports/review.js";
import {
  exitVexStream,
  loadSystemIdentity,
} from "./features/system/client.js";
import {
  ShellView,
  createShellNavigation,
} from "./features/shell/navigation.js";
import { escapeHTML } from "./foundation/html.js";
import { resolveText } from "./foundation/text.js";
import { en } from "./locales/en.js";

const locale=en;
const audio=document.querySelector("#audio");
const playPause=document.querySelector("#playPause");
const previous=document.querySelector("#previous");
const next=document.querySelector("#next");
const shuffle=document.querySelector("#shuffle");
const repeat=document.querySelector("#repeat");
const buildIdentity=document.querySelector("#buildIdentity");
const exitVexStreamButton=document.querySelector("#exitVexStream");
const status=document.querySelector("#status");
const semanticState=document.querySelector("#semanticState");
const queueState=document.querySelector("#queueState");
const sessionState=document.querySelector("#sessionState");
const queueSummary=document.querySelector("#queueSummary");
const queueList=document.querySelector("#queueList");
const clearAfterCurrent=document.querySelector("#clearAfterCurrent");
const trackChoices=document.querySelector("#trackChoices");
const libraryGroupDetail=document.querySelector("#libraryGroupDetail");
const libraryGroupBack=document.querySelector("#libraryGroupBack");
const libraryGroupTerrain=document.querySelector("#libraryGroupTerrain");
const libraryGroupEyebrow=document.querySelector("#libraryGroupEyebrow");
const libraryGroupTitle=document.querySelector("#libraryGroupTitle");
const libraryGroupSummary=document.querySelector("#libraryGroupSummary");
const libraryGroupPlay=document.querySelector("#libraryGroupPlay");
const libraryGroupAppend=document.querySelector("#libraryGroupAppend");
const libraryGroupTracks=document.querySelector("#libraryGroupTracks");
const librarySummary=document.querySelector("#librarySummary");
const librarySearch=document.querySelector("#librarySearch");
const libraryGenre=document.querySelector("#libraryGenre");
const librarySort=document.querySelector("#librarySort");
const libraryModeButtons=[...document.querySelectorAll("[data-library-mode]")];
const shellViewButtons=[...document.querySelectorAll("[data-shell-view]")];
const shellPanels=[...document.querySelectorAll("[data-shell-panel]")];
const libraryNavToggle=document.querySelector("#libraryNavToggle");
const librarySidebar=document.querySelector("#librarySidebar");
const homeSongCount=document.querySelector("#homeSongCount");
const homeAlbumCount=document.querySelector("#homeAlbumCount");
const homeArtistCount=document.querySelector("#homeArtistCount");
const homeSourceCount=document.querySelector("#homeSourceCount");
const recentSongs=document.querySelector("#recentSongs");
const sourceSummary=document.querySelector("#sourceSummary");
const sourceAdd=document.querySelector("#sourceAdd");
const sourceDiscover=document.querySelector("#sourceDiscover");
const sourceDiscoverDeep=document.querySelector("#sourceDiscoverDeep");
const sourceList=document.querySelector("#sourceList");
const sourceDiscovery=document.querySelector("#sourceDiscovery");
const ignoredSourceList=document.querySelector("#ignoredSourceList");
const ignoredSources=document.querySelector("#ignoredSources");
const clearIgnoredSourcesButton=document.querySelector("#clearIgnoredSources");
const libraryEditor=document.querySelector("#libraryEditor");
const libraryEditorClose=document.querySelector("#libraryEditorClose");
const libraryEditorSummary=document.querySelector("#libraryEditorSummary");
const editTitle=document.querySelector("#editTitle");
const editArtist=document.querySelector("#editArtist");
const editAlbum=document.querySelector("#editAlbum");
const editGenre=document.querySelector("#editGenre");
const editPlaybackStart=document.querySelector("#editPlaybackStart");
const editPlaybackEnd=document.querySelector("#editPlaybackEnd");
const saveTrackDisplay=document.querySelector("#saveTrackDisplay");
const restoreTrackDisplayButton=document.querySelector("#restoreTrackDisplay");
const savePlaybackWindow=document.querySelector("#savePlaybackWindow");
const restorePlaybackWindowButton=document.querySelector("#restorePlaybackWindow");
const newPlaylistName=document.querySelector("#newPlaylistName");
const createPlaylistButton=document.querySelector("#createPlaylist");
const playlistChoices=document.querySelector("#playlistChoices");
const trashSummary=document.querySelector("#trashSummary");
const trashRefresh=document.querySelector("#trashRefresh");
const trashItems=document.querySelector("#trashItems");
const deletedHistory=document.querySelector("#deletedHistory");
const purgeRecovery=document.querySelector("#purgeRecovery");
const recoverTrashPurgeButton=document.querySelector("#recoverTrashPurge");
const exploreSummary=document.querySelector("#exploreSummary");
const exploreCandidates=document.querySelector("#exploreCandidates");
const explorePreview=document.querySelector("#explorePreview");
const exploreBack=document.querySelector("#exploreBack");
const exploreClear=document.querySelector("#exploreClear");
const importReviewPanel=document.querySelector("#importReview");
const importReviewClose=document.querySelector("#importReviewClose");
const importReviewSourceTitle=document.querySelector("#importReviewSourceTitle");
const importReviewSourceCreator=document.querySelector("#importReviewSourceCreator");
const importDuplicateSummary=document.querySelector("#importDuplicateSummary");
const importDestination=document.querySelector("#importDestination");
const importMode=document.querySelector("#importMode");
const importCollection=document.querySelector("#importCollection");
const importArtist=document.querySelector("#importArtist");
const importGenre=document.querySelector("#importGenre");
const importSections=document.querySelector("#importSections");
const importStart=document.querySelector("#importStart");
const importReviewHint=document.querySelector("#importReviewHint");
const importRefresh=document.querySelector("#importRefresh");
const importJobs=document.querySelector("#importJobs");
const title=document.querySelector("#trackTitle");
const artist=document.querySelector("#trackArtist");
const error=document.querySelector("#error");

let systemIdentity=null;
let catalog=Object.freeze([]);
let byRef=new Map();
let refs=[];
let libraryView=Object.freeze({
 query:"",
 genre:"all",
 sort:"title",
 mode:"songs"
});
let sourceConfig=Object.freeze({
 sources:Object.freeze([]),
 ignored:Object.freeze([])
});
let sourceDiscoveryState=Object.freeze({
 candidates:Object.freeze([]),
 scannedDirs:0
});
let playlists=Object.freeze([]);
let resolvedPlaylists=Object.freeze([]);
let currentEditingTrackRef="";
let trashSnapshot=Object.freeze({
 trash:Object.freeze([]),
 deletedHistory:Object.freeze([]),
 pendingPurge:null
});
let currentSessionRestoration=Object.freeze({
 kind:"fresh",
 queue:null,
 droppedRefs:Object.freeze([]),
 currentRef:""
});
let playerSessionPersistenceEnabled=false;
let playerSessionSaveTimer=null;
let pendingPlayerSessionState=null;
let playerSessionSaveChain=Promise.resolve();
let currentImportReview=null;
let currentImportJobs=Object.freeze({
 active:Object.freeze([]),
 ready:Object.freeze([]),
 completed:Object.freeze([]),
 recovery:Object.freeze({}),
 issues:Object.freeze([])
});

const shellNavigation=createShellNavigation();


function trackTitle(item){
 return item?.title?.trim()||resolveText(locale,"library.unknown-title");
}

function trackArtist(item){
 return item?.artist?.trim()||resolveText(locale,"library.unknown-artist");
}

function renderHome(){
 const projection=projectLibraryHome(catalog,{recentLimit:8});
 homeSongCount.textContent=String(projection.songCount);
 homeAlbumCount.textContent=String(projection.albumCount);
 homeArtistCount.textContent=String(projection.artistCount);
 homeSourceCount.textContent=String(sourceConfig.sources.length);
 recentSongs.innerHTML=projection.recent.length
  ? projection.recent.map(item=>`<article class="home-track" data-home-track-ref="${escapeHTML(item.ref)}">
      <div class="home-track-copy">
       <strong>${escapeHTML(trackTitle(item))}</strong>
       <small>${escapeHTML(trackArtist(item))}</small>
      </div>
      <button type="button" data-home-track-action="play">${escapeHTML(resolveText(locale,"playback.play"))}</button>
     </article>`).join("")
  : `<div class="empty-state">${escapeHTML(resolveText(locale,"library.empty.filtered"))}</div>`;
}

function renderShell(projection=shellNavigation.currentProjection){
 document.body.dataset.shellView=projection.view;
 document.body.dataset.sidebarOpen=String(projection.sidebarOpen);
 librarySidebar.hidden=!projection.sidebarOpen;
 libraryNavToggle.setAttribute("aria-expanded",String(projection.sidebarOpen));

 for(const button of shellViewButtons){
  const selected=button.id==="libraryTopTab"
   ? projection.view!==ShellView.DISCOVER
   : button.dataset.shellView===projection.view;
  button.classList.toggle("active",selected);
  button.setAttribute("aria-pressed",String(selected));
 }

 for(const panel of shellPanels){
  panel.hidden=panel.dataset.shellPanel!==projection.panel;
 }

 if(projection.libraryMode&&libraryView.mode!==projection.libraryMode){
  libraryView=Object.freeze({
   ...libraryView,
   mode:projection.libraryMode
  });
  renderTrackChoices();
 }
 renderLibraryGroupDetail();
}

shellNavigation.subscribe(renderShell);

function currentLibraryView(){
 return projectLibraryView(catalog,{
  query:libraryView.query,
  genre:libraryView.genre,
  sort:libraryView.sort
 });
}

function libraryPlaybackRefs(){
 return [...refs];
}

function applyCatalog(nextCatalog){
 catalog=nextCatalog;
 byRef=new Map(catalog.map(item=>[item.ref,item]));
 refs=catalog.map(item=>item.ref);
 resolvedPlaylists=Object.freeze(
  playlists.map(playlist=>resolvePlaylistAvailability(playlist,catalog))
 );
 reconcileQueueAgainstCurrentLibrary();
}

function reconcileQueueAgainstCurrentLibrary(){
 const state=queue.currentProjection.state;
 if(state.kind!=="positioned")return;
 const available=new Set(refs);
 const kept=[];
 let currentIndex=-1;
 state.items.forEach((ref,index)=>{
  if(!available.has(ref))return;
  if(index===state.currentIndex){
   currentIndex=kept.length;
  }
  kept.push(ref);
 });

 if(currentIndex<0){
  audio.pause();
  audio.removeAttribute("src");
  delete audio.dataset.mediaRef;
  audio.load();
  playback.refresh();
  queue.restore({
   items:[],
   currentIndex:-1,
   shuffle:state.shuffle,
   repeat:state.repeat
  });
  currentSessionRestoration=Object.freeze({
   kind:"held-current-unavailable",
   queue:null,
   droppedRefs:Object.freeze(
    state.items.filter(ref=>!available.has(ref))
   ),
   currentRef:currentQueueRef(state)??""
  });
  return;
 }

 queue.restore({
  items:kept,
  currentIndex,
  shuffle:state.shuffle,
  repeat:state.repeat
 });
 player.presentCurrent();
}

async function refreshCatalog(){
 const nextCatalog=await loadLibraryCatalog();
 applyCatalog(nextCatalog);
 renderTrackChoices();
 renderSources();
 renderPlaylistChoices();
 renderLibraryEditor();
 renderTrash();
 render();
 return nextCatalog;
}

async function refreshPlaylists(){
 playlists=await loadPlaylists();
 resolvedPlaylists=Object.freeze(
  playlists.map(playlist=>resolvePlaylistAvailability(playlist,catalog))
 );
 renderPlaylistChoices();
 renderTrackChoices();
 return playlists;
}

async function refreshTrash(){
 trashSnapshot=await loadTrash();
 renderTrash();
 return trashSnapshot;
}

function renderLibraryEditor(){
 const item=byRef.get(currentEditingTrackRef);
 if(!item){
  libraryEditor.hidden=true;
  return;
 }
 libraryEditor.hidden=false;
 libraryEditorSummary.textContent=resolveText(locale,"library.editor.summary",{
  title:trackTitle(item)
 });
 editTitle.value=item.title??"";
 editArtist.value=item.artist??"";
 editAlbum.value=item.album??"";
 editGenre.value=item.genre??"";
 editPlaybackStart.value=String(Number(item.playbackStartSeconds)||0);
 editPlaybackEnd.value=item.playbackEndSeconds==null
  ?""
  :String(item.playbackEndSeconds);
}

function renderTrash(){
 trashSummary.textContent=trashSnapshot.trash.length
  ? resolveText(locale,"trash.summary",{count:trashSnapshot.trash.length})
  : resolveText(locale,"trash.empty");
 trashItems.innerHTML=trashSnapshot.trash.length
  ? trashSnapshot.trash.map(item=>`<article class="trash-row" data-trash-id="${escapeHTML(item.id)}">
      <div>
       <strong>${escapeHTML(item.track?.title||resolveText(locale,"library.unknown-title"))}</strong>
       <small>${escapeHTML(item.track?.artist||resolveText(locale,"library.unknown-artist"))}</small>
      </div>
      <div class="trash-actions">
       <button data-trash-action="restore">${escapeHTML(resolveText(locale,"trash.restore"))}</button>
       <button data-trash-action="purge">${escapeHTML(resolveText(locale,"trash.delete"))}</button>
      </div>
     </article>`).join("")
  : "";

 deletedHistory.innerHTML=trashSnapshot.deletedHistory.length
  ? `<details class="deleted-history">
      <summary>${escapeHTML(resolveText(locale,"trash.history",{
       count:trashSnapshot.deletedHistory.length
      }))}</summary>
      <div>${trashSnapshot.deletedHistory.slice(0,20).map(item=>`
       <div class="deleted-row">
        <span>${escapeHTML(item.track?.title||resolveText(locale,"library.unknown-title"))}</span>
        <small>${escapeHTML(item.deletedAt||"")}</small>
       </div>`).join("")}</div>
     </details>`
  : "";

 purgeRecovery.hidden=!trashSnapshot.pendingPurge;
}

function randomShuffle(values){
 const result=[...values];
 for(let i=result.length-1;i>0;i--){
  const j=Math.floor(Math.random()*(i+1));
  [result[i],result[j]]=[result[j],result[i]];
 }
 return result;
}

const queue=createQueueConnection({
 initialState:QueueState.empty(),
 shuffle:window.__VEXSTREAM_V4_SHUFFLE??randomShuffle
});

const playback=createHTMLMediaPlaybackConnection({
 media:audio,
 hasCurrentMedia:()=>Boolean(currentQueueRef(queue.currentProjection.state)&&audio.dataset.mediaRef),
 restartCurrent:async media=>{
  const item=byRef.get(currentQueueRef(queue.currentProjection.state));
  media.currentTime=Math.max(0,Number(item?.playbackStartSeconds)||0);
 }
});

const player=createPlayerComposition({
 queue,
 playback,
 media:audio,
 resolveMedia:ref=>byRef.get(ref),
 defaultContextRefs:()=>libraryPlaybackRefs(),
 onCurrentMedia:item=>{
  title.textContent=trackTitle(item);
  artist.textContent=trackArtist(item);
 }
});

const exploreSession=createExploreSession();
const explore=createExploreComposition({
 session:exploreSession,
 playback
});

function sessionStatusText(restoration=currentSessionRestoration){
 switch(restoration?.kind){
 case "restored":
  return restoration.droppedRefs?.length
   ? resolveText(locale,"session.status.restored-trimmed",{
       count:restoration.droppedRefs.length
     })
   : resolveText(locale,"session.status.restored");
 case "empty":
  return resolveText(locale,"session.status.empty");
 case "held-current-unavailable":
  return resolveText(locale,"session.status.held-current");
 case "held-schema":
  return resolveText(locale,"session.status.held-schema");
 case "persistence-held":
  return resolveText(locale,"session.status.save-held");
 default:
  return resolveText(locale,"session.status.fresh");
 }
}

function renderSessionState(){
 if(!sessionState)return;
 sessionState.dataset.sessionKind=currentSessionRestoration?.kind??"fresh";
 sessionState.textContent=sessionStatusText();
}

function schedulePlayerSessionSave(state){
 if(!playerSessionPersistenceEnabled)return;
 pendingPlayerSessionState=state;
 if(playerSessionSaveTimer){
  clearTimeout(playerSessionSaveTimer);
 }
 playerSessionSaveTimer=setTimeout(()=>{
  playerSessionSaveTimer=null;
  const snapshot=pendingPlayerSessionState;
  pendingPlayerSessionState=null;
  playerSessionSaveChain=playerSessionSaveChain
   .then(()=>savePlayerSession(snapshot))
   .catch(failure=>{
    playerSessionPersistenceEnabled=false;
    currentSessionRestoration=Object.freeze({
     kind:"persistence-held",
     queue:null,
     droppedRefs:Object.freeze([]),
     currentRef:"",
     detail:failure?.message??String(failure)
    });
    render();
   });
 },60);
}

function setControl(button,presentation){
 button.disabled=!presentation.enabled;
 button.dataset.textRef=presentation.textRef;
 button.dataset.actRef=presentation.act?.ref??"";
 button.textContent=resolveText(locale,presentation.textRef);
 if("selected" in presentation){
  button.setAttribute("aria-pressed",String(Boolean(presentation.selected)));
 }
}

function renderExplore(){
 const projection=exploreSession.currentProjection;
 exploreBack.disabled=!projection.canGoBack;
 exploreClear.disabled=projection.kind==="empty";

 if(projection.kind==="empty"){
  exploreSummary.textContent=resolveText(locale,"explore.summary.empty");
  exploreCandidates.innerHTML="";
  return;
 }

 const seedTitle=String(projection.frame?.seed?.title??"").trim();
 exploreSummary.textContent=resolveText(locale,"explore.summary.active",{
  title:seedTitle||resolveText(locale,"library.unknown-title"),
  count:projection.frame.candidates.length
 });

 exploreCandidates.innerHTML=projection.frame.candidates.map((candidate,index)=>{
  const match=candidate.libraryMatch?.state||"NOT_IN_LIBRARY";
  const matchTextRef={
   IN_LIBRARY:"explore.match.in-library",
   LIKELY_MATCH:"explore.match.likely",
   POSSIBLE_MATCH:"explore.match.possible",
   NOT_IN_LIBRARY:"explore.match.not-in-library"
  }[match]??"explore.match.possible";
  const axis=candidate.axisLabel||candidate.axis||resolveText(locale,"explore.axis.other");
  return `<article class="explore-card" data-explore-index="${index}">
   <div class="explore-copy">
    <strong>${escapeHTML(String(candidate.title??"").trim()||resolveText(locale,"library.unknown-title"))}</strong>
    <small>${escapeHTML(String(candidate.channel??"").trim()||resolveText(locale,"library.unknown-artist"))}</small>
    <span class="explore-meta">${escapeHTML(String(axis))} · ${escapeHTML(resolveText(locale,matchTextRef))}</span>
   </div>
   <div class="explore-card-actions">
    <button data-explore-action="preview">${escapeHTML(resolveText(locale,"explore.preview"))}</button>
    <button data-explore-action="branch">${escapeHTML(resolveText(locale,"explore.branch"))}</button>
    ${candidate.url&&match!=="IN_LIBRARY"
      ? `<button data-explore-action="import">${escapeHTML(resolveText(locale,"import.review"))}</button>`
      : ""}
   </div>
  </article>`;
 }).join("");
}

function renderImportReview(){
 if(!currentImportReview){
  importReviewPanel.hidden=true;
  return;
 }
 importReviewPanel.hidden=false;
 importReviewSourceTitle.textContent=currentImportReview.title;
 importReviewSourceCreator.textContent=currentImportReview.creator;
 importDuplicateSummary.textContent=currentImportReview.duplicateItemCount
  ? resolveText(locale,"import.duplicates.present",{
      count:currentImportReview.duplicateItemCount
    })
  : resolveText(locale,"import.duplicates.none");

 importDestination.innerHTML=currentImportReview.destinations.length
  ? currentImportReview.destinations.map(value=>
      `<option value="${escapeHTML(value)}">${escapeHTML(value)}</option>`
    ).join("")
  : `<option value="">${escapeHTML(resolveText(locale,"import.destination.none"))}</option>`;

 importMode.value=currentImportReview.defaultMode;
 for(const option of importMode.options){
  if(option.value==="split"||option.value==="both"){
   option.disabled=!currentImportReview.splitAvailable;
  }
 }
 importCollection.value=currentImportReview.collection;
 importArtist.value=currentImportReview.artistOverride;
 importGenre.value=currentImportReview.genre;

 importSections.innerHTML=currentImportReview.splitAvailable
  ? currentImportReview.sections.map(section=>`<label class="import-section" data-import-section="${section.index}">
      <input type="checkbox" data-import-section-selected ${section.selected?"checked":""}>
      <span>${escapeHTML(formatSeconds(section.start))}–${escapeHTML(formatSeconds(section.end))}</span>
      <input type="text" data-import-section-title value="${escapeHTML(section.title)}">
      <input type="text" data-import-section-artist value="${escapeHTML(section.artist)}" placeholder="${escapeHTML(resolveText(locale,"import.section.artist"))}">
    </label>`).join("")
  : "";

 importStart.disabled=currentImportReview.destinations.length===0;
 importReviewHint.textContent=currentImportReview.splitAvailable
  ? resolveText(locale,"import.hint.sections")
  : resolveText(locale,"import.hint.whole");
}

function renderImportJobs(){
 const rows=[
  ...currentImportJobs.active,
  ...currentImportJobs.ready,
  ...currentImportJobs.completed
 ];
 if(!rows.length){
  importJobs.innerHTML=`<div class="muted">${escapeHTML(resolveText(locale,"import.jobs.empty"))}</div>`;
  return;
 }

 importJobs.innerHTML=rows.map(job=>{
  const recovery=currentImportJobs.recovery[job.id]??"";
  const problem=job.problem?.title||job.problem?.message||"";
  const actions=[];
  if(["queued","running","cancelling"].includes(job.status)){
   actions.push(`<button data-import-job-action="cancel">${escapeHTML(resolveText(locale,"import.cancel"))}</button>`);
  }
  if(job.status==="ready"){
   actions.push(`<button data-import-job-action="retry">${escapeHTML(resolveText(locale,"import.retry"))}</button>`);
   actions.push(`<button data-import-job-action="discard">${escapeHTML(resolveText(locale,"import.discard"))}</button>`);
  }
  if(job.status==="failed"){
   actions.push(`<button data-import-job-action="discard">${escapeHTML(resolveText(locale,"import.discard"))}</button>`);
  }
  if(recovery==="COMMIT_UNCERTAIN"){
   actions.push(`<button data-import-job-action="recover">${escapeHTML(resolveText(locale,"import.recover"))}</button>`);
  }
  return `<article class="import-job" data-import-job-id="${escapeHTML(job.id)}">
    <div class="import-job-copy">
      <strong>${escapeHTML(job.title||resolveText(locale,"import.untitled"))}</strong>
      <span>${escapeHTML(job.stage||job.status||"")}</span>
      <small>${escapeHTML(job.detail||problem)}</small>
    </div>
    <div class="import-job-progress">${Number(job.progress||0)}%</div>
    <div class="import-job-actions">${actions.join("")}</div>
  </article>`;
 }).join("");
}

function formatSeconds(value){
 const seconds=Math.max(0,Math.floor(Number(value)||0));
 const minutes=Math.floor(seconds/60);
 const rest=String(seconds%60).padStart(2,"0");
 return `${minutes}:${rest}`;
}

function importSectionEdits(){
 return [...importSections.querySelectorAll("[data-import-section]")].map(row=>({
  index:Number(row.dataset.importSection),
  selected:row.querySelector("[data-import-section-selected]")?.checked!==false,
  title:row.querySelector("[data-import-section-title]")?.value??"",
  artist:row.querySelector("[data-import-section-artist]")?.value??""
 }));
}

async function refreshImportJobs(){
 currentImportJobs=await loadImportJobs();
 renderImportJobs();
 return currentImportJobs;
}

function render(){
 const pp=player.playPausePresentation;
 const qp=queue.currentProjection;
 setControl(playPause,pp);
 setControl(next,qp.controls.next);
 setControl(shuffle,qp.controls.shuffle);
 setControl(repeat,qp.controls.repeat);
 setControl(clearAfterCurrent,qp.controls.clearAfterCurrent);

 const hasCurrent=Boolean(currentQueueRef(qp.state));
 previous.disabled=!hasCurrent;
 previous.textContent=resolveText(locale,"queue.previous");
 previous.dataset.textRef="queue.previous";

 semanticState.textContent=playback.currentProjection.state.kind;
 queueState.textContent=qp.state.kind;
 renderSessionState();
 status.textContent=playback.currentProjection.state.requestPending
  ? resolveText(locale,"playback.status.pending")
  : resolveText(locale,"playback.status.ready");

 const current=currentQueueRef(qp.state);
 if(!current){
  title.textContent=resolveText(locale,"player.nothing.title");
  artist.textContent=resolveText(locale,"player.nothing.artist");
 }

 queueSummary.textContent=qp.state.kind==="empty"
  ? resolveText(locale,"queue.summary.empty")
  : resolveText(locale,"queue.summary.positioned",{
      count:qp.state.items.length,
      position:qp.state.currentIndex+1
    });

 renderExplore();
 renderImportReview();
 renderImportJobs();

 queueList.innerHTML=qp.state.kind==="empty"?"":qp.state.items.map((ref,index)=>{
  const item=byRef.get(ref);
  const isCurrent=index===qp.state.currentIndex;
  const currentClass=isCurrent?"current":"";
  const stateTextRef=index<qp.state.currentIndex
   ? "queue.item.played"
   : isCurrent
     ? "queue.item.current"
     : "queue.item.up-next";
  const stateLabel=resolveText(locale,stateTextRef);
  const upDisabled=isCurrent||index===0;
  const downDisabled=isCurrent||index===qp.state.items.length-1;
  return `<li class="${currentClass}" data-queue-ref="${escapeHTML(ref)}" data-queue-index="${index}">
    <span class="queue-index">${index+1}</span>
    <span class="queue-title"><strong>${escapeHTML(trackTitle(item))}</strong><span class="queue-state">${escapeHTML(stateLabel)}</span></span>
    <span class="queue-state">${isCurrent?"●":" "}</span>
    <span class="queue-actions">
      <button data-queue-action="play" ${isCurrent?"disabled":""}>${escapeHTML(resolveText(locale,"playback.play"))}</button>
      <button data-queue-action="up" ${upDisabled?"disabled":""}>↑</button>
      <button data-queue-action="down" ${downDisabled?"disabled":""}>↓</button>
      <button data-queue-action="remove" ${isCurrent?"disabled":""}>${escapeHTML(resolveText(locale,"queue.remove"))}</button>
    </span>
  </li>`;
 }).join("");
}

function trackCardHTML(item){
 return `<article class="track-card" data-track-ref="${escapeHTML(item.ref)}">
      <div class="track-card-copy">
       <strong>${escapeHTML(trackTitle(item))}</strong>
       <small>${escapeHTML(trackArtist(item))}</small>
       <span class="track-meta">${escapeHTML(
        [item.album,item.genre,item.year].filter(Boolean).join(" · ")
       )}</span>
      </div>
      <span class="track-actions">
       <button data-track-action="play" data-vex-action="vexstream.player.play-track">${escapeHTML(resolveText(locale,"playback.play"))}</button>
       <button data-track-action="play-next" data-vex-action="vexstream.player.play-next">${escapeHTML(resolveText(locale,"player.play-next"))}</button>
       <button data-track-action="append" data-vex-action="vexstream.player.add-queue">${escapeHTML(resolveText(locale,"player.add-to-queue"))}</button>
       <button data-track-action="explore" data-vex-action="vexstream.explore.from-track">${escapeHTML(resolveText(locale,"explore.open"))}</button>
       <button data-track-action="edit">${escapeHTML(resolveText(locale,"library.edit"))}</button>
       <button data-track-action="trash">${escapeHTML(resolveText(locale,"trash.move"))}</button>
       ${playlists.length
        ? `<select data-track-playlist-target>${playlists.map(playlist=>
             `<option value="${escapeHTML(playlist.id)}">${escapeHTML(playlist.name)}</option>`
            ).join("")}</select>
           <button data-track-action="playlist-add">${escapeHTML(resolveText(locale,"playlist.add-track"))}</button>`
        : ""}
      </span>
     </article>`;
}

function currentLibraryDetailGroup(){
 const detail=shellNavigation.currentProjection.libraryDetail;
 if(!detail)return null;
 return groupLibrary(catalog,detail.kind)
  .find(group=>group.key===detail.key)??null;
}

function renderLibraryGroupDetail(){
 const detail=shellNavigation.currentProjection.libraryDetail;
 if(!detail){
  libraryGroupDetail.hidden=true;
  trackChoices.hidden=false;
  return;
 }
 const group=currentLibraryDetailGroup();
 if(!group){
  shellNavigation.closeLibraryDetail();
  return;
 }

 const isAlbum=detail.kind==="album";
 const related=isAlbum
  ?new Set(group.tracks.map(item=>trackArtist(item))).size
  :new Set(group.tracks.map(item=>
    String(item?.album??"").trim()||"Unknown album"
   )).size;

 libraryGroupDetail.hidden=false;
 trackChoices.hidden=true;
 libraryGroupBack.textContent=resolveText(
  locale,
  isAlbum?"library.detail.back-albums":"library.detail.back-artists"
 );
 libraryGroupTerrain.textContent=resolveText(
  locale,
  isAlbum?"library.detail.terrain.album":"library.detail.terrain.artist"
 );
 libraryGroupEyebrow.textContent=resolveText(
  locale,
  isAlbum?"library.detail.eyebrow.album":"library.detail.eyebrow.artist"
 );
 libraryGroupTitle.textContent=group.name;
 libraryGroupSummary.textContent=resolveText(
  locale,
  isAlbum?"library.detail.summary.album":"library.detail.summary.artist",
  {tracks:group.refs.length,related}
 );
 libraryGroupPlay.textContent=resolveText(
  locale,
  isAlbum?"library.detail.play.album":"library.detail.play.artist"
 );
 libraryGroupAppend.textContent=resolveText(
  locale,
  isAlbum?"library.detail.queue.album":"library.detail.queue.artist"
 );
 libraryGroupTracks.innerHTML=group.tracks.map(trackCardHTML).join("");
}

function renderTrackChoices(){
 const visible=currentLibraryView();
 const genres=libraryGenres(catalog);
 const preferredGenre=libraryView.genre;
 libraryGenre.innerHTML=[
  `<option value="all">${escapeHTML(resolveText(locale,"library.genre.all"))}</option>`,
  ...genres.map(value=>
   `<option value="${escapeHTML(value)}">${escapeHTML(value)}</option>`
  )
 ].join("");
 libraryGenre.value=genres.some(
  value=>value.toLocaleLowerCase()===preferredGenre.toLocaleLowerCase()
 )?preferredGenre:"all";

 librarySearch.value=libraryView.query;
 librarySort.value=libraryView.sort;
 for(const button of libraryModeButtons){
  const selected=button.dataset.libraryMode===libraryView.mode;
  button.setAttribute("aria-pressed",String(selected));
 }

 if(libraryView.mode==="songs"){
  librarySummary.textContent=resolveText(locale,"library.summary.songs",{
   visible:visible.length,
   total:catalog.length
  });
  trackChoices.innerHTML=visible.length
   ?visible.map(trackCardHTML).join("")
   :`<div class="empty-state">${escapeHTML(resolveText(locale,"library.empty.filtered"))}</div>`;
  renderLibraryGroupDetail();
  return;
 }

 const groupKey=libraryView.mode==="albums"?"album":"artist";
 const groups=groupLibrary(visible,groupKey);
 librarySummary.textContent=resolveText(
  locale,
  libraryView.mode==="albums"
   ?"library.summary.albums"
   :"library.summary.artists",
  {count:groups.length,tracks:visible.length}
 );
 trackChoices.innerHTML=groups.length
  ?groups.map(group=>`<article class="library-group-card"
       data-library-group="${escapeHTML(group.key)}"
       data-library-group-kind="${escapeHTML(groupKey)}">
      <div class="library-group-copy">
       <strong>${escapeHTML(group.name)}</strong>
       <small>${escapeHTML(resolveText(locale,"library.group.tracks",{
        count:group.refs.length
       }))}</small>
       <span class="track-meta">${escapeHTML(
        group.tracks.slice(0,3).map(track=>trackTitle(track)).join(" · ")
       )}</span>
      </div>
      <div class="track-actions">
       <button data-library-group-action="open">${escapeHTML(resolveText(locale,"library.group.open"))}</button>
       <button data-library-group-action="play">${escapeHTML(resolveText(locale,"playback.play"))}</button>
       <button data-library-group-action="append">${escapeHTML(resolveText(locale,"player.add-to-queue"))}</button>
      </div>
     </article>`).join("")
  :`<div class="empty-state">${escapeHTML(resolveText(locale,"library.empty.filtered"))}</div>`;
 renderLibraryGroupDetail();
}

function sourceLabel(path){
 const parts=String(path??"").replace(/\\/g,"/").split("/").filter(Boolean);
 return parts.at(-1)||String(path??"");
}

function renderSources(){
 sourceSummary.textContent=sourceConfig.sources.length
  ? resolveText(locale,"sources.summary",{
      count:sourceConfig.sources.length
    })
  : resolveText(locale,"sources.summary.empty");

 sourceList.innerHTML=sourceConfig.sources.length
  ? sourceConfig.sources.map(path=>`<article class="source-row" data-source-path="${escapeHTML(path)}">
      <div>
       <strong>${escapeHTML(sourceLabel(path))}</strong>
       <small>${escapeHTML(path)}</small>
      </div>
      <button data-source-action="remove">${escapeHTML(resolveText(locale,"sources.remove"))}</button>
     </article>`).join("")
  : `<div class="empty-state">${escapeHTML(resolveText(locale,"sources.none"))}</div>`;

 sourceDiscovery.innerHTML=sourceDiscoveryState.candidates.length
  ? `<div class="source-discovery-summary">${escapeHTML(resolveText(locale,"sources.discovery.summary",{
      count:sourceDiscoveryState.candidates.length,
      scanned:sourceDiscoveryState.scannedDirs
     }))}</div>`+
    sourceDiscoveryState.candidates.map(candidate=>`<article class="source-row source-candidate" data-source-candidate="${escapeHTML(candidate.path)}">
      <div>
       <strong>${escapeHTML(candidate.name||sourceLabel(candidate.path))}</strong>
       <small>${escapeHTML(candidate.path)} · ${escapeHTML(resolveText(locale,"sources.mp3-count",{
        count:candidate.mp3Count
       }))}</small>
      </div>
      <div class="source-row-actions">
       <button class="primary" data-source-candidate-action="include">${escapeHTML(resolveText(locale,"sources.include"))}</button>
       <button data-source-candidate-action="ignore">${escapeHTML(resolveText(locale,"sources.ignore"))}</button>
      </div>
     </article>`).join("")
  : "";

 ignoredSources.hidden=sourceConfig.ignored.length===0;
 clearIgnoredSourcesButton.disabled=sourceConfig.ignored.length===0;
 ignoredSourceList.innerHTML=sourceConfig.ignored.map(path=>`<article class="source-row" data-ignored-source="${escapeHTML(path)}">
   <div><small>${escapeHTML(path)}</small></div>
   <button data-ignored-source-action="restore">${escapeHTML(resolveText(locale,"sources.restore"))}</button>
  </article>`).join("");
 renderHome();
}

function renderPlaylistChoices(){
 playlistChoices.innerHTML=resolvedPlaylists.length
  ? resolvedPlaylists.map(resolved=>{
   const playlist=resolved.playlist;
   const availability=resolveText(locale,"playlist.availability",{
    available:resolved.playableRefs.length,
    missing:resolved.missingRefs.length
   });
   const entries=resolved.entries.map((entry,index)=>{
    const label=entry.available
     ? trackTitle(entry.track)
     : resolveText(locale,"playlist.missing-track");
    return `<li data-playlist-track-ref="${escapeHTML(entry.ref)}"
        data-playlist-index="${index}"
        data-available="${String(entry.available)}">
      <span>${escapeHTML(label)}</span>
      <span class="playlist-entry-actions">
       <button data-playlist-entry-action="up" ${index===0?"disabled":""}>↑</button>
       <button data-playlist-entry-action="down" ${index===resolved.entries.length-1?"disabled":""}>↓</button>
       <button data-playlist-entry-action="remove">${escapeHTML(resolveText(locale,"playlist.remove-track"))}</button>
      </span>
     </li>`;
   }).join("");
   return `<article class="playlist-card" data-playlist-id="${escapeHTML(playlist.id)}">
    <div class="playlist-heading">
     <input data-playlist-name value="${escapeHTML(playlist.name)}" aria-label="${escapeHTML(resolveText(locale,"playlist.name"))}">
     <small>${escapeHTML(availability)}</small>
    </div>
    <div class="playlist-card-actions">
     <button data-playlist-action="play" ${resolved.playableRefs.length?"":"disabled"}>${escapeHTML(resolveText(locale,"playlist.play"))}</button>
     <button data-playlist-action="rename">${escapeHTML(resolveText(locale,"playlist.rename"))}</button>
     <button data-playlist-action="delete">${escapeHTML(resolveText(locale,"playlist.delete"))}</button>
    </div>
    <ol class="playlist-entries">${entries}</ol>
   </article>`;
  }).join("")
  : `<div class="empty-state">${escapeHTML(resolveText(locale,"playlist.empty"))}</div>`;
}

async function guarded(effect){
 error.textContent="";
 try{
  await effect();
 }catch(failure){
  error.textContent=failure?.message??String(failure);
 }
 render();
}

async function applySourceMutation(mutation,{refreshLibrary=false}={}){
 sourceConfig=mutation.config;
 if(mutation.warning){
  error.textContent=mutation.warning;
 }
 if(refreshLibrary){
  await refreshCatalog();
 }
 renderSources();
}

function removeDiscoveryCandidate(path){
 sourceDiscoveryState=Object.freeze({
  ...sourceDiscoveryState,
  candidates:Object.freeze(
   sourceDiscoveryState.candidates.filter(candidate=>candidate.path!==path)
  )
 });
}

exitVexStreamButton.addEventListener("click",()=>void guarded(async()=>{
 exitVexStreamButton.disabled=true;
 exitVexStreamButton.textContent=resolveText(locale,"system.exit.exiting");
 await exitVexStream();
 buildIdentity.textContent=resolveText(locale,"system.exit.complete");
 status.textContent=resolveText(locale,"system.exit.complete");
}));

playPause.addEventListener("click",()=>void guarded(()=>player.activatePlayPause()));
next.addEventListener("click",()=>void guarded(()=>player.next()));
previous.addEventListener("click",()=>void guarded(()=>player.previous()));
shuffle.addEventListener("click",()=>{player.toggleShuffle();render();});
repeat.addEventListener("click",()=>{player.toggleRepeat();render();});
clearAfterCurrent.addEventListener("click",()=>{player.clearAfterCurrent();render();});

librarySearch.addEventListener("input",()=>{
 libraryView=Object.freeze({
  ...libraryView,
  query:librarySearch.value
 });
 renderTrackChoices();
});

libraryGenre.addEventListener("change",()=>{
 libraryView=Object.freeze({
  ...libraryView,
  genre:libraryGenre.value
 });
 renderTrackChoices();
});

librarySort.addEventListener("change",()=>{
 libraryView=Object.freeze({
  ...libraryView,
  sort:librarySort.value
 });
 renderTrackChoices();
});

for(const button of libraryModeButtons){
 button.addEventListener("click",()=>{
  shellNavigation.select(button.dataset.libraryMode);
 });
}

for(const button of shellViewButtons){
 button.addEventListener("click",()=>{
  shellNavigation.select(button.dataset.shellView);
 });
}

libraryNavToggle.addEventListener("click",()=>{
 shellNavigation.toggleSidebar();
});

recentSongs.addEventListener("click",event=>{
 const action=event.target.closest("[data-home-track-action]");
 const row=event.target.closest("[data-home-track-ref]");
 if(!action||!row)return;
 if(action.dataset.homeTrackAction==="play"){
  void guarded(()=>player.playTrack(
   row.dataset.homeTrackRef,
   libraryPlaybackRefs()
  ));
 }
});

function handleTrackAction(event,{playbackRefs=libraryPlaybackRefs()}={}){
 const action=event.target.closest("[data-track-action]");
 const card=event.target.closest("[data-track-ref]");
 if(!action||!card)return false;
 const ref=card.dataset.trackRef;
 if(action.dataset.trackAction==="play"){
  void guarded(()=>player.playTrack(ref,playbackRefs));
  return true;
 }
 if(action.dataset.trackAction==="play-next"){
  void guarded(()=>player.playNext([ref]));
  return true;
 }
 if(action.dataset.trackAction==="append"){
  player.append([ref]);
  render();
  return true;
 }
 if(action.dataset.trackAction==="explore"){
  void guarded(async()=>{
   await explore.replaceFromTrack(ref);
   explorePreview.hidden=true;
   explorePreview.removeAttribute("src");
  });
  return true;
 }
 if(action.dataset.trackAction==="edit"){
  currentEditingTrackRef=ref;
  renderLibraryEditor();
  return true;
 }
 if(action.dataset.trackAction==="trash"){
  void guarded(async()=>{
   await moveTracksToTrash([ref]);
   if(currentEditingTrackRef===ref){
    currentEditingTrackRef="";
   }
   await Promise.all([
    refreshCatalog(),
    refreshTrash()
   ]);
   renderLibraryEditor();
  });
  return true;
 }
 if(action.dataset.trackAction==="playlist-add"){
  const target=card.querySelector("[data-track-playlist-target]")?.value??"";
  if(!target)return true;
  void guarded(async()=>{
   await addTracksToPlaylist(target,[ref]);
   await refreshPlaylists();
  });
  return true;
 }
 return false;
}

trackChoices.addEventListener("click",event=>{
 const groupAction=event.target.closest("[data-library-group-action]");
 const groupCard=event.target.closest("[data-library-group]");
 if(groupAction&&groupCard){
  const groups=groupLibrary(
   currentLibraryView(),
   groupCard.dataset.libraryGroupKind
  );
  const group=groups.find(value=>value.key===groupCard.dataset.libraryGroup);
  if(!group)return;
  if(groupAction.dataset.libraryGroupAction==="open"){
   shellNavigation.openLibraryDetail(
    groupCard.dataset.libraryGroupKind,
    group.key
   );
   return;
  }
  if(groupAction.dataset.libraryGroupAction==="play"){
   void guarded(()=>player.playContext(group.refs,0));
   return;
  }
  if(groupAction.dataset.libraryGroupAction==="append"){
   player.append(group.refs);
   render();
   return;
  }
 }
 handleTrackAction(event);
});

libraryGroupTracks.addEventListener("click",event=>{
 const group=currentLibraryDetailGroup();
 handleTrackAction(event,{
  playbackRefs:group?.refs??libraryPlaybackRefs()
 });
});

libraryGroupBack.addEventListener("click",()=>{
 shellNavigation.closeLibraryDetail();
});

libraryGroupPlay.addEventListener("click",()=>{
 const group=currentLibraryDetailGroup();
 if(!group)return;
 void guarded(()=>player.playContext(group.refs,0));
});

libraryGroupAppend.addEventListener("click",()=>{
 const group=currentLibraryDetailGroup();
 if(!group)return;
 player.append(group.refs);
 render();
});

libraryEditorClose.addEventListener("click",()=>{
 currentEditingTrackRef="";
 renderLibraryEditor();
});

saveTrackDisplay.addEventListener("click",()=>void guarded(async()=>{
 if(!currentEditingTrackRef)return;
 await updateTrackDisplay(currentEditingTrackRef,{
  title:editTitle.value,
  artist:editArtist.value,
  album:editAlbum.value,
  genre:editGenre.value
 });
 await refreshCatalog();
 renderLibraryEditor();
}));

restoreTrackDisplayButton.addEventListener("click",()=>void guarded(async()=>{
 if(!currentEditingTrackRef)return;
 await restoreTrackDisplay(currentEditingTrackRef);
 await refreshCatalog();
 renderLibraryEditor();
}));

savePlaybackWindow.addEventListener("click",()=>void guarded(async()=>{
 if(!currentEditingTrackRef)return;
 const end=editPlaybackEnd.value.trim()===""
  ?null
  :Number(editPlaybackEnd.value);
 await updatePlaybackWindow(currentEditingTrackRef,{
  start:Number(editPlaybackStart.value||0),
  end
 });
 await refreshCatalog();
 renderLibraryEditor();
}));

restorePlaybackWindowButton.addEventListener("click",()=>void guarded(async()=>{
 if(!currentEditingTrackRef)return;
 await restorePlaybackWindow(currentEditingTrackRef);
 await refreshCatalog();
 renderLibraryEditor();
}));

sourceAdd.addEventListener("click",()=>void guarded(async()=>{
 const path=await chooseSourceFolder();
 const mutation=await addSource(path);
 await applySourceMutation(mutation,{refreshLibrary:true});
}));

sourceDiscover.addEventListener("click",()=>void guarded(async()=>{
 sourceDiscoveryState=await discoverSources({deep:false});
 renderSources();
}));

sourceDiscoverDeep.addEventListener("click",()=>void guarded(async()=>{
 sourceDiscoveryState=await discoverSources({deep:true});
 renderSources();
}));

sourceList.addEventListener("click",event=>{
 const action=event.target.closest("[data-source-action]");
 const row=event.target.closest("[data-source-path]");
 if(!action||!row)return;
 if(action.dataset.sourceAction==="remove"){
  void guarded(async()=>{
   const mutation=await removeSource(row.dataset.sourcePath);
   await applySourceMutation(mutation,{refreshLibrary:true});
  });
 }
});

sourceDiscovery.addEventListener("click",event=>{
 const action=event.target.closest("[data-source-candidate-action]");
 const row=event.target.closest("[data-source-candidate]");
 if(!action||!row)return;
 const path=row.dataset.sourceCandidate;

 if(action.dataset.sourceCandidateAction==="include"){
  void guarded(async()=>{
   const mutation=await addSource(path);
   removeDiscoveryCandidate(path);
   await applySourceMutation(mutation,{refreshLibrary:true});
  });
  return;
 }

 if(action.dataset.sourceCandidateAction==="ignore"){
  void guarded(async()=>{
   const mutation=await ignoreSourceCandidate(path);
   sourceConfig=mutation.config;
   removeDiscoveryCandidate(path);
   renderSources();
  });
 }
});

ignoredSourceList.addEventListener("click",event=>{
 const action=event.target.closest("[data-ignored-source-action]");
 const row=event.target.closest("[data-ignored-source]");
 if(!action||!row)return;
 void guarded(async()=>{
  const mutation=await unignoreSource(row.dataset.ignoredSource);
  sourceConfig=mutation.config;
  renderSources();
 });
});

clearIgnoredSourcesButton.addEventListener("click",()=>void guarded(async()=>{
 const mutation=await clearIgnoredSources();
 sourceConfig=mutation.config;
 renderSources();
}));

createPlaylistButton.addEventListener("click",()=>void guarded(async()=>{
 const name=newPlaylistName.value.trim();
 if(!name)return;
 await createPlaylist(name,[]);
 newPlaylistName.value="";
 await refreshPlaylists();
}));

playlistChoices.addEventListener("click",event=>{
 const card=event.target.closest("[data-playlist-id]");
 if(!card)return;
 const id=card.dataset.playlistId;
 const resolved=resolvedPlaylists.find(value=>value.playlist.id===id);
 if(!resolved)return;

 const action=event.target.closest("[data-playlist-action]");
 if(action){
  if(action.dataset.playlistAction==="play"){
   if(resolved.playableRefs.length){
    void guarded(()=>player.playContext(resolved.playableRefs,0));
   }
   return;
  }
  if(action.dataset.playlistAction==="rename"){
   const name=card.querySelector("[data-playlist-name]")?.value??"";
   void guarded(async()=>{
    await renamePlaylist(id,name);
    await refreshPlaylists();
   });
   return;
  }
  if(action.dataset.playlistAction==="delete"){
   void guarded(async()=>{
    await deletePlaylist(id);
    await refreshPlaylists();
   });
   return;
  }
 }

 const entryAction=event.target.closest("[data-playlist-entry-action]");
 const entry=event.target.closest("[data-playlist-index]");
 if(!entryAction||!entry)return;
 const index=Number(entry.dataset.playlistIndex);
 const refs=[...resolved.playlist.trackRefs];

 if(entryAction.dataset.playlistEntryAction==="remove"){
  const ref=refs[index];
  void guarded(async()=>{
   await removeTracksFromPlaylist(id,[ref]);
   await refreshPlaylists();
  });
  return;
 }

 const target=entryAction.dataset.playlistEntryAction==="up"
  ?index-1
  :index+1;
 if(target<0||target>=refs.length)return;
 [refs[index],refs[target]]=[refs[target],refs[index]];
 void guarded(async()=>{
  await reorderPlaylist(id,refs);
  await refreshPlaylists();
 });
});

exploreCandidates.addEventListener("click",event=>{
 const action=event.target.closest("[data-explore-action]");
 const card=event.target.closest("[data-explore-index]");
 if(!action||!card)return;
 const index=Number(card.dataset.exploreIndex);

 if(action.dataset.exploreAction==="preview"){
  void guarded(async()=>{
   const preview=await explore.preview(index);
   explorePreview.src=preview.url;
   explorePreview.hidden=false;
  });
  return;
 }

 if(action.dataset.exploreAction==="branch"){
  void guarded(async()=>{
   await explore.branchFromCandidate(index);
   explorePreview.hidden=true;
   explorePreview.removeAttribute("src");
  });
  return;
 }

 if(action.dataset.exploreAction==="import"){
  void guarded(async()=>{
   const candidate=exploreSession.currentProjection.frame?.candidates?.[index];
   if(!candidate?.url){
    throw new Error(resolveText(locale,"import.source.unavailable"));
   }
   const [reviewed,destinations]=await Promise.all([
    inspectImport(candidate.url),
    loadImportDestinations()
   ]);
   currentImportReview=formImportReview(reviewed,destinations);
  });
 }
});

exploreBack.addEventListener("click",()=>void guarded(async()=>{
 await explore.back();
 explorePreview.hidden=true;
 explorePreview.removeAttribute("src");
}));

exploreClear.addEventListener("click",()=>void guarded(async()=>{
 await explore.clear();
 explorePreview.hidden=true;
 explorePreview.removeAttribute("src");
}));

importReviewClose.addEventListener("click",()=>{
 currentImportReview=null;
 renderImportReview();
});

importStart.addEventListener("click",()=>void guarded(async()=>{
 if(!currentImportReview)return;
 const plan=buildImportPlan(currentImportReview,{
  destination:importDestination.value,
  mode:importMode.value,
  collection:importCollection.value,
  artistOverride:importArtist.value,
  genre:importGenre.value,
  sections:importSectionEdits()
 });
 const started=await startImport(plan);
 currentImportReview=null;
 await refreshImportJobs();
 if(started?.jobId){
  importReviewHint.textContent=resolveText(locale,"import.started",{
   id:started.jobId
  });
 }
}));

importRefresh.addEventListener("click",()=>void guarded(()=>refreshImportJobs()));

importJobs.addEventListener("click",event=>{
 const action=event.target.closest("[data-import-job-action]");
 const row=event.target.closest("[data-import-job-id]");
 if(!action||!row)return;
 const id=row.dataset.importJobId;
 void guarded(async()=>{
  switch(action.dataset.importJobAction){
  case "cancel":
   await cancelImport(id);
   break;
  case "retry":
   await retryImportCommit(id);
   break;
  case "recover":
   await recoverImportCommit(id);
   break;
  case "discard":
   await discardImport(id);
   break;
  }
  await refreshImportJobs();
 });
});

trashRefresh.addEventListener("click",()=>void guarded(()=>refreshTrash()));

trashItems.addEventListener("click",event=>{
 const action=event.target.closest("[data-trash-action]");
 const row=event.target.closest("[data-trash-id]");
 if(!action||!row)return;
 const id=row.dataset.trashId;

 if(action.dataset.trashAction==="restore"){
  void guarded(async()=>{
   await restoreTrashItems([id]);
   await Promise.all([
    refreshTrash(),
    refreshCatalog()
   ]);
  });
  return;
 }

 if(action.dataset.trashAction==="purge"){
  void guarded(async()=>{
   await purgeTrashItems([id],{preserveMetadata:true});
   await refreshTrash();
  });
 }
});

recoverTrashPurgeButton.addEventListener("click",()=>void guarded(async()=>{
 await recoverTrashPurge();
 await refreshTrash();
}));

queueList.addEventListener("click",event=>{
 const action=event.target.closest("[data-queue-action]");
 const row=event.target.closest("[data-queue-index]");
 if(!action||!row)return;
 const index=Number(row.dataset.queueIndex);
 if(action.dataset.queueAction==="play"){
  void guarded(()=>player.playQueueIndex(index));
  return;
 }
 if(action.dataset.queueAction==="up"){
  player.moveQueue(index,index-1);
  render();
  return;
 }
 if(action.dataset.queueAction==="down"){
  player.moveQueue(index,index+2);
  render();
  return;
 }
 if(action.dataset.queueAction==="remove"){
  player.removeQueue(index);
  render();
 }
});

queue.subscribe(projection=>{
 render();
 schedulePlayerSessionSave(projection.state);
});
playback.subscribe(render);
exploreSession.subscribe(render);
render();

async function initialize(){
 const playerSessionLoad=loadPlayerSession().catch(failure=>Object.freeze({
  failure
 }));
 const [nextCatalog,nextPlaylists,nextImportJobs,loadedPlayerSession,nextIdentity,nextSourceConfig,nextTrash]=await Promise.all([
  loadLibraryCatalog(),
  loadPlaylists(),
  loadImportJobs(),
  playerSessionLoad,
  loadSystemIdentity(),
  loadSourceConfig(),
  loadTrash()
 ]);
 systemIdentity=nextIdentity;
 buildIdentity.textContent=resolveText(locale,"system.identity.ready",{
  version:systemIdentity.version,
  build:systemIdentity.buildRef
 });
 buildIdentity.title=systemIdentity.buildRef;
 buildIdentity.dataset.buildRef=systemIdentity.buildRef;
 exitVexStreamButton.textContent=resolveText(locale,"system.exit");

 playlists=nextPlaylists;
 sourceConfig=nextSourceConfig;
 trashSnapshot=nextTrash;
 applyCatalog(nextCatalog);
 currentImportJobs=nextImportJobs;

 if(loadedPlayerSession?.failure){
  currentSessionRestoration=Object.freeze({
   kind:"held-schema",
   queue:null,
   droppedRefs:Object.freeze([]),
   currentRef:"",
   detail:loadedPlayerSession.failure?.message??String(loadedPlayerSession.failure)
  });
 }else{
  currentSessionRestoration=reconcilePlayerSession(
   loadedPlayerSession,
   refs
  );
  if(currentSessionRestoration.queue){
   queue.restore(currentSessionRestoration.queue);
   player.presentCurrent();
  }
  playerSessionPersistenceEnabled=true;
 }

 renderTrackChoices();
 renderSources();
 renderPlaylistChoices();
 renderShell();
 render();
 window.vexV4=Object.freeze({
  ready:true,
  get identity(){return systemIdentity;},
  journey:playerJourney,
  queue,
  playback,
  player,
  media:audio,
  get catalog(){return catalog;},
  get libraryView(){return libraryView;},
  shellNavigation,
  get sourceConfig(){return sourceConfig;},
  get playlists(){return playlists;},
  get resolvedPlaylists(){return resolvedPlaylists;},
  get trash(){return trashSnapshot;},
  get editingTrackRef(){return currentEditingTrackRef;},
  exploreSession,
  explore,
  get importReview(){return currentImportReview;},
  get importJobs(){return currentImportJobs;},
  get sessionRestoration(){return currentSessionRestoration;},
  controls:Object.freeze({PlaybackControl,QueueControl})
 });
}

void initialize().catch(failure=>{
 error.textContent=failure?.message??String(failure);
});
