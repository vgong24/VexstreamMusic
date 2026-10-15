// [VXG RealForever]
import { PlaybackControl } from "./features/playback/behavior.js";
import { createHTMLMediaPlaybackConnection } from "./features/playback/connection.js";
import { QueueControl, QueueState, currentQueueRef } from "./features/queue/behavior.js";
import { createQueueConnection } from "./features/queue/connection.js";
import { createPlayerComposition } from "./features/player/composition.js";
import { playerJourney } from "./features/player/journey.js";
import { loadLibraryCatalog } from "./features/library/client.js";
import {
  loadPlaylists,
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
const status=document.querySelector("#status");
const semanticState=document.querySelector("#semanticState");
const queueState=document.querySelector("#queueState");
const queueSummary=document.querySelector("#queueSummary");
const queueList=document.querySelector("#queueList");
const clearAfterCurrent=document.querySelector("#clearAfterCurrent");
const trackChoices=document.querySelector("#trackChoices");
const playlistChoices=document.querySelector("#playlistChoices");
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

let catalog=Object.freeze([]);
let byRef=new Map();
let refs=[];
let playlists=Object.freeze([]);
let resolvedPlaylists=Object.freeze([]);
let currentImportReview=null;
let currentImportJobs=Object.freeze({
 active:Object.freeze([]),
 ready:Object.freeze([]),
 completed:Object.freeze([]),
 recovery:Object.freeze({}),
 issues:Object.freeze([])
});

function trackTitle(item){
 return item?.title?.trim()||resolveText(locale,"library.unknown-title");
}

function trackArtist(item){
 return item?.artist?.trim()||resolveText(locale,"library.unknown-artist");
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
 defaultContextRefs:()=>refs,
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

function renderTrackChoices(){
 trackChoices.innerHTML=catalog.map(item=>`<div class="track-card" data-track-ref="${escapeHTML(item.ref)}">
  <strong>${escapeHTML(trackTitle(item))}</strong>
  <small>${escapeHTML(trackArtist(item))}</small>
  <span class="track-actions">
    <button data-track-action="play" data-vex-action="vexstream.player.play-track">${escapeHTML(resolveText(locale,"playback.play"))}</button>
    <button data-track-action="play-next" data-vex-action="vexstream.player.play-next">${escapeHTML(resolveText(locale,"player.play-next"))}</button>
    <button data-track-action="append" data-vex-action="vexstream.player.add-queue">${escapeHTML(resolveText(locale,"player.add-to-queue"))}</button>
    <button data-track-action="explore" data-vex-action="vexstream.explore.from-track">${escapeHTML(resolveText(locale,"explore.open"))}</button>
  </span>
</div>`).join("");
}

function renderPlaylistChoices(){
 playlistChoices.innerHTML=resolvedPlaylists.map(resolved=>{
  const playlist=resolved.playlist;
  const availability=resolveText(locale,"playlist.availability",{
   available:resolved.playableRefs.length,
   missing:resolved.missingRefs.length
  });
  const entries=resolved.entries.map(entry=>{
   const label=entry.available
    ? trackTitle(entry.track)
    : resolveText(locale,"playlist.missing-track");
   return `<li data-playlist-track-ref="${escapeHTML(entry.ref)}" data-available="${String(entry.available)}">${escapeHTML(label)}</li>`;
  }).join("");
  return `<article class="playlist-card" data-playlist-id="${escapeHTML(playlist.id)}">
   <div class="playlist-heading">
    <strong>${escapeHTML(playlist.name)}</strong>
    <small>${escapeHTML(availability)}</small>
   </div>
   <button data-playlist-action="play" ${resolved.playableRefs.length?"":"disabled"}>${escapeHTML(resolveText(locale,"playlist.play"))}</button>
   <ol class="playlist-entries">${entries}</ol>
  </article>`;
 }).join("");
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

playPause.addEventListener("click",()=>void guarded(()=>player.activatePlayPause()));
next.addEventListener("click",()=>void guarded(()=>player.next()));
previous.addEventListener("click",()=>void guarded(()=>player.previous()));
shuffle.addEventListener("click",()=>{player.toggleShuffle();render();});
repeat.addEventListener("click",()=>{player.toggleRepeat();render();});
clearAfterCurrent.addEventListener("click",()=>{player.clearAfterCurrent();render();});

trackChoices.addEventListener("click",event=>{
 const action=event.target.closest("[data-track-action]");
 const card=event.target.closest("[data-track-ref]");
 if(!action||!card)return;
 const ref=card.dataset.trackRef;
 if(action.dataset.trackAction==="play"){
  void guarded(()=>player.playTrack(ref,refs));
  return;
 }
 if(action.dataset.trackAction==="play-next"){
  void guarded(()=>player.playNext([ref]));
  return;
 }
 if(action.dataset.trackAction==="append"){
  player.append([ref]);
  render();
  return;
 }
 if(action.dataset.trackAction==="explore"){
  void guarded(async()=>{
   await explore.replaceFromTrack(ref);
   explorePreview.hidden=true;
   explorePreview.removeAttribute("src");
  });
 }
});

playlistChoices.addEventListener("click",event=>{
 const action=event.target.closest("[data-playlist-action]");
 const card=event.target.closest("[data-playlist-id]");
 if(!action||!card)return;
 const resolved=resolvedPlaylists.find(value=>value.playlist.id===card.dataset.playlistId);
 if(!resolved||!resolved.playableRefs.length)return;
 if(action.dataset.playlistAction==="play"){
  void guarded(()=>player.playContext(resolved.playableRefs,0));
 }
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

queue.subscribe(render);
playback.subscribe(render);
exploreSession.subscribe(render);
render();

async function initialize(){
 const [nextCatalog,nextPlaylists,nextImportJobs]=await Promise.all([
  loadLibraryCatalog(),
  loadPlaylists(),
  loadImportJobs()
 ]);
 catalog=nextCatalog;
 byRef=new Map(catalog.map(item=>[item.ref,item]));
 refs=catalog.map(item=>item.ref);
 playlists=nextPlaylists;
 resolvedPlaylists=Object.freeze(
  playlists.map(playlist=>resolvePlaylistAvailability(playlist,catalog))
 );
 currentImportJobs=nextImportJobs;
 renderTrackChoices();
 renderPlaylistChoices();
 render();
 window.vexV4=Object.freeze({
  ready:true,
  journey:playerJourney,
  queue,
  playback,
  player,
  media:audio,
  catalog,
  playlists,
  resolvedPlaylists,
  exploreSession,
  explore,
  get importReview(){return currentImportReview;},
  get importJobs(){return currentImportJobs;},
  controls:Object.freeze({PlaybackControl,QueueControl})
 });
}

void initialize().catch(failure=>{
 error.textContent=failure?.message??String(failure);
});
