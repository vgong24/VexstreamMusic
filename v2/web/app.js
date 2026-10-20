// [VXG RealForever]
import { PlaybackControl } from "./features/playback/behavior.js";
import { createHTMLMediaPlaybackConnection } from "./features/playback/connection.js";
import { createHTMLMediaPlaybackTimelineConnection } from "./features/playback/timeline.js";
import { createHTMLMediaPlaybackVolumeConnection } from "./features/playback/volume.js";
import { QueueControl, QueueState, currentQueueRef } from "./features/queue/behavior.js";
import { createQueueConnection } from "./features/queue/connection.js";
import {
  QueueWindowPolicy,
  queueWindowAroundIndex,
  queueWindowForScroll,
} from "./features/queue/window.js";
import { createPlayerComposition } from "./features/player/composition.js";
import { playerJourney } from "./features/player/journey.js";
import {
  loadPlayerSession,
  reconcilePlayerSession,
  savePlayerSession,
} from "./features/player/session.js";
import { loadLibraryCatalog, rescanLibrary } from "./features/library/client.js";
import {
  restorePlaybackWindow,
  restoreTrackDisplay,
  updatePlaybackWindow,
  updateTrackDisplay,
  updateTrackDisplays,
} from "./features/library/admin.js";
import {
  loadTrash,
  moveTracksToTrash,
  purgeTrashItems,
  recoverTrashPurge,
  restoreTrashItems,
} from "./features/library/trash.js";
import {
  activateLibrarySortStack,
  groupLibrary,
  libraryGenres,
  librarySortPresetForStack,
  librarySortStackForPreset,
  projectLibraryHome,
  projectLibraryView,
  resetLibrarySortStack,
} from "./features/library/projection.js";
import { createTrackSelection } from "./features/library/selection.js";
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
import { createPlaylistNavigation } from "./features/playlists/navigation.js";
import { createExploreSession } from "./features/explore/session.js";
import { createExploreComposition } from "./features/explore/composition.js";
import {
  cancelImport,
  discardImport,
  inspectImport,
  loadImportDestinations,
  loadImportJobs,
  loadImportRuntimeStatus,
  recoverImportCommit,
  retryImportCommit,
  setupImportRuntime,
  startImport,
} from "./features/imports/client.js";
import {
  buildImportPlan,
  formImportReview,
} from "./features/imports/review.js";
import {
  exitVexStream,
  loadSessionRetirePlan,
  loadSystemIdentity,
  retireSessions,
} from "./features/system/client.js";
import { projectSystemStatus } from "./features/system/status.js";
import {
  decidePhonePairing,
  forgetPairedPhone,
  loadPhonePairing,
  startPhonePairing,
} from "./features/phone/client.js";
import { createPhoneConnection } from "./features/phone/connection.js";
import { presentPhoneConnection } from "./features/phone/presentation.js";
import {
  ShellView,
  createShellNavigation,
} from "./features/shell/navigation.js";
import {
  DiscoverMode,
  createDiscoverNavigation,
} from "./features/discover/navigation.js";
import { escapeHTML } from "./foundation/html.js";
import { marqueeDurationSeconds } from "./foundation/marquee.js";
import { resolveText } from "./foundation/text.js";
import { en } from "./locales/en.js";

const locale=en;
const audio=document.querySelector("#audio");
const playPause=document.querySelector("#playPause");
const previous=document.querySelector("#previous");
const next=document.querySelector("#next");
const shuffle=document.querySelector("#shuffle");
const repeat=document.querySelector("#repeat");
const seek=document.querySelector("#seek");
const curTime=document.querySelector("#curTime");
const durTime=document.querySelector("#durTime");
const volumeInput=document.querySelector("#volume");
const queueMiniBtn=document.querySelector("#queueMiniBtn");
const queueMiniCount=document.querySelector("#queueMiniCount");
const buildIdentity=document.querySelector("#buildIdentity");
const phoneConnectButton=document.querySelector("#phoneConnectButton");
const modalBackdrop=document.querySelector("#modalBackdrop");
const modalTitle=document.querySelector("#modalTitle");
const modalBody=document.querySelector("#modalBody");
const modalActions=document.querySelector("#modalActions");
const modalCloseBtn=document.querySelector("#modalCloseBtn");
const systemStatusButton=document.querySelector("#systemStatusButton");
const systemStatusBadge=document.querySelector("#systemStatusBadge");
const systemDrawer=document.querySelector("#systemDrawer");
const systemDrawerClose=document.querySelector("#systemDrawerClose");
const systemSummary=document.querySelector("#systemSummary");
const systemBuildStatus=document.querySelector("#systemBuildStatus");
const systemBuildRef=document.querySelector("#systemBuildRef");
const systemLibraryStatus=document.querySelector("#systemLibraryStatus");
const systemTrashStatus=document.querySelector("#systemTrashStatus");
const systemSessionStatus=document.querySelector("#systemSessionStatus");
const systemImportStatus=document.querySelector("#systemImportStatus");
const systemAttention=document.querySelector("#systemAttention");
const reviewOldSessions=document.querySelector("#reviewOldSessions");
const oldSessionPlan=document.querySelector("#oldSessionPlan");
const exitVexStreamButton=document.querySelector("#exitVexStream");
const status=document.querySelector("#status");
const semanticState=document.querySelector("#semanticState");
const queueState=document.querySelector("#queueState");
const sessionState=document.querySelector("#sessionState");
const queueDrawer=document.querySelector("#queueDrawer");
const queueDrawerHandle=document.querySelector("#queueDrawerHandle");
const queueDrawerCount=document.querySelector("#queueDrawerCount");
const queueDrawerNextCount=document.querySelector("#queueDrawerNextCount");
const queueSummary=document.querySelector("#queueSummary");
const queueCurrentPane=document.querySelector("#queueCurrentPane");
const queueTimelinePane=document.querySelector("#queueTimelinePane");
const queueList=document.querySelector("#queueList");
const shuffleQueueUpcoming=document.querySelector("#shuffleQueueUpcoming");
const clearAfterCurrent=document.querySelector("#clearAfterCurrent");
const trackChoices=document.querySelector("#trackChoices");
const playAllSongs=document.querySelector("#playAllSongs");
const songLayoutButton=document.querySelector("#songLayoutButton");
const songLayoutPanel=document.querySelector("#songLayoutPanel");
const songLayoutDone=document.querySelector("#songLayoutDone");
const songLayoutReset=document.querySelector("#songLayoutReset");
const songLayoutFieldInputs=[...document.querySelectorAll("[data-song-layout-field]")];
const selectSongs=document.querySelector("#selectSongs");
const librarySelectionBar=document.querySelector("#librarySelectionBar");
const librarySelectionCount=document.querySelector("#librarySelectionCount");
const selectVisibleSongs=document.querySelector("#selectVisibleSongs");
const bulkPlaySelected=document.querySelector("#bulkPlaySelected");
const bulkPlayNextSelected=document.querySelector("#bulkPlayNextSelected");
const bulkQueueSelected=document.querySelector("#bulkQueueSelected");
const bulkEditSelected=document.querySelector("#bulkEditSelected");
const bulkDisplayEditor=document.querySelector("#bulkDisplayEditor");
const bulkDisplayEditorTitle=document.querySelector("#bulkDisplayEditorTitle");
const bulkDisplayEditorSummary=document.querySelector("#bulkDisplayEditorSummary");
const bulkDisplayEditorClose=document.querySelector("#bulkDisplayEditorClose");
const bulkApplyArtist=document.querySelector("#bulkApplyArtist");
const bulkArtist=document.querySelector("#bulkArtist");
const bulkApplyAlbum=document.querySelector("#bulkApplyAlbum");
const bulkAlbum=document.querySelector("#bulkAlbum");
const bulkApplyGenre=document.querySelector("#bulkApplyGenre");
const bulkGenre=document.querySelector("#bulkGenre");
const applyBulkDisplay=document.querySelector("#applyBulkDisplay");
const cancelBulkDisplay=document.querySelector("#cancelBulkDisplay");
const bulkPlaylistTarget=document.querySelector("#bulkPlaylistTarget");
const bulkPlaylistSelected=document.querySelector("#bulkPlaylistSelected");
const bulkTrashSelected=document.querySelector("#bulkTrashSelected");
const clearSongSelection=document.querySelector("#clearSongSelection");
const libraryGroupDetail=document.querySelector("#libraryGroupDetail");
const libraryGroupBack=document.querySelector("#libraryGroupBack");
const libraryGroupTerrain=document.querySelector("#libraryGroupTerrain");
const libraryGroupArtwork=document.querySelector("#libraryGroupArtwork");
const libraryGroupEyebrow=document.querySelector("#libraryGroupEyebrow");
const libraryGroupTitle=document.querySelector("#libraryGroupTitle");
const libraryGroupSummary=document.querySelector("#libraryGroupSummary");
const libraryGroupPlay=document.querySelector("#libraryGroupPlay");
const libraryGroupShuffle=document.querySelector("#libraryGroupShuffle");
const libraryGroupPlayNext=document.querySelector("#libraryGroupPlayNext");
const libraryGroupAppend=document.querySelector("#libraryGroupAppend");
const libraryGroupRelations=document.querySelector("#libraryGroupRelations");
const artistDetailAlbums=document.querySelector("#artistDetailAlbums");
const artistAlbumsToggle=document.querySelector("#artistAlbumsToggle");
const artistAlbumCount=document.querySelector("#artistAlbumCount");
const artistAlbumsHint=document.querySelector("#artistAlbumsHint");
const artistAlbumShelf=document.querySelector("#artistAlbumShelf");
const artistDetailSongs=document.querySelector("#artistDetailSongs");
const artistSongSummary=document.querySelector("#artistSongSummary");
const artistSongSearch=document.querySelector("#artistSongSearch");
const artistAlbumFilter=document.querySelector("#artistAlbumFilter");
const artistSortSummary=document.querySelector("#artistSortSummary");
const artistSongTable=document.querySelector("#artistSongTable");
const libraryGroupTracks=document.querySelector("#libraryGroupTracks");
const librarySummary=document.querySelector("#librarySummary");
const songSortSummary=document.querySelector("#songSortSummary");
const librarySearch=document.querySelector("#librarySearch");
const libraryGenre=document.querySelector("#libraryGenre");
const librarySort=document.querySelector("#librarySort");
const libraryUtilityBar=document.querySelector("#libraryUtilityBar");
const librarySearchExecute=document.querySelector("#librarySearchExecute");
const librarySearchClear=document.querySelector("#librarySearchClear");
const goDiscoverAdd=document.querySelector("#goDiscoverAdd");
const rescanLibraryBtn=document.querySelector("#rescanLibraryBtn");
const shuffleAllBtn=document.querySelector("#shuffleAllBtn");
const playlistNavCount=document.querySelector("#playlistNavCount");
const trashNavCount=document.querySelector("#trashNavCount");
const libraryModeButtons=[...document.querySelectorAll("[data-library-mode]")];
const shellViewButtons=[...document.querySelectorAll("button[data-shell-view]")];
const shellPanels=[...document.querySelectorAll("[data-shell-panel]")];
const libraryNavToggle=document.querySelector("#libraryNavToggle");
const librarySidebar=document.querySelector("#librarySidebar");
const homeSongCount=document.querySelector("#homeSongCount");
const homeAlbumCount=document.querySelector("#homeAlbumCount");
const homeArtistCount=document.querySelector("#homeArtistCount");
const homeSourceCount=document.querySelector("#homeSourceCount");
const recentSongs=document.querySelector("#recentSongs");
const discoverModeButtons=[...document.querySelectorAll("[data-discover-mode]")];
const discoverPanels=[...document.querySelectorAll("[data-discover-panel]")];
const youtubeImportURL=document.querySelector("#youtubeImportURL");
const youtubeInspect=document.querySelector("#youtubeInspect");
const youtubeImportEntry=document.querySelector("#youtubeImportEntry");
const importRuntimeNotice=document.querySelector("#importRuntimeStatus");
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
const playbackWindowSummary=document.querySelector("#playbackWindowSummary");
const trackSourceDetails=document.querySelector("#trackSourceDetails");
const saveTrackDisplay=document.querySelector("#saveTrackDisplay");
const restoreTrackDisplayButton=document.querySelector("#restoreTrackDisplay");
const savePlaybackWindow=document.querySelector("#savePlaybackWindow");
const restorePlaybackWindowButton=document.querySelector("#restorePlaybackWindow");
const newPlaylistName=document.querySelector("#newPlaylistName");
const createPlaylistButton=document.querySelector("#createPlaylist");
const playlistOverview=document.querySelector("#playlistOverview");
const playlistChoices=document.querySelector("#playlistChoices");
const playlistDetail=document.querySelector("#playlistDetail");
const playlistDetailBack=document.querySelector("#playlistDetailBack");
const playlistDetailName=document.querySelector("#playlistDetailName");
const playlistDetailAvailability=document.querySelector("#playlistDetailAvailability");
const playlistDetailPlay=document.querySelector("#playlistDetailPlay");
const playlistDetailShuffle=document.querySelector("#playlistDetailShuffle");
const playlistDetailAddSongs=document.querySelector("#playlistDetailAddSongs");
const playlistDetailRename=document.querySelector("#playlistDetailRename");
const playlistDetailDelete=document.querySelector("#playlistDetailDelete");
const playlistDetailEntries=document.querySelector("#playlistDetailEntries");
const trashSummary=document.querySelector("#trashSummary");
const trashRefresh=document.querySelector("#trashRefresh");
const trashItems=document.querySelector("#trashItems");
const trashCount=document.querySelector("#trashCount");
const trashSelectionBar=document.querySelector("#trashSelectionBar");
const trashSelectionCount=document.querySelector("#trashSelectionCount");
const restoreSelectedTrashButton=document.querySelector("#restoreSelectedTrash");
const purgeSelectedTrashButton=document.querySelector("#purgeSelectedTrash");
const clearTrashSelectionButton=document.querySelector("#clearTrashSelection");
const deletedHistory=document.querySelector("#deletedHistory");
const deletedHistoryCount=document.querySelector("#deletedHistoryCount");
const trashPurgeConfirm=document.querySelector("#trashPurgeConfirm");
const trashPurgeConfirmSummary=document.querySelector("#trashPurgeConfirmSummary");
const preserveDeletedMetadata=document.querySelector("#preserveDeletedMetadata");
const trashPurgeCancel=document.querySelector("#trashPurgeCancel");
const trashPurgeCommit=document.querySelector("#trashPurgeCommit");
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
const importReviewDuration=document.querySelector("#importReviewDuration");
const importReviewThumbnail=document.querySelector("#importReviewThumbnail");
const importReviewThumbnailFallback=document.querySelector("#importReviewThumbnailFallback");
const importReviewLoading=document.querySelector("#importReviewLoading");
const importReviewLoadingDetail=document.querySelector("#importReviewLoadingDetail");
const importReviewContent=document.querySelector("#importReviewContent");
const importDuplicateSummary=document.querySelector("#importDuplicateSummary");
const importDuplicateDetails=document.querySelector("#importDuplicateDetails");
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
const nowPlaying=document.querySelector("#nowPlaying");
const nowArtButton=document.querySelector("#nowArtButton");
const nowArt=document.querySelector("#nowArt");
const nowTitleButton=document.querySelector("#nowTitleButton");
const title=document.querySelector("#trackTitle");
const nowDetails=document.querySelector("#nowDetails");
const nowMeta=document.querySelector("#nowMeta");
const error=document.querySelector("#error");

let systemIdentity=null;
let currentSystemStatus=null;
let catalog=Object.freeze([]);
let catalogRevision=0;
let byRef=new Map();
let refs=[];
let libraryView=Object.freeze({
 query:"",
 genre:"all",
 sort:"title",
 sortStack:librarySortStackForPreset("title"),
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
let playlistSelectionTargetID="";
let currentEditingTrackRef="";
const songColumnRules=Object.freeze({
 title:Object.freeze({defaultWidth:300,minWidth:110,maxWidth:1200}),
 added:Object.freeze({defaultWidth:80,minWidth:52,maxWidth:360}),
 artist:Object.freeze({defaultWidth:120,minWidth:32,maxWidth:900}),
 album:Object.freeze({defaultWidth:140,minWidth:32,maxWidth:1100}),
 genre:Object.freeze({defaultWidth:90,minWidth:32,maxWidth:700}),
 duration:Object.freeze({defaultWidth:60,minWidth:52,maxWidth:220})
});
function defaultSongWidths(){
 return Object.freeze(Object.fromEntries(
  Object.entries(songColumnRules).map(([key,rule])=>[key,rule.defaultWidth])
 ));
}
const defaultSongLayout=Object.freeze({
 artist:true,
 album:true,
 added:true,
 duration:true,
 genre:false,
 widths:defaultSongWidths()
});
function freshDefaultSongLayout(){
 return Object.freeze({
  artist:defaultSongLayout.artist,
  album:defaultSongLayout.album,
  added:defaultSongLayout.added,
  duration:defaultSongLayout.duration,
  genre:defaultSongLayout.genre,
  widths:defaultSongWidths()
 });
}
let songLayout=freshDefaultSongLayout();
let bulkDisplayRefs=Object.freeze([]);
let artistDetailIdentity="";
let artistAlbumsExpanded=false;
let artistSongQuery="";
let artistAlbumFilterKey="";
let trashSnapshot=Object.freeze({
 trash:Object.freeze([]),
 deletedHistory:Object.freeze([]),
 pendingPurge:null
});
const selectedTrashIDs=new Set();
let pendingTrashPurgeIDs=Object.freeze([]);
let currentSessionRestoration=Object.freeze({
 kind:"fresh",
 queue:null,
 droppedRefs:Object.freeze([]),
 currentRef:""
});
let playerSessionPersistenceEnabled=false;
let queueDrawerOpen=false;
const QUEUE_ROW_HEIGHT=58;
let queueWindowStart=0;
let queueWindowInitialized=false;
let queueWindowRenderKey="";
let queueScrollFrame=null;
let queueDragIndex=-1;
let queueDropIndex=-1;
const queueDropPlaceholder=document.createElement("li");
queueDropPlaceholder.className="queue-drop-placeholder";
queueDropPlaceholder.hidden=true;
queueDropPlaceholder.setAttribute("aria-hidden","true");
let playerSessionSaveTimer=null;
let pendingPlayerSessionState=null;
let playerSessionSaveChain=Promise.resolve();
let currentImportReview=null;
let importReviewLoadingState=false;
let importReviewLoadingURL="";
const importDuplicatePreviewAudio=new Audio();
let importDuplicatePreviewTimer=null;
let resumeAfterImportDuplicatePreview=false;
let importRuntimeState=null;
let importRefreshTimer=null;
let observedImportCompletionKey="";
let currentImportJobs=Object.freeze({
 active:Object.freeze([]),
 ready:Object.freeze([]),
 completed:Object.freeze([]),
 recovery:Object.freeze({}),
 issues:Object.freeze([])
});

const shellNavigation=createShellNavigation();
const discoverNavigation=createDiscoverNavigation();
const librarySelection=createTrackSelection();
const playlistNavigation=createPlaylistNavigation();
const phoneConnection=createPhoneConnection({
 loadPairing:()=>loadPhonePairing(),
 startPairing:()=>startPhonePairing(),
 decidePairing:(requestRef,decision)=>decidePhonePairing(requestRef,decision),
 forgetPairedPhone:deviceRef=>forgetPairedPhone(deviceRef),
 confirmForget:()=>globalThis.confirm(
  "Forget this phone? It will need to pair again before using your VexStream library."
 )
});


function trackTitle(item){
 return item?.title?.trim()||resolveText(locale,"library.unknown-title");
}

function trackArtist(item){
 return item?.artist?.trim()||resolveText(locale,"library.unknown-artist");
}

function renderNowArtwork(item){
 const source=String(item?.thumbnailURL??"").trim();
 nowArt.classList.toggle("has-art",Boolean(source));
 nowArt.innerHTML=source
  ?`<img src="${escapeHTML(source)}" alt="">`
  :"♫";
}

function currentTrackLibraryRelation(kind,item){
 if(!item)return null;
 return groupLibrary(catalog,kind).find(group=>
  group.refs.includes(item.ref)
 )??null;
}

function nowLibraryRelationsHTML(item){
 if(!item)return escapeHTML(resolveText(locale,"player.nothing.artist"));
 const artistGroup=currentTrackLibraryRelation("artist",item);
 const albumGroup=currentTrackLibraryRelation("album",item);
 const artistLabel=trackArtist(item);
 const albumLabel=String(item?.album??"").trim()||"Unknown album";
 const artistHTML=artistGroup&&artistLabel!=="Unknown artist"
  ?`<button type="button" class="now-relation"
       data-now-library-kind="artist"
       data-now-library-key="${escapeHTML(artistGroup.key)}">${escapeHTML(artistLabel)}</button>`
  :`<span class="now-relation-static">${escapeHTML(artistLabel)}</span>`;
 const albumHTML=albumGroup&&albumLabel!=="Unknown album"
  ?`<button type="button" class="now-relation"
       data-now-library-kind="album"
       data-now-library-key="${escapeHTML(albumGroup.key)}">${escapeHTML(albumLabel)}</button>`
  :`<span class="now-relation-static">${escapeHTML(albumLabel)}</span>`;
 return `${artistHTML}<span aria-hidden="true"> · </span>${albumHTML}`;
}

function renderNowPlaying(item){
 renderNowArtwork(item);
 const active=Boolean(item);
 nowPlaying.toggleAttribute("data-track-ref",active);
 if(active)nowPlaying.dataset.trackRef=item.ref;
 else delete nowPlaying.dataset.trackRef;
 nowArtButton.disabled=!active;
 nowTitleButton.disabled=!active;
 nowDetails.hidden=!active;
 if(!active)nowDetails.open=false;
 const visibleTitle=active
  ?trackTitle(item)
  :resolveText(locale,"player.nothing.title");
 title.classList.add("song-marquee");
 title.title=visibleTitle;
 title.innerHTML=`<span class="song-marquee-text">${escapeHTML(visibleTitle)}</span>`;
 nowMeta.innerHTML=nowLibraryRelationsHTML(item);
 requestAnimationFrame(()=>refreshSongMarqueeMetrics(nowPlaying));
}

function marqueeTextHTML(value,{tag="span",className=""}={}){
 const text=String(value??"");
 const classes=["song-marquee",className].filter(Boolean).join(" ");
 return `<${tag} class="${classes}" title="${escapeHTML(text)}"><span class="song-marquee-text">${escapeHTML(text)}</span></${tag}>`;
}

function refreshSongMarqueeMetrics(root=document){
 for(const outer of root.querySelectorAll(".song-marquee")){
  const inner=outer.querySelector(".song-marquee-text");
  if(!inner)continue;
  const shift=Math.max(0,inner.scrollWidth-outer.clientWidth);
  outer.style.setProperty("--song-marquee-shift",shift+"px");
  if(shift>3){
   // Long text travels farther, so it receives proportionally more time instead
   // of accelerating merely to finish inside one fixed animation duration.
   const duration=marqueeDurationSeconds(shift);
   outer.style.setProperty("--song-marquee-duration",duration.toFixed(2)+"s");
  }else{
   outer.style.removeProperty("--song-marquee-duration");
  }
  outer.classList.toggle("overflowing",shift>3);
 }
}

function syncActiveSongRows(){
 const current=player.currentItem?.ref??"";
 for(const row of document.querySelectorAll("[data-track-ref]")){
  row.classList.toggle("playing",Boolean(current)&&row.dataset.trackRef===current);
 }
 for(const row of document.querySelectorAll("[data-home-track-ref]")){
  row.classList.toggle("playing",Boolean(current)&&row.dataset.homeTrackRef===current);
 }
 requestAnimationFrame(()=>refreshSongMarqueeMetrics());
}

function homeRecentRowHTML(item){
 const secondary=String(item?.sourceTitle??item?.channel??"").trim();
 const playing=player.currentItem?.ref===item.ref;
 return `<tr class="${playing?"playing":""}" tabindex="0" aria-label="Play ${escapeHTML(trackTitle(item))}" data-home-track-ref="${escapeHTML(item.ref)}" data-track-ref="${escapeHTML(item.ref)}">
   <td class="song-title-cell">
    <div class="song-main">
     ${trackArtworkHTML(item)}
     <div class="song-copy">
      ${marqueeTextHTML(trackTitle(item),{tag:"strong"})}
      ${secondary?marqueeTextHTML(secondary,{tag:"small"}):""}
     </div>
    </div>
   </td>
   <td class="col-added"><span class="small">${escapeHTML(formatLibraryAdded(item.addedAt))}</span></td>
   <td class="col-artist">${marqueeTextHTML(trackArtist(item),{className:"clip2"})}</td>
   <td class="col-album">${marqueeTextHTML(String(item?.album??"").trim()||"Unknown album",{className:"clip2"})}</td>
   <td class="col-duration">${escapeHTML(formatSeconds(item?.durationSeconds))}</td>
   <td class="col-actions">
    <div class="song-actions">
     <button data-track-action="play" data-vex-action="vexstream.player.play-track" title="${escapeHTML(resolveText(locale,"playback.play"))}" aria-label="${escapeHTML(resolveText(locale,"playback.play"))}">▶</button>
     <button data-track-action="append" data-vex-action="vexstream.player.add-queue" title="${escapeHTML(resolveText(locale,"player.add-to-queue"))}" aria-label="${escapeHTML(resolveText(locale,"player.add-to-queue"))}">☷＋</button>
     <details class="song-more">
      <summary aria-label="Track actions" title="Track actions">⋯</summary>
      <div class="song-more-menu">
       <button data-track-action="play-next" data-vex-action="vexstream.player.play-next">${escapeHTML(resolveText(locale,"player.play-next"))}</button>
       <button data-track-action="explore" data-vex-action="vexstream.explore.from-track">${escapeHTML(resolveText(locale,"explore.open"))}</button>
       <button data-track-action="edit">${escapeHTML(resolveText(locale,"library.edit"))}</button>
       <button data-track-action="trash">${escapeHTML(resolveText(locale,"trash.move"))}</button>
       ${trackPlaylistActionHTML()}
      </div>
     </details>
    </div>
   </td>
  </tr>`;
}

function renderHome(){
 const projection=projectLibraryHome(catalog,{recentLimit:8});
 homeSongCount.textContent=String(projection.songCount);
 homeAlbumCount.textContent=String(projection.albumCount);
 homeArtistCount.textContent=String(projection.artistCount);
 homeSourceCount.textContent=String(sourceConfig.sources.length);
 recentSongs.innerHTML=projection.recent.length
  ?`<div class="song-table-wrap home-recent-table-wrap">
     <table class="song-table home-recent-table">
      <thead>
       <tr>
        ${songColumnHeader("title","Title","song-title-cell",{sortable:false})}
        ${songColumnHeader("added","Added","col-added",{sortable:false})}
        ${songColumnHeader("artist","Artist","col-artist",{sortable:false})}
        ${songColumnHeader("album","Album","col-album",{sortable:false})}
        ${songColumnHeader("duration","Time","col-duration",{sortable:false})}
        <th class="col-actions"></th>
       </tr>
      </thead>
      <tbody>
       ${projection.recent.map(homeRecentRowHTML).join("")}
      </tbody>
     </table>
    </div>`
  : `<div class="empty-state">${escapeHTML(resolveText(locale,"library.empty.filtered"))}</div>`;
}

function renderShell(projection=shellNavigation.currentProjection){
 document.body.dataset.shellCurrentView=projection.view;
 libraryUtilityBar.hidden=projection.view===ShellView.DISCOVER;
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
 renderDiscoverNavigation();
}

function renderDiscoverNavigation(
 projection=discoverNavigation.currentProjection
){
 const discoverVisible=
  shellNavigation.currentProjection.view===ShellView.DISCOVER;
 document.body.dataset.discoverMode=projection.mode;
 for(const button of discoverModeButtons){
  const selected=button.dataset.discoverMode===projection.mode;
  button.classList.toggle("active",selected);
  button.setAttribute("aria-pressed",String(selected));
 }
 for(const panel of discoverPanels){
  panel.hidden=
   !discoverVisible||
   panel.dataset.discoverPanel!==projection.mode;
 }
}

shellNavigation.subscribe(renderShell);
discoverNavigation.subscribe(renderDiscoverNavigation);
librarySelection.subscribe(renderTrackChoices);
playlistNavigation.subscribe(renderPlaylistChoices);

function currentLibraryView(){
 return projectLibraryView(catalog,{
  query:libraryView.query,
  genre:libraryView.genre,
  sort:libraryView.sort,
  sortStack:libraryView.sortStack
 });
}

function libraryPlaybackView(){
 return projectLibraryView(catalog,{
  query:"",
  genre:"all",
  sort:libraryView.sort,
  sortStack:libraryView.sortStack
 });
}

function libraryPlaybackRefs(){
 return libraryPlaybackView().map(item=>item.ref);
}

function applyCatalog(nextCatalog){
 catalog=nextCatalog;
 catalogRevision+=1;
 queueWindowRenderKey="";
 byRef=new Map(catalog.map(item=>[item.ref,item]));
 refs=catalog.map(item=>item.ref);
 librarySelection.reconcile(refs);
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
 playlistNavigation.reconcile(playlists.map(playlist=>playlist.id));
 renderPlaylistChoices();
 renderTrackChoices();
 return playlists;
}

async function refreshTrash(){
 trashSnapshot=await loadTrash();
 renderTrash();
 return trashSnapshot;
}

function sourceDetailValue(value,{fallback="—"}={}){
 const text=String(value??"").trim();
 return text||fallback;
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

 const start=Number(item.playbackStartSeconds)||0;
 const end=item.playbackEndSeconds==null?null:Number(item.playbackEndSeconds);
 const cropped=start>0||Number.isFinite(end);
 playbackWindowSummary.innerHTML=cropped
  ?`<strong>Playback window active.</strong>
    <span>${escapeHTML(formatSeconds(start))} → ${escapeHTML(Number.isFinite(end)?formatSeconds(end):"source end")} · non-destructive</span>`
  :"<strong>Full track.</strong><span>Playback uses the original media duration; no source bytes are rewritten.</span>";

 const sourceRows=[
  ["Source title",sourceDetailValue(item.sourceTitle,{fallback:"Same as current title"})],
  ["Source artist",sourceDetailValue(item.sourceArtist,{fallback:"Unresolved"})],
  ["Source album",sourceDetailValue(item.sourceAlbum,{fallback:"—"})],
  ["Source genre",sourceDetailValue(item.sourceGenre,{fallback:"—"})],
  ["Provider creator",sourceDetailValue(item.channel,{fallback:"—"})],
  ["Provider ID",sourceDetailValue(item.providerId,{fallback:"—"})],
  ["Source URL",sourceDetailValue(item.sourceURL,{fallback:"—"})],
  ["Source duration",item.durationSeconds>0?formatSeconds(item.durationSeconds):"—"]
 ];
 trackSourceDetails.innerHTML=sourceRows.map(([label,value])=>`
   <div class="track-source-row">
    <strong>${escapeHTML(label)}</strong>
    <span>${escapeHTML(value)}</span>
   </div>`
  ).join("");
}

function validTrashIDs(values){
 const live=new Set(trashSnapshot.trash.map(item=>item.id));
 return [...new Set(values)].filter(id=>live.has(id));
}

function openTrashPurgeConfirmation(ids){
 const valid=validTrashIDs(ids);
 if(!valid.length)return;
 pendingTrashPurgeIDs=Object.freeze(valid);
 preserveDeletedMetadata.checked=true;
 trashPurgeConfirmSummary.textContent=resolveText(locale,"trash.purge-confirm",{
  count:valid.length,
  suffix:valid.length===1?"":"s"
 });
 trashPurgeConfirm.hidden=false;
}

function closeTrashPurgeConfirmation(){
 pendingTrashPurgeIDs=Object.freeze([]);
 trashPurgeConfirm.hidden=true;
}

function renderTrash(){
 const live=new Set(trashSnapshot.trash.map(item=>item.id));
 for(const id of [...selectedTrashIDs]){
  if(!live.has(id))selectedTrashIDs.delete(id);
 }
 if(pendingTrashPurgeIDs.some(id=>!live.has(id))){
  closeTrashPurgeConfirmation();
 }

 const trashCountValue=trashSnapshot.trash.length;
 const historyCountValue=trashSnapshot.deletedHistory.length;
 trashNavCount.textContent=trashCountValue?`(${trashCountValue})`:"";
 trashSummary.textContent="A reversible file trash first; optional metadata history after permanent erase.";
 trashCount.textContent=resolveText(locale,"trash.count",{
  count:trashCountValue,
  suffix:trashCountValue===1?"":"s"
 });
 deletedHistoryCount.textContent=resolveText(locale,"trash.history-count",{
  count:historyCountValue,
  suffix:historyCountValue===1?"":"s"
 });
 trashSelectionBar.hidden=selectedTrashIDs.size===0;
 trashSelectionCount.textContent=resolveText(locale,"trash.selection-count",{
  count:selectedTrashIDs.size
 });

 trashItems.innerHTML=trashCountValue
  ?trashSnapshot.trash.map(item=>`<article class="trash-row ${selectedTrashIDs.has(item.id)?"selected":""}" data-trash-id="${escapeHTML(item.id)}">
      <label class="trash-select">
       <input type="checkbox" data-trash-select aria-label="Select ${escapeHTML(item.track?.title||resolveText(locale,"library.unknown-title"))}" ${selectedTrashIDs.has(item.id)?"checked":""}>
      </label>
      <div class="trash-copy">
       <strong>${escapeHTML(item.track?.title||resolveText(locale,"library.unknown-title"))}</strong>
       <small>${escapeHTML(item.track?.artist||resolveText(locale,"library.unknown-artist"))} · trashed ${escapeHTML(item.trashedAt||"")}</small>
      </div>
      <div class="trash-actions">
       <button data-trash-action="restore">${escapeHTML(resolveText(locale,"trash.restore"))}</button>
       <button data-trash-action="purge" class="danger">Permanently delete…</button>
      </div>
     </article>`).join("")
  :`<div class="empty-state">${escapeHTML(resolveText(locale,"trash.empty"))}</div>`;

 deletedHistory.innerHTML=historyCountValue
  ?trashSnapshot.deletedHistory.slice(0,20).map(item=>`<div class="deleted-row">
      <span>
       <strong>${escapeHTML(item.track?.title||resolveText(locale,"library.unknown-title"))}</strong>
       <small>${escapeHTML(item.track?.artist||resolveText(locale,"library.unknown-artist"))}</small>
      </span>
      <small>${escapeHTML(item.deletedAt||"")}</small>
     </div>`).join("")
  :`<div class="empty-state">No metadata-only deletion history yet.</div>`;

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

const shuffleRefs=window.__VEXSTREAM_V4_SHUFFLE??randomShuffle;
const queue=createQueueConnection({
 initialState:QueueState.empty(),
 shuffle:shuffleRefs
});

const playback=createHTMLMediaPlaybackConnection({
 media:audio,
 hasCurrentMedia:()=>Boolean(currentQueueRef(queue.currentProjection.state)&&audio.dataset.mediaRef),
 restartCurrent:async media=>{
  const item=byRef.get(currentQueueRef(queue.currentProjection.state));
  media.currentTime=Math.max(0,Number(item?.playbackStartSeconds)||0);
 }
});

const timeline=createHTMLMediaPlaybackTimelineConnection({
 media:audio,
 currentWindow:()=>{
  const item=byRef.get(currentQueueRef(queue.currentProjection.state));
  return item?{
   startSeconds:item.playbackStartSeconds,
   endSeconds:item.playbackEndSeconds
  }:null;
 }
});

const playbackVolume=createHTMLMediaPlaybackVolumeConnection({
 media:audio,
 initialVolume:.8
});

const player=createPlayerComposition({
 queue,
 playback,
 timeline,
 media:audio,
 resolveMedia:ref=>byRef.get(ref),
 defaultContextRefs:()=>libraryPlaybackRefs(),
 onCurrentMedia:item=>{
  renderNowPlaying(item);
 }
});

timeline.subscribe(renderPlaybackTimeline);
playbackVolume.subscribe(renderPlaybackVolume);

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

function holdPlayerSessionPersistence(failure){
 playerSessionPersistenceEnabled=false;
 currentSessionRestoration=Object.freeze({
  kind:"persistence-held",
  queue:null,
  droppedRefs:Object.freeze([]),
  currentRef:"",
  detail:failure?.message??String(failure)
 });
 render();
}

function enqueuePlayerSessionSave(snapshot){
 playerSessionSaveChain=playerSessionSaveChain
  .then(()=>savePlayerSession(snapshot))
  .catch(failure=>{
   holdPlayerSessionPersistence(failure);
  });
 return playerSessionSaveChain;
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
  if(snapshot)enqueuePlayerSessionSave(snapshot);
 },60);
}

async function flushPlayerSessionSave(){
 if(playerSessionSaveTimer){
  clearTimeout(playerSessionSaveTimer);
  playerSessionSaveTimer=null;
 }
 const snapshot=pendingPlayerSessionState;
 pendingPlayerSessionState=null;
 if(snapshot&&playerSessionPersistenceEnabled){
  enqueuePlayerSessionSave(snapshot);
 }
 await playerSessionSaveChain;
 if(currentSessionRestoration?.kind==="persistence-held"){
  throw new Error(
   currentSessionRestoration.detail||
   "Player session could not be saved before Exit"
  );
 }
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

function setTransportIcon(button,presentation,icon){
 const label=resolveText(locale,presentation.textRef);
 button.textContent=icon;
 button.setAttribute("aria-label",label);
 button.title=label;
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

function renderImportRuntimeStatus(){
 if(!importRuntimeNotice)return;
 if(!importRuntimeState){
  importRuntimeNotice.dataset.ready="unknown";
  importRuntimeNotice.textContent="Checking YouTube import tools…";
  return;
 }
 const detail=importRuntimeState.detail
  ? ` <span>${escapeHTML(importRuntimeState.detail)}</span>`
  : "";
 let title="YouTube import needs one-time setup.";
 let action=`<button type="button" class="primary" data-import-runtime-action="setup">Set up import tools</button>`;
 if(importRuntimeState.importReady){
  title="YouTube import is ready.";
  action="";
 }else if(importRuntimeState.searchReady){
  title="YouTube search is ready; MP3 download still needs FFmpeg.";
  action=`<button type="button" class="primary" data-import-runtime-action="setup">Finish import setup</button>`;
 }
 importRuntimeNotice.dataset.ready=String(importRuntimeState.importReady);
 importRuntimeNotice.innerHTML=`<div><strong>${escapeHTML(title)}</strong>${detail}</div>${action}`;
}

async function refreshImportRuntime(){
 importRuntimeState=await loadImportRuntimeStatus();
 renderImportRuntimeStatus();
 return importRuntimeState;
}

async function prepareImportRuntime(){
 importRuntimeNotice.dataset.ready="false";
 importRuntimeNotice.innerHTML="<div><strong>Setting up YouTube import tools…</strong></div>";
 importRuntimeState=await setupImportRuntime();
 renderImportRuntimeStatus();
 return importRuntimeState;
}

async function ensureImportRuntimeReady(){
 let runtime=await refreshImportRuntime();
 if(runtime.importReady)return runtime;
 runtime=await prepareImportRuntime();
 if(runtime.importReady)return runtime;
 throw new Error(
  runtime.detail||
  "YouTube search can still work, but MP3 download needs FFmpeg before this import can start."
 );
}

function importCompletionKey(snapshot=currentImportJobs){
 return (snapshot?.completed??[])
  .filter(job=>job.status==="done")
  .map(job=>`${job.id}:${job.completedAt??""}:${(job.trackIds??[]).join(",")}`)
  .sort()
  .join("|");
}

function scheduleImportJobRefresh(delay=700){
 if(importRefreshTimer!==null){
  clearTimeout(importRefreshTimer);
  importRefreshTimer=null;
 }
 if((currentImportJobs?.active?.length??0)===0)return;
 importRefreshTimer=setTimeout(()=>{
  importRefreshTimer=null;
  void guarded(()=>refreshImportJobs());
 },delay);
}

function stopImportDuplicatePreview({resume=true}={}){
 if(importDuplicatePreviewTimer!==null){
  clearTimeout(importDuplicatePreviewTimer);
  importDuplicatePreviewTimer=null;
 }
 importDuplicatePreviewAudio.pause();
 importDuplicatePreviewAudio.removeAttribute("src");
 importDuplicatePreviewAudio.load();
 importDuplicatePreviewAudio.onloadedmetadata=null;
 importDuplicatePreviewAudio.onerror=null;
 for(const button of importDuplicateDetails.querySelectorAll("[data-import-duplicate-preview]")){
  button.classList.remove("previewing");
  button.textContent="Preview 12s";
 }
 if(resume&&resumeAfterImportDuplicatePreview){
  audio.play().catch(()=>{});
 }
 resumeAfterImportDuplicatePreview=false;
}

function importDuplicateRelationLabel(value){
 return ({
  SAME_SOURCE:"Same source",
  LIKELY_SAME_RECORDING:"Likely same recording",
  SAME_SONG_DIFFERENT_EDIT_POSSIBLE:"Likely same song · different edit possible",
  SAME_SONG_POSSIBLE:"Same song possible",
  POSSIBLE_MATCH:"Possible match"
 })[value]||"Possible match";
}

function importQualityLabel(value){
 if(!value)return "Unknown";
 const bits=[];
 if(value.codec)bits.push(String(value.codec).toUpperCase());
 if(Number(value.bitrateKbps)>0)bits.push(`~${Math.round(Number(value.bitrateKbps))} kbps`);
 if(Number(value.sampleRateHz)>0)bits.push(`${Math.round(Number(value.sampleRateHz)/100)/10} kHz`);
 return bits.join(" · ")||"Unknown";
}

function importDuplicateReviewHTML(review){
 const items=review?.duplicateItems??[];
 if(!items.length)return "";
 const rows=items.map(item=>{
  const match=item.matches?.[0];
  if(!match)return "";
  const incoming=item.kind==="section"
   ? `Track section ${item.index}: ${item.title}`
   : `Incoming source: ${item.title}`;
  const sameSource=match.relation==="SAME_SOURCE";
  const heading=sameSource
   ?"Already in your Library — same source"
   :"Possible duplicate already in your Library";
  const artwork=match.thumbnailUrl
   ?`<img class="import-duplicate-art" src="${escapeHTML(match.thumbnailUrl)}" alt="">`
   :'<div class="import-duplicate-art fallback">♫</div>';
  const reasons=(match.reasons??[]).slice(0,4).join(" · ");
  const quality=match.qualityComparison?.summary||
   "Not enough format data to compare audio quality before download.";
  const preview=match.trackId&&byRef.has(match.trackId)
   ?`<button type="button" data-import-duplicate-preview="${escapeHTML(match.trackId)}">Preview 12s</button>`
   :"";
  return `<section class="import-duplicate-item ${sameSource?"same-source":""}">
    <div class="import-duplicate-heading">
     <strong>${escapeHTML(heading)}</strong>
     <span>${escapeHTML(incoming)}</span>
    </div>
    <div class="import-duplicate-match">
     ${artwork}
     <div class="import-duplicate-copy">
      <div><strong>${escapeHTML(match.title||"Existing Library track")}</strong>
       <span class="import-duplicate-relation">${escapeHTML(importDuplicateRelationLabel(match.relation))}</span>
      </div>
      <div>${escapeHTML(match.artist||"Unknown artist")} · ${escapeHTML(match.album||"Unknown album")} · ${formatSeconds(match.duration)}</div>
      <div>${escapeHTML(match.confidence||"possible")} match${reasons?` · ${escapeHTML(reasons)}`:""}</div>
      <div><strong>Existing:</strong> ${escapeHTML(importQualityLabel(match.localQuality))}
       &nbsp; <strong>Source:</strong> ${escapeHTML(importQualityLabel(item.sourceQuality))}</div>
      <div><strong>Quality check:</strong> ${escapeHTML(quality)}</div>
     </div>
     ${preview}
    </div>
   </section>`;
 }).filter(Boolean).join("");
 if(!rows)return "";
 return `<div class="import-duplicate-review">
   ${rows}
   <div class="import-duplicate-advisory">
    Duplicate matching is advisory. Compare the existing Library track before downloading another copy; different edits, remixes, live versions, or intentionally different-quality copies may still be valid.
   </div>
  </div>`;
}

function previewImportDuplicate(ref,button){
 const track=byRef.get(ref);
 if(!track)return;
 stopImportDuplicatePreview({resume:false});
 resumeAfterImportDuplicatePreview=!audio.paused;
 if(resumeAfterImportDuplicatePreview)audio.pause();
 button.classList.add("previewing");
 button.textContent="Playing…";
 importDuplicatePreviewAudio.src="/media/"+encodeURIComponent(ref);
 importDuplicatePreviewAudio.onloadedmetadata=()=>{
  const duration=Number.isFinite(importDuplicatePreviewAudio.duration)
   ?importDuplicatePreviewAudio.duration
   :Number(track.durationSeconds)||0;
  importDuplicatePreviewAudio.currentTime=duration>60?Math.min(30,duration*.2):0;
  importDuplicatePreviewAudio.play().catch(()=>{});
  importDuplicatePreviewTimer=setTimeout(
   ()=>stopImportDuplicatePreview(),
   12000
  );
 };
 importDuplicatePreviewAudio.onerror=()=>stopImportDuplicatePreview();
}

async function loadImportReview(url,{selectYouTube=false}={}){
 const value=String(url??"").trim();
 if(!value)throw new Error("Paste a YouTube link to review");
 stopImportDuplicatePreview();
 if(selectYouTube)discoverNavigation.select(DiscoverMode.YOUTUBE);
 youtubeImportURL.value=value;
 currentImportReview=null;
 importReviewLoadingState=true;
 importReviewLoadingURL=value;
 renderImportReview();
 try{
  const [reviewed,destinations]=await Promise.all([
   inspectImport(value),
   loadImportDestinations()
  ]);
  currentImportReview=formImportReview(reviewed,destinations);
  return currentImportReview;
 }finally{
  importReviewLoadingState=false;
  importReviewLoadingURL="";
 }
}

function renderImportReview(){
 if(importReviewLoadingState){
  youtubeImportEntry.hidden=true;
  importReviewPanel.hidden=false;
  importReviewClose.hidden=true;
  importReviewLoading.hidden=false;
  importReviewContent.hidden=true;
  importReviewLoadingDetail.textContent=
   "Fetching title, artwork, track layout, and duplicate matches from your current Library. Nothing is downloading yet.";
  importReviewPanel.dataset.reviewState="loading";
  return;
 }
 importReviewClose.hidden=false;
 importReviewLoading.hidden=true;
 importReviewContent.hidden=false;
 importReviewPanel.dataset.reviewState=currentImportReview?"ready":"idle";
 if(!currentImportReview){
  importReviewPanel.hidden=true;
  youtubeImportEntry.hidden=false;
  importReviewThumbnail.hidden=true;
  importReviewThumbnail.removeAttribute("src");
  importReviewThumbnailFallback.hidden=false;
  importDuplicateDetails.innerHTML="";
  return;
 }
 youtubeImportEntry.hidden=true;
 importReviewPanel.hidden=false;
 importReviewSourceTitle.textContent=currentImportReview.title;
 importReviewSourceCreator.textContent=currentImportReview.creator||"Unknown creator";
 importReviewDuration.textContent=currentImportReview.durationSeconds
  ? `${formatSeconds(currentImportReview.durationSeconds)} source duration`
  : "";
 if(currentImportReview.thumbnail){
  importReviewThumbnail.src=currentImportReview.thumbnail;
  importReviewThumbnail.hidden=false;
  importReviewThumbnailFallback.hidden=true;
 }else{
  importReviewThumbnail.hidden=true;
  importReviewThumbnail.removeAttribute("src");
  importReviewThumbnailFallback.hidden=false;
 }
 importDuplicateSummary.textContent=currentImportReview.duplicateItemCount
  ? resolveText(locale,"import.duplicates.present",{
      count:currentImportReview.duplicateItemCount
    })
  : resolveText(locale,"import.duplicates.none");
 importDuplicateDetails.innerHTML=importDuplicateReviewHTML(currentImportReview);

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

function importJobTime(value){
 if(!value)return "";
 const date=new Date(value);
 if(Number.isNaN(date.getTime()))return "";
 return date.toLocaleTimeString([],{hour:"numeric",minute:"2-digit"});
}

function importJobStatusLabel(job){
 if(job.status==="done")return "Added";
 if(job.status==="committing")return "Adding";
 if(job.status==="ready")return "Needs attention";
 if(job.status==="running")return "Downloading";
 if(job.status==="queued")return "Queued";
 if(job.status==="cancelling")return "Cancelling";
 if(job.status==="cancelled")return "Cancelled";
 if(job.status==="discarded")return "Discarded";
 if(job.status==="failed")return "Failed";
 return job.status||"Import";
}

function importJobCard(job,group){
 const recovery=currentImportJobs.recovery[job.id]??"";
 const problem=job.problem?.title||job.problem?.message||"";
 const actions=[];
 if(["queued","running","cancelling"].includes(job.status)){
  actions.push(`<button data-import-job-action="cancel">${escapeHTML(resolveText(locale,"import.cancel"))}</button>`);
 }
 if(job.status==="ready"){
  actions.push(`<button class="primary" data-import-job-action="retry">${escapeHTML(resolveText(locale,"import.retry"))}</button>`);
  actions.push(`<button data-import-job-action="discard">${escapeHTML(resolveText(locale,"import.discard"))}</button>`);
 }
 if(job.status==="failed"){
  actions.push(`<button data-import-job-action="discard">${escapeHTML(resolveText(locale,"import.discard"))}</button>`);
 }
 if(recovery==="COMMIT_UNCERTAIN"){
  actions.push(`<button class="primary" data-import-job-action="recover">${escapeHTML(resolveText(locale,"import.recover"))}</button>`);
 }
 if(job.status==="done"&&(job.trackIds??[]).some(ref=>byRef.has(ref))){
  actions.push('<button data-import-job-action="view">View in Library</button>');
 }

 const pct=Math.max(0,Math.min(100,Number(job.progress)||0));
 const thumbnail=job.thumbnail
  ? `<img class="import-job-thumbnail" src="${escapeHTML(job.thumbnail)}" alt="">`
  : '<div class="import-job-thumbnail fallback">♫</div>';
 const creator=job.creator
  ? `<div class="import-job-creator">${escapeHTML(job.creator)}</div>`
  : "";
 const stage=job.stage||importJobStatusLabel(job);
 const detail=(job.detail||problem||"").trim();
 const detailHTML=detail&&detail!==stage
  ? `<div class="import-job-detail">${escapeHTML(detail)}</div>`
  : "";
 const activeProgress=group==="active"
  ? `<div class="import-progress-row">
      <div class="import-progress-track" role="progressbar" aria-label="${escapeHTML(stage)}" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${pct}">
       <div class="import-progress-fill" style="width:${pct}%"></div>
      </div>
      <strong class="import-progress-value">${pct}%</strong>
    </div>`
  : "";
 const when=importJobTime(group==="completed"?job.completedAt:job.startedAt);

 return `<article class="import-job import-job-${escapeHTML(group)}" data-import-job-id="${escapeHTML(job.id)}">
   ${thumbnail}
   <div class="import-job-main">
    <div class="import-job-title-row">
     <strong>${escapeHTML(job.title||resolveText(locale,"import.untitled"))}</strong>
     <span class="import-job-status status-${escapeHTML(job.status||"unknown")}">${escapeHTML(importJobStatusLabel(job))}</span>
    </div>
    ${creator}
    <div class="import-job-stage">${escapeHTML(stage)}</div>
    ${detailHTML}
    ${activeProgress}
   </div>
   <div class="import-job-side">
    <div class="import-job-actions">${actions.join("")}</div>
    ${when?`<time class="import-job-time">${escapeHTML(when)}</time>`:""}
   </div>
  </article>`;
}

function importJobSection(title,summary,jobs,group){
 if(!jobs.length)return "";
 return `<section class="import-activity-group import-activity-${escapeHTML(group)}">
   <div class="import-activity-group-head">
    <div><strong>${escapeHTML(title)}</strong><span>${escapeHTML(summary)}</span></div>
    <span class="import-activity-count">${jobs.length}</span>
   </div>
   <div class="import-activity-list">${jobs.map(job=>importJobCard(job,group)).join("")}</div>
  </section>`;
}

function renderImportJobs(){
 const active=currentImportJobs.active??[];
 const ready=currentImportJobs.ready??[];
 const completed=currentImportJobs.completed??[];
 if(!active.length&&!ready.length&&!completed.length){
  importJobs.innerHTML=`<div class="import-activity-empty">${escapeHTML(resolveText(locale,"import.jobs.empty"))}</div>`;
  return;
 }

 const summary=active.length
  ? `${active.length} import${active.length===1?"":"s"} in progress · VexStream will add finished media to the Library automatically.`
  : ready.length
    ? `${ready.length} import${ready.length===1?"":"s"} need attention before the Library add can finish.`
    : `${completed.length} recent import event${completed.length===1?"":"s"}.`;

 importJobs.innerHTML=`<div class="import-activity-summary">${escapeHTML(summary)}</div>
  ${importJobSection("In progress","Download and preparation are happening now.",active,"active")}
  ${importJobSection("Needs attention","Prepared media is still safe in temporary staging.",ready,"ready")}
  ${importJobSection("Recent","Completed, cancelled, discarded, or failed imports.",completed,"completed")}`;
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
 if(importRefreshTimer!==null){
  clearTimeout(importRefreshTimer);
  importRefreshTimer=null;
 }
 try{
  const nextJobs=await loadImportJobs();
  const nextCompletionKey=importCompletionKey(nextJobs);
  const libraryChanged=
   nextCompletionKey!==""&&
   nextCompletionKey!==observedImportCompletionKey;
  currentImportJobs=nextJobs;
  renderImportJobs();
  if(libraryChanged){
   await refreshCatalog();
  }
  observedImportCompletionKey=nextCompletionKey;
  return currentImportJobs;
 }finally{
  scheduleImportJobRefresh();
 }
}

function renderSystemStatus(){
 if(!systemIdentity){
  currentSystemStatus=null;
  systemStatusBadge.hidden=true;
  return;
 }
 const projection=projectSystemStatus({
  identity:systemIdentity,
  catalog,
  sourceConfig,
  sessionRestoration:currentSessionRestoration,
  importJobs:currentImportJobs,
  trash:trashSnapshot
 });
 currentSystemStatus=projection;
 const attention=projection.attentionCount;
 systemStatusBadge.hidden=attention===0;
 systemStatusBadge.textContent=String(attention);
 systemStatusButton.setAttribute(
  "aria-label",
  attention
   ?resolveText(locale,"system.status.attention",{count:attention})
   :resolveText(locale,"system.status.title")
 );
 systemSummary.innerHTML=`<strong>${escapeHTML(resolveText(
  locale,
  attention?"system.status.attention":"system.status.healthy",
  {count:attention}
 ))}</strong>`;
 systemBuildStatus.textContent=`VexStream ${projection.version}`;
 systemBuildRef.textContent=projection.buildRef;
 systemLibraryStatus.textContent=resolveText(locale,"system.status.library-summary",{
  tracks:projection.trackCount,
  sources:projection.sourceCount
 });
 systemTrashStatus.textContent=resolveText(locale,"system.status.trash-summary",{
  count:projection.trashCount
 });
 systemSessionStatus.textContent=sessionStatusText(currentSessionRestoration);
 const importAttention=
  projection.readyImports+
  projection.importIssues+
  projection.importRecovery;
 systemImportStatus.textContent=resolveText(locale,"system.status.import-summary",{
  active:projection.activeImports,
  attention:importAttention
 });
 const attentionRows=[
  ...projection.warnings.map(warning=>({
   message:resolveText(locale,"system.status.startup-warning",{warning})
  })),
  ...(projection.sessionHeld
   ?[{
      message:resolveText(
       locale,
       currentSessionRestoration?.kind==="held-current-unavailable"
        ?"system.status.session-held-current"
        :currentSessionRestoration?.kind==="held-schema"
          ?"system.status.session-held-schema"
          :"system.status.session-save-held"
      ),
      action:["held-current-unavailable","held-schema"].includes(
       currentSessionRestoration?.kind
      )
       ?`<button type="button" data-system-action="reset-listening-session">${escapeHTML(resolveText(locale,"system.status.session-reset"))}</button>`
       :""
    }]
   :[]),
  ...(importAttention
   ?[{message:resolveText(locale,"system.status.import-attention")}]
   :[]),
  ...(projection.pendingPurge
   ?[{message:resolveText(locale,"system.status.trash-recovery")}]
   :[])
 ];
 systemAttention.innerHTML=attentionRows.length
  ?`<ul class="system-attention-list">${attentionRows.map(row=>
      `<li><span>${escapeHTML(row.message)}</span>${row.action??""}</li>`
     ).join("")}</ul>`
  :`<span class="muted">${escapeHTML(resolveText(locale,"system.status.no-warnings"))}</span>`;
}

function sessionIdentityText(session){
 const version=String(session?.version??"").trim();
 const build=String(session?.buildRef??"").trim();
 const pid=Number(session?.pid??0);
 return [
  version?`VexStream ${version}`:"VexStream",
  build?build.slice(0,12):"",
  pid>0?`PID ${pid}`:""
 ].filter(Boolean).join(" · ");
}

function renderOldSessionPlan(plan){
 if(!oldSessionPlan)return;
 oldSessionPlan.hidden=false;
 if(plan?.scanError){
  oldSessionPlan.innerHTML=`<div class="system-session-error">${escapeHTML(plan.scanError)}</div>`;
  return;
 }
 const others=(plan?.sessions??[]).filter(session=>!session.current);
 if(!others.length){
  oldSessionPlan.innerHTML=`<div class="muted">${escapeHTML(resolveText(locale,"system.sessions.none"))}</div>`;
  return;
 }
 const eligible=plan?.eligible??[];
 const held=plan?.held??[];
 const eligibleHTML=eligible.length
  ?`<div class="system-session-group">
      <strong>${escapeHTML(resolveText(locale,"system.sessions.eligible"))}</strong>
      ${eligible.map(session=>`<label class="system-session-row eligible">
       <input type="checkbox" data-old-session-pid="${session.pid}">
       <span><b>${escapeHTML(sessionIdentityText(session))}</b><small>${escapeHTML(session.reason)}</small></span>
      </label>`).join("")}
     </div>`
  :"";
 const heldHTML=held.length
  ?`<div class="system-session-group">
      <strong>${escapeHTML(resolveText(locale,"system.sessions.held"))}</strong>
      ${held.map(session=>`<div class="system-session-row held">
       <span aria-hidden="true">🛡</span>
       <span><b>${escapeHTML(sessionIdentityText(session))}</b><small>${escapeHTML(session.reason)}</small></span>
      </div>`).join("")}
     </div>`
  :"";
 oldSessionPlan.innerHTML=`
  ${eligibleHTML}
  ${heldHTML}
  <div class="muted system-session-warning">${escapeHTML(plan.warning??"")}</div>
  ${eligible.length?`<button id="retireOldSessions" type="button" disabled>${escapeHTML(resolveText(locale,"system.sessions.retire"))}</button>`:""}`;
}

function updateOldSessionRetireButton(){
 const button=oldSessionPlan?.querySelector("#retireOldSessions");
 if(!button)return;
 button.disabled=!oldSessionPlan.querySelector("[data-old-session-pid]:checked");
}

function renderPlaybackTimeline(
 projection=timeline.currentProjection
){
 const ready=projection.kind==="ready"&&projection.seekAvailable;
 seek.disabled=!ready;
 seek.value=ready?String(Math.round(projection.normalized*1000)):"0";
 curTime.textContent=formatSeconds(ready?projection.elapsedSeconds:0);
 durTime.textContent=formatSeconds(ready?projection.durationSeconds:0);
}

function renderPlaybackVolume(
 projection=playbackVolume.currentProjection
){
 volumeInput.value=String(projection.volume);
 volumeInput.title=`Volume ${projection.percent}%`;
}

function setQueueDrawerOpen(open){
 const nextOpen=Boolean(open);
 const opening=nextOpen&&!queueDrawerOpen;
 queueDrawerOpen=nextOpen;
 if(opening){
  queueWindowInitialized=false;
  queueWindowRenderKey="";
 }
 render();
 if(opening){
  requestAnimationFrame(()=>{
   const state=queue.currentProjection.state;
   if(state.kind!=="positioned")return;
   const target=Math.max(0,state.currentIndex-2);
   queueTimelinePane.scrollTop=
    queueList.offsetTop+(target*QUEUE_ROW_HEIGHT);
  });
 }
}

function toggleQueueDrawer(){
 setQueueDrawerOpen(!queueDrawerOpen);
}


function renderQueueCurrentPane(state){
 const ref=currentQueueRef(state);
 const item=ref?byRef.get(ref):null;
 if(!item){
  queueCurrentPane.removeAttribute("data-track-ref");
  queueCurrentPane.innerHTML=
   '<div class="queue-current-empty"><strong>'+
   escapeHTML(resolveText(locale,"player.nothing.title"))+
   '</strong><span>'+
   escapeHTML(resolveText(locale,"player.nothing.artist"))+
   '</span></div>';
  return;
 }

 const titleText=trackTitle(item);
 const artwork=String(item?.thumbnailURL??"").trim();
 queueCurrentPane.dataset.trackRef=ref;
 queueCurrentPane.innerHTML=
  '<div class="queue-current-art'+(artwork?' has-art':'')+'">'+
   (artwork
    ?'<img src="'+escapeHTML(artwork)+'" alt="">'
    :'<span aria-hidden="true">♫</span>')+
  '</div>'+
  '<div class="eyebrow">'+
   escapeHTML(resolveText(locale,"queue.current.eyebrow"))+
  '</div>'+
  marqueeTextHTML(titleText,{tag:"h2",className:"queue-current-title"})+
  '<div class="queue-current-meta">'+nowLibraryRelationsHTML(item)+'</div>'+
  '<details class="queue-current-actions"><summary>'+
   escapeHTML(resolveText(locale,"queue.current.actions"))+
  '</summary><div class="queue-current-action-menu">'+
   '<button data-track-action="explore" data-vex-action="vexstream.explore.from-track">'+
    escapeHTML(resolveText(locale,"explore.open"))+
   '</button>'+
   '<button data-track-action="edit">'+
    escapeHTML(resolveText(locale,"library.edit"))+
   '</button>'+
   '<button data-track-action="trash">'+
    escapeHTML(resolveText(locale,"trash.move"))+
   '</button>'+
   trackPlaylistActionHTML()+
  '</div></details>';
}


function queueTimelineRowHTML(state,ref,index){
 const item=byRef.get(ref);
 const isCurrent=index===state.currentIndex;
 const currentClass=isCurrent?"current":"";
 const stateTextRef=index<state.currentIndex
  ?"queue.item.played"
  :isCurrent
    ?"queue.item.current"
    :"queue.item.up-next";
 const stateLabel=resolveText(locale,stateTextRef);
 const titleText=trackTitle(item);
 const album=String(item?.album??"").trim()||"Unknown album";
 const metadata=[trackArtist(item),album];
 const accessibleIdentity=[stateLabel,titleText,...metadata].join(". ");
 const upDisabled=index===0;
 const downDisabled=index===state.items.length-1;
 return `<li class="${currentClass}" data-queue-ref="${escapeHTML(ref)}" data-track-ref="${escapeHTML(ref)}" data-queue-index="${index}" draggable="true" aria-label="${escapeHTML(accessibleIdentity)}">
   <span class="queue-drag-handle" aria-hidden="true" title="Drag to reorder">⋮⋮</span>
   ${trackArtworkHTML(item)}
   <span class="queue-title">
     <span class="queue-title-line">
      ${marqueeTextHTML(titleText,{tag:"strong"})}
      <span class="queue-state">${escapeHTML(stateLabel)}</span>
     </span>
     <span class="queue-meta" title="${escapeHTML(metadata.join(" · "))}">${escapeHTML(metadata.join(" · "))}</span>
   </span>
   <span class="queue-actions">
     ${isCurrent?"":`<button data-queue-action="play" title="${escapeHTML(resolveText(locale,"playback.play"))}" aria-label="${escapeHTML(resolveText(locale,"playback.play"))}">▶</button>`}
     <details class="song-more queue-more">
      <summary aria-label="Track actions" title="Track actions">⋯</summary>
      <div class="song-more-menu">
       <button data-track-action="play-next" data-vex-action="vexstream.player.play-next">${escapeHTML(resolveText(locale,"player.play-next"))}</button>
       <button data-track-action="append" data-vex-action="vexstream.player.add-queue">${escapeHTML(resolveText(locale,"player.add-to-queue"))}</button>
       <button data-track-action="explore" data-vex-action="vexstream.explore.from-track">${escapeHTML(resolveText(locale,"explore.open"))}</button>
       <button data-track-action="edit">${escapeHTML(resolveText(locale,"library.edit"))}</button>
       <button data-track-action="trash">${escapeHTML(resolveText(locale,"trash.move"))}</button>
       ${trackPlaylistActionHTML()}
       <span class="queue-menu-separator" aria-hidden="true"></span>
       <button data-queue-action="up" ${upDisabled?"disabled":""}>${escapeHTML(resolveText(locale,"queue.move-up"))}</button>
       <button data-queue-action="down" ${downDisabled?"disabled":""}>${escapeHTML(resolveText(locale,"queue.move-down"))}</button>
       <button data-queue-action="remove" ${isCurrent?"disabled":""}>${escapeHTML(resolveText(locale,"queue.remove"))}</button>
      </div>
     </details>
   </span>
 </li>`;
}

function queueVirtualSpacerHTML(kind,count){
 if(count<=0)return"";
 return `<li class="queue-virtual-spacer queue-virtual-spacer-${kind}" data-queue-virtual="${kind}" style="height:${count*QUEUE_ROW_HEIGHT}px" aria-hidden="true"></li>`;
}

function renderQueueWindow(state,{force=false}={}){
 if(!queueDrawerOpen){
  if(queueList.childElementCount)queueList.innerHTML="";
  queueWindowRenderKey="";
  return;
 }
 if(state.kind==="empty"){
  queueList.innerHTML="";
  queueWindowStart=0;
  queueWindowInitialized=false;
  queueWindowRenderKey="empty";
  return;
 }

 const length=state.items.length;
 if(!queueWindowInitialized){
  const initial=queueWindowAroundIndex(
   length,
   state.currentIndex,
   QueueWindowPolicy
  );
  queueWindowStart=initial.start;
  queueWindowInitialized=true;
 }
 const maxStart=Math.max(0,length-QueueWindowPolicy.size);
 queueWindowStart=Math.max(0,Math.min(queueWindowStart,maxStart));
 const end=Math.min(length,queueWindowStart+QueueWindowPolicy.size);
 const visibleRefs=state.items.slice(queueWindowStart,end);
 const playlistSignature=playlists
  .map(playlist=>`${playlist.id}:${playlist.name}`)
  .join("|");
 const key=[
  catalogRevision,
  state.currentIndex,
  queueWindowStart,
  end,
  visibleRefs.join("\\u001f"),
  playlistSignature
 ].join("\\u001e");
 if(!force&&key===queueWindowRenderKey)return;

 queueList.dataset.windowStart=String(queueWindowStart);
 queueList.dataset.windowEnd=String(end);
 queueList.innerHTML=
  queueVirtualSpacerHTML("top",queueWindowStart)+
  visibleRefs.map(
   (ref,offset)=>queueTimelineRowHTML(
    state,
    ref,
    queueWindowStart+offset
   )
  ).join("")+
  queueVirtualSpacerHTML("bottom",length-end);
 queueWindowRenderKey=key;
 requestAnimationFrame(()=>refreshSongMarqueeMetrics(queueList));
}

function render(){
 const pp=player.playPausePresentation;
 const qp=queue.currentProjection;
 setControl(playPause,pp);
 setControl(next,qp.controls.next);
 setControl(shuffle,qp.controls.shuffle);
 setControl(repeat,qp.controls.repeat);
 setControl(clearAfterCurrent,qp.controls.clearAfterCurrent);
 setTransportIcon(playPause,pp,pp.iconRef==="pause"?"❚❚":"▶");
 setTransportIcon(next,qp.controls.next,"⏭");
 setTransportIcon(shuffle,qp.controls.shuffle,"🔀");
 setTransportIcon(repeat,qp.controls.repeat,"↻");

 const queueCount=qp.state.kind==="empty"?0:qp.state.items.length;
 const queueNextCount=qp.state.kind==="empty"
  ?0
  :Math.max(0,qp.state.items.length-qp.state.currentIndex-1);
 queueDrawer.classList.toggle("open",queueDrawerOpen);
 queueDrawer.setAttribute("aria-expanded",String(queueDrawerOpen));
 queueDrawerHandle.setAttribute("aria-expanded",String(queueDrawerOpen));
 queueMiniBtn.setAttribute("aria-expanded",String(queueDrawerOpen));
 queueMiniCount.textContent=String(queueCount);
 queueMiniBtn.title=queueDrawerOpen?"Close Queue":"Open Queue";
 queueDrawerCount.textContent=String(queueCount);
 queueDrawerNextCount.textContent=resolveText(locale,"queue.drawer.next",{
  count:queueNextCount
 });
 shuffleQueueUpcoming.disabled=queueNextCount<2;
 shuffleQueueUpcoming.title=resolveText(locale,"queue.shuffle-upcoming");

 const hasCurrent=Boolean(currentQueueRef(qp.state));
 const previousLabel=resolveText(locale,"queue.previous");
 previous.disabled=!hasCurrent;
 previous.textContent="⏮";
 previous.dataset.textRef="queue.previous";
 previous.setAttribute("aria-label",previousLabel);
 previous.title=previousLabel;

 semanticState.textContent=playback.currentProjection.state.kind;
 queueState.textContent=qp.state.kind;
 renderSessionState();
 status.textContent=playback.currentProjection.state.requestPending
  ? resolveText(locale,"playback.status.pending")
  : resolveText(locale,"playback.status.ready");

 const current=currentQueueRef(qp.state);
 if(!current){
  renderNowPlaying(null);
 }

 queueSummary.textContent=qp.state.kind==="empty"
  ? resolveText(locale,"queue.summary.empty")
  : resolveText(locale,"queue.summary.positioned",{
      count:qp.state.items.length,
      position:qp.state.currentIndex+1
    });
 renderQueueCurrentPane(qp.state);

 renderExplore();
 renderImportRuntimeStatus();
 renderImportReview();
 renderImportJobs();
 renderSystemStatus();
 syncActiveSongRows();

 renderQueueWindow(qp.state);
}

function trackPlaylistActionHTML(){
 return playlists.length
  ? `<select data-track-playlist-target>${playlists.map(playlist=>
      `<option value="${escapeHTML(playlist.id)}">${escapeHTML(playlist.name)}</option>`
     ).join("")}</select>
    <button data-track-action="playlist-add">${escapeHTML(resolveText(locale,"playlist.add-track"))}</button>`
  : "";
}

function trackCardHTML(item,{selectable=false}={}){
 const selected=selectable&&librarySelection.isSelected(item.ref);
 return `<article class="track-card ${selectable?"selectable":""} ${selected?"selected":""}" data-track-ref="${escapeHTML(item.ref)}">
      ${selectable
       ? `<label class="track-selection">
           <input type="checkbox" data-track-select ${selected?"checked":""}>
           <span class="sr-only">Select ${escapeHTML(trackTitle(item))}</span>
          </label>`
       : ""}
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
       ${trackPlaylistActionHTML()}
      </span>
     </article>`;
}

function formatLibraryAdded(value){
 const parsed=Date.parse(String(value??""));
 if(!Number.isFinite(parsed))return "—";
 return new Intl.DateTimeFormat("en-US",{
  month:"short",
  day:"numeric"
 }).format(new Date(parsed));
}

function trackArtworkHTML(item){
 const source=String(item?.thumbnailURL??"").trim();
 return `<span class="song-art${source?" has-art":""}">
   ${source
    ?`<img src="${escapeHTML(source)}" alt="">`
    :"<span>♫</span>"}
  </span>`;
}

function songRowHTML(item,{selectable=false}={}){
 const selected=selectable&&librarySelection.isSelected(item.ref);
 const playing=player.currentItem?.ref===item.ref;
 const rowIntent=selectable?"Select":"Play";
 const secondary=String(item?.sourceTitle??item?.channel??"").trim();
 return `<tr class="${playing?"playing ":""}${selected?"selected":""}" tabindex="0" aria-label="${rowIntent} ${escapeHTML(trackTitle(item))}" data-track-ref="${escapeHTML(item.ref)}">
   ${selectable
    ?`<td class="select-cell">
       <label class="song-select">
        <input type="checkbox" data-track-select ${selected?"checked":""}>
        <span class="sr-only">Select ${escapeHTML(trackTitle(item))}</span>
       </label>
      </td>`
    :""}
   <td class="song-title-cell">
    <div class="song-main">
     ${trackArtworkHTML(item)}
     <div class="song-copy">
      ${marqueeTextHTML(trackTitle(item),{tag:"strong"})}
      ${secondary?marqueeTextHTML(secondary,{tag:"small"}):""}
     </div>
    </div>
   </td>
   <td class="col-added"><span class="small">${escapeHTML(formatLibraryAdded(item.addedAt))}</span></td>
   <td class="col-artist">${marqueeTextHTML(trackArtist(item),{className:"clip2"})}</td>
   <td class="col-album">${marqueeTextHTML(String(item?.album??"").trim()||"Unknown album",{className:"clip2"})}</td>
   <td class="col-genre">${marqueeTextHTML(String(item?.genre??"").trim()||"—")}</td>
   <td class="col-duration">${escapeHTML(formatSeconds(item?.durationSeconds))}</td>
   <td class="col-actions">
    <div class="song-actions">
     <button data-track-action="play" data-vex-action="vexstream.player.play-track" title="${escapeHTML(resolveText(locale,"playback.play"))}" aria-label="${escapeHTML(resolveText(locale,"playback.play"))}">▶</button>
     <button data-track-action="append" data-vex-action="vexstream.player.add-queue" title="${escapeHTML(resolveText(locale,"player.add-to-queue"))}" aria-label="${escapeHTML(resolveText(locale,"player.add-to-queue"))}">☷＋</button>
     <details class="song-more">
      <summary aria-label="Track actions" title="Track actions">⋯</summary>
      <div class="song-more-menu">
       <button data-track-action="play-next" data-vex-action="vexstream.player.play-next">${escapeHTML(resolveText(locale,"player.play-next"))}</button>
       <button data-track-action="explore" data-vex-action="vexstream.explore.from-track">${escapeHTML(resolveText(locale,"explore.open"))}</button>
       <button data-track-action="edit">${escapeHTML(resolveText(locale,"library.edit"))}</button>
       <button data-track-action="trash">${escapeHTML(resolveText(locale,"trash.move"))}</button>
       ${trackPlaylistActionHTML()}
      </div>
     </details>
    </div>
   </td>
  </tr>`;
}

function libraryGroupArtworkHTML(group,{kind,hero=false}={}){
 const artworkTrack=group?.tracks?.find(track=>
  String(track?.thumbnailURL??"").trim()
 )??group?.tracks?.[0]??null;
 const source=String(artworkTrack?.thumbnailURL??"").trim();
 const fallback=kind==="artist"?"♫":group?.name??"♫";
 return `<span class="library-group-cover${hero?" library-group-cover-hero":""}${source?" has-art":""}">
   ${source
    ?`<img src="${escapeHTML(source)}" alt="">`
    :`<span>${escapeHTML(fallback)}</span>`}
  </span>`;
}

function libraryGroupCardMeta(group,kind){
 const related=kind==="album"
  ?[...new Set(group.tracks.map(track=>trackArtist(track)))]
  :[...new Set(group.tracks.map(track=>
    String(track?.album??"").trim()||"Unknown album"
   ))];
 const relationship=related.slice(0,2).join(", ");
 const trackCount=resolveText(locale,"library.group.tracks",{
  count:group.refs.length
 });
 return [relationship,trackCount].filter(Boolean).join(" · ");
}

function currentLibraryDetailGroup(){
 const detail=shellNavigation.currentProjection.libraryDetail;
 if(!detail)return null;
 return groupLibrary(catalog,detail.kind)
  .find(group=>group.key===detail.key)??null;
}

function currentLibraryDetailRelations(group,detail){
 if(!group||!detail)return Object.freeze([]);
 return groupLibrary(group.tracks,detail.kind==="album"?"artist":"album");
}

function resetArtistDetailExperience(){
 artistDetailIdentity="";
 artistAlbumsExpanded=false;
 artistSongQuery="";
 artistAlbumFilterKey="";
}

function artistDetailVisibleTracks(group){
 const projected=projectLibraryView(group?.tracks??[],{
  query:artistSongQuery,
  sortStack:libraryView.sortStack
 });
 return artistAlbumFilterKey
  ?projected.filter(track=>
    (String(track?.album??"").trim()||"Unknown album").toLocaleLowerCase()===artistAlbumFilterKey
   )
  :projected;
}

function artistSortSummaryHTML(){
 return `<span>Sort precedence:</span>
  ${libraryView.sortStack.map((spec,index)=>`
   <span class="sort-chip"><strong>${index+1}</strong> ${escapeHTML(sortLabel(spec.key))} ${escapeHTML(sortDirectionLabel(spec))}</span>
  `).join("")}
  ${librarySortPresetForStack(libraryView.sortStack)!=="title"
   ?'<button type="button" class="sort-reset" data-artist-sort-reset>Reset</button>'
   :""}`;
}

function renderArtistDetailTerrain(group,detail){
 const active=detail?.kind==="artist";
 artistDetailAlbums.hidden=!active;
 artistDetailSongs.hidden=!active;
 libraryGroupTracks.hidden=active;
 if(!active){
  resetArtistDetailExperience();
  artistAlbumShelf.innerHTML="";
  artistSongTable.innerHTML="";
  return;
 }
 if(artistDetailIdentity!==detail.key){
  artistDetailIdentity=detail.key;
  artistAlbumsExpanded=false;
  artistSongQuery="";
  artistAlbumFilterKey="";
 }
 const albums=groupLibrary(group.tracks,"album");
 const visible=artistDetailVisibleTracks(group);
 artistAlbumCount.textContent=String(albums.length);
 artistAlbumsToggle.setAttribute("aria-expanded",String(artistAlbumsExpanded));
 artistAlbumsToggle.firstElementChild.textContent=(artistAlbumsExpanded?"▾ ":"▸ ")+"Albums";
 artistAlbumsHint.textContent=artistAlbumsExpanded?"Hide album terrain":"Expand album terrain";
 artistAlbumShelf.hidden=!artistAlbumsExpanded;
 artistAlbumShelf.innerHTML=artistAlbumsExpanded
  ?albums.map(album=>`<button type="button" class="artist-album-card" data-artist-album-filter="${escapeHTML(album.key)}">
      ${libraryGroupArtworkHTML(album,{kind:"album"})}
      <span class="artist-album-copy">
       <strong>${escapeHTML(album.name)}</strong>
       <small>${escapeHTML(resolveText(locale,"library.group.tracks",{count:album.refs.length}))}</small>
      </span>
     </button>`).join("")
  :"";
 artistSongSearch.value=artistSongQuery;
 artistSongSummary.textContent=`${visible.length} of ${group.refs.length} song${group.refs.length===1?"":"s"} shown${artistSongQuery?` · Search: “${artistSongQuery}”`:""}`;
 const selectedAlbum=albums.find(album=>album.key===artistAlbumFilterKey);
 artistAlbumFilter.innerHTML=selectedAlbum
  ?`<span class="artist-filter-chip">Album: <strong>${escapeHTML(selectedAlbum.name)}</strong><button type="button" data-artist-album-filter-clear aria-label="Clear album filter">×</button></span>`
  :"";
 artistSortSummary.innerHTML=artistSortSummaryHTML();
 artistSongTable.innerHTML=visible.length
  ?`<div class="song-table-wrap">
     <table class="song-table artist-song-table">
      <thead><tr>
       ${songColumnHeader("title","Title","song-title-cell")}
       ${songColumnHeader("added","Added","col-added")}
       <th class="col-artist">Artist</th>
       ${songColumnHeader("album","Album","col-album")}
       ${songColumnHeader("genre","Genre","col-genre")}
       ${songColumnHeader("duration","Time","col-duration")}
       <th class="col-actions"></th>
      </tr></thead>
      <tbody>${visible.map(item=>songRowHTML(item)).join("")}</tbody>
     </table>
    </div>`
  :'<div class="empty-state">No matching songs for this artist.</div>';
}

function renderLibraryGroupDetail(){
 const detail=shellNavigation.currentProjection.libraryDetail;
 if(!detail){
  libraryGroupDetail.hidden=true;
  delete libraryGroupDetail.dataset.libraryGroupKind;
  libraryGroupArtwork.innerHTML="";
  libraryGroupRelations.innerHTML="";
  libraryGroupRelations.hidden=true;
  renderArtistDetailTerrain(null,null);
  libraryGroupTracks.hidden=false;
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
 libraryGroupDetail.dataset.libraryGroupKind=detail.kind;
 libraryGroupArtwork.innerHTML=libraryGroupArtworkHTML(group,{
  kind:detail.kind,
  hero:true
 });
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
 libraryGroupShuffle.textContent=resolveText(
  locale,
  isAlbum?"library.detail.shuffle.album":"library.detail.shuffle.artist"
 );
 libraryGroupPlayNext.textContent=resolveText(
  locale,
  isAlbum?"library.detail.play-next.album":"library.detail.play-next.artist"
 );
 libraryGroupAppend.textContent=resolveText(
  locale,
  isAlbum?"library.detail.queue.album":"library.detail.queue.artist"
 );
 const relations=currentLibraryDetailRelations(group,detail);
 libraryGroupRelations.hidden=relations.length===0;
 libraryGroupRelations.innerHTML=relations.length
  ?`<span class="library-detail-relation-label">${escapeHTML(resolveText(
     locale,
     isAlbum?"library.detail.relations.album":"library.detail.relations.artist"
    ))}:</span>
    ${relations.map(relation=>`<button type="button" class="library-detail-relation"
      data-library-relation-kind="${escapeHTML(isAlbum?"artist":"album")}"
      data-library-relation-key="${escapeHTML(relation.key)}">${escapeHTML(relation.name)}</button>`).join("")}`
  :"";
 renderArtistDetailTerrain(group,detail);
 libraryGroupTracks.innerHTML=isAlbum
  ?group.tracks.map(trackCardHTML).join("")
  :"";
}

function selectedLibraryRefs(){
 return librarySelection.currentProjection.selectedRefs.filter(ref=>byRef.has(ref));
}

function sharedDisplayValue(rows,key){
 const values=[...new Set(rows.map(item=>String(item?.[key]??"")))];
 return values.length===1?values[0]:"";
}

function clampSongColumnWidth(key,value){
 const rule=songColumnRules[key];
 if(!rule)return 0;
 return Math.round(Math.max(
  rule.minWidth,
  Math.min(rule.maxWidth,Number(value)||rule.defaultWidth)
 ));
}

function visibleSongColumnKeys({home=false}={}){
 const keys=["title","added","artist","album","genre","duration"];
 return keys.filter(key=>
  key==="title"||
  (key==="genre"&&!home&&songLayout.genre)||
  (key!=="genre"&&Boolean(songLayout[key]))
 );
}

function applySongLayoutProjection(){
 for(const key of Object.keys(songColumnRules)){
  const width=clampSongColumnWidth(key,songLayout.widths[key]);
  document.documentElement.style.setProperty("--song-col-"+key,width+"px");
 }
 const songsMin=visibleSongColumnKeys().reduce(
  (sum,key)=>sum+clampSongColumnWidth(key,songLayout.widths[key]),
  112
 );
 const homeMin=visibleSongColumnKeys({home:true}).reduce(
  (sum,key)=>sum+clampSongColumnWidth(key,songLayout.widths[key]),
  82
 );
 document.documentElement.style.setProperty("--song-table-min-width",songsMin+"px");
 document.documentElement.style.setProperty("--song-home-table-min-width",homeMin+"px");
 for(const key of ["artist","album","added","duration","genre"]){
  document.body.dataset["songField"+key[0].toUpperCase()+key.slice(1)]=String(Boolean(songLayout[key]));
 }
 for(const input of songLayoutFieldInputs){
  input.checked=Boolean(songLayout[input.dataset.songLayoutField]);
 }
 for(const handle of document.querySelectorAll("[data-song-resize]")){
  const key=handle.dataset.songResize;
  if(songColumnRules[key]){
   handle.setAttribute("aria-valuenow",String(clampSongColumnWidth(key,songLayout.widths[key])));
  }
 }
 requestAnimationFrame(()=>refreshSongMarqueeMetrics());
}

function setSongColumnWidth(key,value){
 if(!songColumnRules[key])return;
 songLayout=Object.freeze({
  ...songLayout,
  widths:Object.freeze({
   ...songLayout.widths,
   [key]:clampSongColumnWidth(key,value)
  })
 });
 applySongLayoutProjection();
}

let activeSongColumnResize=null;

function beginSongColumnResize(event){
 const handle=event.target.closest("[data-song-resize]");
 if(!handle)return false;
 const key=handle.dataset.songResize;
 const cell=handle.closest("th");
 if(!songColumnRules[key]||!cell)return false;
 activeSongColumnResize={
  key,
  startX:event.clientX,
  startWidth:cell.getBoundingClientRect().width
 };
 document.body.classList.add("song-column-resizing");
 try{handle.setPointerCapture?.(event.pointerId);}catch{}
 event.preventDefault();
 event.stopPropagation();
 return true;
}

function moveSongColumnResize(event){
 if(!activeSongColumnResize)return;
 const delta=event.clientX-activeSongColumnResize.startX;
 setSongColumnWidth(
  activeSongColumnResize.key,
  activeSongColumnResize.startWidth+delta
 );
}

function finishSongColumnResize(){
 if(!activeSongColumnResize)return;
 activeSongColumnResize=null;
 document.body.classList.remove("song-column-resizing");
 refreshSongMarqueeMetrics();
}

function resetSongColumnWidth(key){
 if(!songColumnRules[key])return;
 setSongColumnWidth(key,songColumnRules[key].defaultWidth);
}

function closeBulkDisplayEditor(){
 bulkDisplayRefs=Object.freeze([]);
 bulkDisplayEditor.hidden=true;
}

function openBulkDisplayEditor(){
 const refs=selectedLibraryRefs();
 if(!refs.length)return;
 if(refs.length===1){
  currentEditingTrackRef=refs[0];
  renderLibraryEditor();
  return;
 }
 const rows=refs.map(ref=>byRef.get(ref)).filter(Boolean);
 bulkDisplayRefs=Object.freeze(refs);
 bulkDisplayEditorTitle.textContent=`Edit shared display fields for ${refs.length} tracks`;
 bulkDisplayEditorSummary.textContent="Only checked fields change. Source MP3/provider metadata is not rewritten.";
 bulkApplyArtist.checked=false;
 bulkApplyAlbum.checked=false;
 bulkApplyGenre.checked=false;
 bulkArtist.value=sharedDisplayValue(rows,"artist");
 bulkAlbum.value=sharedDisplayValue(rows,"album");
 bulkGenre.value=sharedDisplayValue(rows,"genre");
 bulkDisplayEditor.hidden=false;
}

function renderTrackSelection(){
 const projection=librarySelection.currentProjection;
 const songs=libraryView.mode==="songs";
 playAllSongs.hidden=!songs;
 playAllSongs.disabled=!songs||currentLibraryView().length===0;
 selectSongs.hidden=!songs;
 selectSongs.textContent=resolveText(
  locale,
  projection.active?"library.selection.done":"library.selection.select"
 );
 librarySelectionBar.hidden=!(songs&&projection.active);
 librarySelectionCount.textContent=resolveText(locale,"library.selection.count",{
  count:projection.count
 });
 const disabled=projection.count===0;
 for(const button of [
  bulkPlaySelected,
  bulkPlayNextSelected,
  bulkQueueSelected,
  bulkEditSelected,
  bulkTrashSelected
 ]){
  button.disabled=disabled;
 }
 const prior=bulkPlaylistTarget.value;
 bulkPlaylistTarget.innerHTML=playlists.length
  ?playlists.map(playlist=>
    `<option value="${escapeHTML(playlist.id)}">${escapeHTML(playlist.name)}</option>`
   ).join("")
  :`<option value="">${escapeHTML(resolveText(locale,"playlist.empty"))}</option>`;
 if(playlists.some(playlist=>playlist.id===prior)){
  bulkPlaylistTarget.value=prior;
 }
 bulkPlaylistSelected.disabled=disabled||playlists.length===0;
 bulkEditSelected.textContent=projection.count>1?"Edit shared fields…":"Edit display…";
 if(projection.count<2&&!bulkDisplayEditor.hidden)closeBulkDisplayEditor();
}

function sortLabel(key){
 return {title:"Title",added:"Added",artist:"Artist",album:"Album",genre:"Genre",duration:"Time"}[key]??key;
}

function sortDirectionLabel(spec){
 if(spec.key==="added")return spec.dir==="desc"?"Newest":"Oldest";
 if(spec.key==="duration")return spec.dir==="desc"?"Longest":"Shortest";
 return spec.dir==="desc"?"Z→A":"A→Z";
}

function songSortHeader(key,label){
 const index=libraryView.sortStack.findIndex(spec=>spec.key===key);
 const active=index>=0;
 const spec=active?libraryView.sortStack[index]:null;
 return `<button type="button" class="song-sort-head" data-song-sort="${escapeHTML(key)}">
   <span>${escapeHTML(label)}</span>
   ${active?`<span class="song-sort-arrow">${spec.dir==="asc"?"↑":"↓"}</span><span class="song-sort-rank">${index+1}</span>`:""}
  </button>`;
}

function songColumnHeader(key,label,className,{sortable=true}={}){
 const head=sortable
  ?songSortHeader(key,label)
  :`<span class="song-static-head">${escapeHTML(label)}</span>`;
 const rule=songColumnRules[key];
 const width=clampSongColumnWidth(key,songLayout.widths[key]);
 return `<th class="${className}" data-song-column="${escapeHTML(key)}">
   ${head}
   <span class="song-column-resizer" data-song-resize="${escapeHTML(key)}" role="separator" tabindex="0" aria-orientation="vertical" aria-valuemin="${rule.minWidth}" aria-valuemax="${rule.maxWidth}" aria-valuenow="${width}" aria-label="Resize ${escapeHTML(label)} column" title="Drag to resize · Arrow keys resize · Double-click to reset"></span>
  </th>`;
}

function renderSongSortSummary(){
 const songs=libraryView.mode==="songs";
 songSortSummary.hidden=!songs;
 if(!songs){
  songSortSummary.innerHTML="";
  return;
 }
 songSortSummary.innerHTML=`<span>Sort precedence:</span>
  ${libraryView.sortStack.map((spec,index)=>`
   <span class="sort-chip"><strong>${index+1}</strong> ${escapeHTML(sortLabel(spec.key))} ${escapeHTML(sortDirectionLabel(spec))}</span>
  `).join("")}
  ${librarySortPresetForStack(libraryView.sortStack)!=="title"
   ?'<button type="button" class="sort-reset" data-song-sort-reset>Reset</button>'
   :""}`;
}

function renderTrackChoices(){
 applySongLayoutProjection();
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
 librarySearchClear.hidden=!libraryView.query.trim();
 librarySort.value=librarySortPresetForStack(libraryView.sortStack);
 renderSongSortSummary();
 for(const button of libraryModeButtons){
  const selected=button.dataset.libraryMode===libraryView.mode;
  button.setAttribute("aria-pressed",String(selected));
 }
 trackChoices.dataset.libraryTerrain=libraryView.mode;

 if(libraryView.mode==="songs"){
  librarySummary.textContent=resolveText(locale,"library.summary.songs",{
   visible:visible.length,
   total:catalog.length
  });
  const selectable=librarySelection.currentProjection.active;
  trackChoices.innerHTML=visible.length
   ?`<div class="song-table-wrap">
      <table class="song-table">
       <thead>
        <tr>
         ${selectable?'<th class="select-cell">✓</th>':""}
         ${songColumnHeader("title","Title","song-title-cell")}
         ${songColumnHeader("added","Added","col-added")}
         ${songColumnHeader("artist","Artist","col-artist")}
         ${songColumnHeader("album","Album","col-album")}
         ${songColumnHeader("genre","Genre","col-genre")}
         ${songColumnHeader("duration","Time","col-duration")}
         <th class="col-actions"></th>
        </tr>
       </thead>
       <tbody>
        ${visible.map(item=>songRowHTML(item,{selectable})).join("")}
       </tbody>
      </table>
     </div>`
   :`<div class="empty-state">${escapeHTML(resolveText(locale,"library.empty.filtered"))}</div>`;
  renderTrackSelection();
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
      ${libraryGroupArtworkHTML(group,{kind:groupKey})}
      <div class="library-group-copy">
       <strong>${escapeHTML(group.name)}</strong>
       <small>${escapeHTML(libraryGroupCardMeta(group,groupKey))}</small>
      </div>
      <div class="library-group-actions">
       <button data-library-group-action="play">▶ ${escapeHTML(resolveText(locale,"playback.play"))}</button>
       <button data-library-group-action="append">☷＋ ${escapeHTML(resolveText(locale,"player.add-to-queue"))}</button>
       <button data-library-group-action="open">${escapeHTML(resolveText(locale,"library.group.open"))}</button>
      </div>
     </article>`).join("")
  :`<div class="empty-state">${escapeHTML(resolveText(locale,"library.empty.filtered"))}</div>`;
 renderTrackSelection();
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

function currentPlaylistDetail(){
 const id=playlistNavigation.currentProjection.openPlaylistID;
 if(!id)return null;
 return resolvedPlaylists.find(value=>value.playlist.id===id)??null;
}

function playlistUnavailableWitness(ref){
 const matches=item=>
  String(item?.track?.ref??item?.track?.id??"")===ref;
 const trashed=trashSnapshot.trash.find(matches);
 if(trashed?.track){
  return Object.freeze({
   track:trashed.track,
   status:resolveText(locale,"playlist.unavailable.trash")
  });
 }
 const deleted=trashSnapshot.deletedHistory.find(matches);
 if(deleted?.track){
  return Object.freeze({
   track:deleted.track,
   status:resolveText(locale,"playlist.unavailable.deleted")
  });
 }
 return Object.freeze({
  track:null,
  status:resolveText(locale,"playlist.unavailable.missing")
 });
}

function playlistEntryHTML(entry,index,total){
 const witness=entry.available
  ?Object.freeze({track:entry.track,status:""})
  :playlistUnavailableWitness(entry.ref);
 const item=witness.track;
 const label=item
  ?trackTitle(item)
  :resolveText(locale,"playlist.missing-track");
 const metadata=entry.available
  ?[trackArtist(item),String(item?.album??"").trim()].filter(Boolean).join(" · ")
  :[item?trackArtist(item):entry.ref,witness.status].filter(Boolean).join(" · ");
 return `<li class="playlist-entry ${entry.available?"":"missing"}"
      data-playlist-track-ref="${escapeHTML(entry.ref)}"
      data-track-ref="${escapeHTML(entry.ref)}"
      data-playlist-index="${index}"
      data-available="${String(entry.available)}">
   <span class="playlist-entry-order">${index+1}</span>
   ${item
    ?trackArtworkHTML(item)
    :'<span class="song-art"><span>♫</span></span>'}
   <span class="playlist-entry-copy">
    <strong>${escapeHTML(label)}</strong>
    <small>${escapeHTML(metadata)}</small>
   </span>
   <span class="playlist-entry-actions">
    ${entry.available
     ?`<span class="playlist-track-actions">
       <button data-track-action="play" data-vex-action="vexstream.player.play-track" title="${escapeHTML(resolveText(locale,"playback.play"))}" aria-label="${escapeHTML(resolveText(locale,"playback.play"))}">▶</button>
       <button data-track-action="append" data-vex-action="vexstream.player.add-queue" title="${escapeHTML(resolveText(locale,"player.add-to-queue"))}" aria-label="${escapeHTML(resolveText(locale,"player.add-to-queue"))}">☷＋</button>
       <details class="song-more">
        <summary aria-label="Track actions" title="Track actions">⋯</summary>
        <div class="song-more-menu">
         <button data-track-action="play-next" data-vex-action="vexstream.player.play-next">${escapeHTML(resolveText(locale,"player.play-next"))}</button>
         <button data-track-action="explore" data-vex-action="vexstream.explore.from-track">${escapeHTML(resolveText(locale,"explore.open"))}</button>
         <button data-track-action="edit">${escapeHTML(resolveText(locale,"library.edit"))}</button>
         <button data-track-action="trash">${escapeHTML(resolveText(locale,"trash.move"))}</button>
         ${trackPlaylistActionHTML()}
        </div>
       </details>
      </span>`
     :""}
    <button data-playlist-entry-action="up" ${index===0?"disabled":""}>↑</button>
    <button data-playlist-entry-action="down" ${index===total-1?"disabled":""}>↓</button>
    <button data-playlist-entry-action="remove">${escapeHTML(resolveText(locale,"playlist.remove-track"))}</button>
   </span>
  </li>`;
}

function renderPlaylistChoices(){
 playlistNavCount.textContent=playlists.length?`(${playlists.length})`:"";
 const openID=playlistNavigation.currentProjection.openPlaylistID;
 playlistOverview.hidden=Boolean(openID);
 playlistDetail.hidden=!openID;

 playlistChoices.innerHTML=resolvedPlaylists.length
  ?resolvedPlaylists.map(resolved=>{
   const playlist=resolved.playlist;
   const availability=resolveText(locale,"playlist.availability",{
    available:resolved.playableRefs.length,
    missing:resolved.missingRefs.length
   });
   return `<article class="playlist-card" data-playlist-id="${escapeHTML(playlist.id)}">
    <div class="playlist-heading">
     <strong>${escapeHTML(playlist.name)}</strong>
     <small>${escapeHTML(availability)}</small>
    </div>
    <div class="playlist-card-actions">
     <button data-playlist-action="play" ${resolved.playableRefs.length?"":"disabled"}>${escapeHTML(resolveText(locale,"playlist.play"))}</button>
     <button data-playlist-action="shuffle" ${resolved.playableRefs.length?"":"disabled"}>${escapeHTML(resolveText(locale,"playlist.shuffle"))}</button>
     <button data-playlist-action="open">${escapeHTML(resolveText(locale,"playlist.open"))}</button>
    </div>
   </article>`;
  }).join("")
  :`<div class="empty-state">${escapeHTML(resolveText(locale,"playlist.empty"))}</div>`;

 if(!openID)return;
 const resolved=currentPlaylistDetail();
 if(!resolved){
  playlistNavigation.close();
  return;
 }
 const playlist=resolved.playlist;
 playlistDetailName.value=playlist.name;
 playlistDetailAvailability.textContent=resolveText(locale,"playlist.availability",{
  available:resolved.playableRefs.length,
  missing:resolved.missingRefs.length
 });
 playlistDetailPlay.disabled=resolved.playableRefs.length===0;
 playlistDetailEntries.innerHTML=resolved.entries.length
  ?resolved.entries.map((entry,index)=>
    playlistEntryHTML(entry,index,resolved.entries.length)
   ).join("")
  :`<li class="empty-state">${escapeHTML(resolveText(locale,"playlist.empty"))}</li>`;
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

function renderPhoneConnection(projection){
 const view=presentPhoneConnection(projection);
 if(!view){
  modalBackdrop.hidden=true;
  modalBody.innerHTML="";
  modalActions.innerHTML="";
  phoneConnectButton.setAttribute("aria-expanded","false");
  return;
 }
 modalTitle.textContent=view.title;
 modalBody.innerHTML=view.bodyHTML;
 modalActions.innerHTML=view.actionsHTML;
 modalBackdrop.hidden=false;
 phoneConnectButton.setAttribute("aria-expanded","true");
}

phoneConnection.subscribe(renderPhoneConnection);

phoneConnectButton.addEventListener(
 "click",
 ()=>void guarded(()=>phoneConnection.open())
);
modalCloseBtn.addEventListener("click",()=>phoneConnection.close());
modalBackdrop.addEventListener("click",event=>{
 if(event.target===modalBackdrop)phoneConnection.close();
});
modalActions.addEventListener("click",event=>{
 if(event.target.closest("[data-modal-close]"))phoneConnection.close();
});
modalBody.addEventListener("click",event=>void guarded(async()=>{
 const target=event.target.closest("[data-phone-action]");
 if(!target)return;
 await phoneConnection.activate({
  kind:target.dataset.phoneAction,
  requestRef:target.dataset.phoneRequestRef,
  deviceRef:target.dataset.phoneDeviceRef,
 });
}));

systemStatusButton.addEventListener("click",()=>{
 renderSystemStatus();
 systemDrawer.hidden=false;
 systemStatusButton.setAttribute("aria-expanded","true");
});

systemDrawerClose.addEventListener("click",()=>{
 systemDrawer.hidden=true;
 systemStatusButton.setAttribute("aria-expanded","false");
});

systemAttention.addEventListener("click",event=>{
 if(!event.target.closest('[data-system-action="reset-listening-session"]'))return;
 void guarded(async()=>{
  const state=queue.currentProjection.state;
  await savePlayerSession(state);
  const adoptedRef=currentQueueRef(state)??"";
  currentSessionRestoration=Object.freeze({
   kind:state.kind==="empty"?"empty":"restored",
   queue:null,
   droppedRefs:Object.freeze([]),
   currentRef:adoptedRef
  });
  playerSessionPersistenceEnabled=true;
  render();
 });
});

reviewOldSessions.addEventListener("click",()=>void guarded(async()=>{
 reviewOldSessions.disabled=true;
 try{
  renderOldSessionPlan(await loadSessionRetirePlan());
 }finally{
  reviewOldSessions.disabled=false;
 }
}));

oldSessionPlan.addEventListener("change",event=>{
 if(!event.target.closest("[data-old-session-pid]"))return;
 updateOldSessionRetireButton();
});

oldSessionPlan.addEventListener("click",event=>{
 if(!event.target.closest("#retireOldSessions"))return;
 const pids=[...oldSessionPlan.querySelectorAll("[data-old-session-pid]:checked")]
  .map(input=>Number(input.dataset.oldSessionPid))
  .filter(Number.isInteger);
 if(!pids.length)return;
 if(!globalThis.confirm(resolveText(locale,"system.sessions.confirm")))return;
 void guarded(async()=>{
  await retireSessions(pids);
  renderOldSessionPlan(await loadSessionRetirePlan());
 });
});

exitVexStreamButton.addEventListener("click",()=>void guarded(async()=>{
 exitVexStreamButton.disabled=true;
 exitVexStreamButton.textContent=resolveText(locale,"system.exit.exiting");
 try{
  await flushPlayerSessionSave();
  await exitVexStream();
  buildIdentity.textContent=resolveText(locale,"system.exit.complete");
  status.textContent=resolveText(locale,"system.exit.complete");
 }catch(failure){
  exitVexStreamButton.disabled=false;
  exitVexStreamButton.textContent=resolveText(locale,"system.exit");
  throw failure;
 }
}));

playPause.addEventListener("click",()=>void guarded(()=>player.activatePlayPause()));
next.addEventListener("click",()=>void guarded(()=>player.next()));
previous.addEventListener("click",()=>void guarded(()=>player.previous()));
shuffle.addEventListener("click",()=>{player.toggleShuffle();render();});
repeat.addEventListener("click",()=>{player.toggleRepeat();render();});
seek.addEventListener("input",()=>{
 timeline.seekNormalized(Number(seek.value)/1000);
});
volumeInput.addEventListener("input",()=>{
 playbackVolume.setVolume(Number(volumeInput.value));
});
queueDrawerHandle.addEventListener("click",toggleQueueDrawer);
queueMiniBtn.addEventListener("click",toggleQueueDrawer);

clearAfterCurrent.addEventListener("click",()=>{player.clearAfterCurrent();render();});

librarySearch.addEventListener("input",()=>{
 libraryView=Object.freeze({...libraryView,query:librarySearch.value});
 librarySearchClear.hidden=!libraryView.query.trim();
 if(shellNavigation.currentProjection.libraryMode)renderTrackChoices();
});
function executeLibrarySearch(){
 shellNavigation.select(ShellView.SONGS);
 renderTrackChoices();
 librarySearch.focus();
}
librarySearchExecute.addEventListener("click",executeLibrarySearch);
librarySearch.addEventListener("keydown",event=>{
 if(event.key!=="Enter")return;
 event.preventDefault();
 executeLibrarySearch();
});
librarySearchClear.addEventListener("click",()=>{
 libraryView=Object.freeze({...libraryView,query:""});
 renderTrackChoices();
 librarySearch.focus();
});
libraryGenre.addEventListener("change",()=>{
 libraryView=Object.freeze({...libraryView,genre:libraryGenre.value});
 if(shellNavigation.currentProjection.view===ShellView.HOME){
  shellNavigation.select(ShellView.SONGS);
 }
 if(shellNavigation.currentProjection.libraryMode){
  renderTrackChoices();
 }
});

librarySort.addEventListener("change",()=>{
 if(librarySort.value==="custom")return;
 libraryView=Object.freeze({
  ...libraryView,
  sort:librarySort.value,
  sortStack:librarySortStackForPreset(librarySort.value)
 });
 renderTrackChoices();
});

songSortSummary.addEventListener("click",event=>{
 if(!event.target.closest("[data-song-sort-reset]"))return;
 libraryView=Object.freeze({
  ...libraryView,
  sort:"title",
  sortStack:resetLibrarySortStack()
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

for(const button of discoverModeButtons){
 button.addEventListener("click",()=>{
  discoverNavigation.select(button.dataset.discoverMode);
 });
}

youtubeInspect.addEventListener("click",()=>void guarded(async()=>{
 await loadImportReview(youtubeImportURL.value);
}));

youtubeImportURL.addEventListener("keydown",event=>{
 if(event.key!=="Enter")return;
 event.preventDefault();
 youtubeInspect.click();
});

importRuntimeNotice.addEventListener("click",event=>{
 const action=event.target.closest("[data-import-runtime-action=\"setup\"]");
 if(!action)return;
 void guarded(()=>prepareImportRuntime());
});

libraryNavToggle.addEventListener("click",()=>{shellNavigation.toggleSidebar();});
goDiscoverAdd.addEventListener("click",()=>{
 shellNavigation.select(ShellView.DISCOVER);
 discoverNavigation.select(DiscoverMode.PC);
});
rescanLibraryBtn.addEventListener("click",()=>void guarded(async()=>{
 await rescanLibrary();
 await refreshCatalog();
}));
shuffleAllBtn.addEventListener("click",()=>void guarded(async()=>{
 const shuffled=shuffleRefs([...refs]);
 if(shuffled.length)await player.playContext(shuffled,0);
}));

nowPlaying.addEventListener("click",event=>{
 const relation=event.target.closest("[data-now-library-kind][data-now-library-key]");
 if(relation){
  shellNavigation.openLibraryDetail(
   relation.dataset.nowLibraryKind,
   relation.dataset.nowLibraryKey
  );
  return;
 }
 if(event.target.closest("#nowArtButton,#nowTitleButton")){
  if(!nowDetails.hidden)nowDetails.open=!nowDetails.open;
  return;
 }
 if(handleTrackAction(event,{playbackRefs:libraryPlaybackRefs()})){
  nowDetails.open=false;
 }
});

function rowTargetOwnsInteraction(target){
 return Boolean(target.closest(
  "button,input,select,option,summary,details,label,a,[role='button']"
 ));
}

function activateTrackRow(
 event,
 {row,ref,playbackRefs=libraryPlaybackRefs(),selectionAware=false}={}
){
 if(!row||!ref||rowTargetOwnsInteraction(event.target))return false;
 const selecting=selectionAware&&librarySelection.currentProjection.active;

 if(event.type==="click"){
  if(!selecting)return false;
  if(Number(event.detail)>1)return true;
  librarySelection.toggle(ref,!librarySelection.isSelected(ref));
  return true;
 }

 if(event.type==="dblclick"){
  if(selecting)return true;
  event.preventDefault();
 }else if(event.type==="keydown"){
  if(event.key!=="Enter"&&event.key!==" ")return false;
  event.preventDefault();
 }else{
  return false;
 }

 if(selecting){
  librarySelection.toggle(ref,!librarySelection.isSelected(ref));
  return true;
 }
 void guarded(()=>player.playNow(ref,playbackRefs));
 return true;
}

recentSongs.addEventListener("click",event=>{
 if(handleTrackAction(event,{playbackRefs:libraryPlaybackRefs()}))return;
 const row=event.target.closest("[data-home-track-ref]");
 if(!row)return;
 activateTrackRow(event,{
  row,
  ref:row.dataset.homeTrackRef,
  playbackRefs:libraryPlaybackRefs()
 });
});
recentSongs.addEventListener("dblclick",event=>{
 const row=event.target.closest("[data-home-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.homeTrackRef,
  playbackRefs:libraryPlaybackRefs()
 });
});
recentSongs.addEventListener("keydown",event=>{
 const row=event.target.closest("[data-home-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.homeTrackRef,
  playbackRefs:libraryPlaybackRefs()
 });
});

function handleTrackAction(event,{playbackRefs=libraryPlaybackRefs()}={}){
 const action=event.target.closest("[data-track-action]");
 const card=event.target.closest("[data-track-ref]");
 if(!action||!card)return false;
 const ref=card.dataset.trackRef;
 if(action.dataset.trackAction==="play"){
  void guarded(()=>player.playNow(ref,playbackRefs));
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
   shellNavigation.select(ShellView.DISCOVER);
   discoverNavigation.select(DiscoverMode.EXPLORE);
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

playAllSongs.addEventListener("click",()=>void guarded(async()=>{
 const visible=currentLibraryView();
 if(visible.length)await player.playContext(
  visible.map(item=>item.ref),
  0
 );
}));

songLayoutButton.addEventListener("click",()=>{songLayoutPanel.hidden=false;});
songLayoutDone.addEventListener("click",()=>{songLayoutPanel.hidden=true;});
songLayoutReset.addEventListener("click",()=>{
 songLayout=freshDefaultSongLayout();
 applySongLayoutProjection();
});
for(const input of songLayoutFieldInputs){
 input.addEventListener("change",()=>{
  const key=input.dataset.songLayoutField;
  songLayout=Object.freeze({...songLayout,[key]:input.checked});
  applySongLayoutProjection();
 });
}
for(const host of [recentSongs,trackChoices,artistSongTable]){
 host.addEventListener("pointerdown",beginSongColumnResize);
}
document.addEventListener("keydown",event=>{
 const handle=event.target.closest("[data-song-resize]");
 if(!handle)return;
 const key=handle.dataset.songResize;
 const rule=songColumnRules[key];
 if(!rule)return;
 let next=null;
 const step=event.shiftKey?32:8;
 if(event.key==="ArrowLeft")next=clampSongColumnWidth(key,songLayout.widths[key]-step);
 if(event.key==="ArrowRight")next=clampSongColumnWidth(key,songLayout.widths[key]+step);
 if(event.key==="Home")next=rule.minWidth;
 if(event.key==="End")next=rule.maxWidth;
 if(next==null)return;
 event.preventDefault();
 event.stopPropagation();
 setSongColumnWidth(key,next);
});
document.addEventListener("pointermove",moveSongColumnResize);
document.addEventListener("pointerup",finishSongColumnResize);
document.addEventListener("pointercancel",finishSongColumnResize);
document.addEventListener("dblclick",event=>{
 const handle=event.target.closest("[data-song-resize]");
 if(!handle)return;
 event.preventDefault();
 event.stopPropagation();
 resetSongColumnWidth(handle.dataset.songResize);
});

selectSongs.addEventListener("click",()=>{
 const wasActive=librarySelection.currentProjection.active;
 librarySelection.toggleMode();
 if(wasActive&&!librarySelection.currentProjection.active){
  playlistSelectionTargetID="";
 }
});

selectVisibleSongs.addEventListener("click",()=>{
 librarySelection.selectVisible(
  currentLibraryView().map(item=>item.ref)
 );
});

clearSongSelection.addEventListener("click",()=>{
 librarySelection.clear();
});

trackChoices.addEventListener("change",event=>{
 const input=event.target.closest("[data-track-select]");
 const card=event.target.closest("[data-track-ref]");
 if(!input||!card)return;
 librarySelection.toggle(card.dataset.trackRef,input.checked);
});

bulkPlaySelected.addEventListener("click",()=>void guarded(async()=>{
 const selected=selectedLibraryRefs();
 if(selected.length)await player.playContext(selected,0);
}));

bulkPlayNextSelected.addEventListener("click",()=>void guarded(async()=>{
 const selected=selectedLibraryRefs();
 if(selected.length)await player.playNext(selected);
}));

bulkQueueSelected.addEventListener("click",()=>{
 const selected=selectedLibraryRefs();
 if(!selected.length)return;
 player.append(selected);
 render();
});

bulkEditSelected.addEventListener("click",openBulkDisplayEditor);
bulkDisplayEditorClose.addEventListener("click",closeBulkDisplayEditor);
cancelBulkDisplay.addEventListener("click",closeBulkDisplayEditor);
applyBulkDisplay.addEventListener("click",()=>void guarded(async()=>{
 const ids=bulkDisplayRefs.filter(ref=>byRef.has(ref));
 if(ids.length<2){closeBulkDisplayEditor();return;}
 const patch={};
 if(bulkApplyArtist.checked)patch.artist=bulkArtist.value;
 if(bulkApplyAlbum.checked)patch.album=bulkAlbum.value;
 if(bulkApplyGenre.checked)patch.genre=bulkGenre.value;
 if(!Object.keys(patch).length)throw new Error("Choose Artist, Album / collection, or Genre before applying the bulk edit");
 await updateTrackDisplays(ids,patch);
 await refreshCatalog();
 closeBulkDisplayEditor();
}));

bulkPlaylistSelected.addEventListener("click",()=>void guarded(async()=>{
 const selected=selectedLibraryRefs();
 const playlistID=bulkPlaylistTarget.value;
 if(!selected.length||!playlistID)return;
 const returnToPlaylist=playlistSelectionTargetID===playlistID
  ?playlistID
  :"";
 await addTracksToPlaylist(playlistID,selected);
 await refreshPlaylists();
 if(returnToPlaylist){
  playlistSelectionTargetID="";
  librarySelection.deactivate();
  shellNavigation.select(ShellView.PLAYLISTS);
  playlistNavigation.open(returnToPlaylist);
 }
}));

bulkTrashSelected.addEventListener("click",()=>void guarded(async()=>{
 const selected=selectedLibraryRefs();
 if(!selected.length)return;
 await moveTracksToTrash(selected);
 librarySelection.clear();
 await Promise.all([
  refreshCatalog(),
  refreshTrash()
 ]);
}));

trackChoices.addEventListener("click",event=>{
 const sortAction=event.target.closest("[data-song-sort]");
 if(sortAction){
  const next=activateLibrarySortStack(
   libraryView.sortStack,
   sortAction.dataset.songSort
  );
  libraryView=Object.freeze({
   ...libraryView,
   sort:librarySortPresetForStack(next),
   sortStack:next
  });
  renderTrackChoices();
  return;
 }

 const groupAction=event.target.closest("[data-library-group-action]");
 const groupCard=event.target.closest("[data-library-group]");
 if(groupAction&&groupCard){
  const groupKind=groupCard.dataset.libraryGroupKind;
  const groupKey=groupCard.dataset.libraryGroup;
  if(groupAction.dataset.libraryGroupAction==="open"){
   shellNavigation.openLibraryDetail(groupKind,groupKey);
   return;
  }

  const groups=groupLibrary(
   currentLibraryView(),
   groupKind
  );
  const group=groups.find(value=>value.key===groupKey);
  if(!group)return;
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
 if(handleTrackAction(event))return;
 const row=event.target.closest("[data-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.trackRef,
  playbackRefs:libraryPlaybackRefs(),
  selectionAware:true
 });
});
trackChoices.addEventListener("dblclick",event=>{
 const row=event.target.closest("[data-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.trackRef,
  playbackRefs:libraryPlaybackRefs(),
  selectionAware:true
 });
});
trackChoices.addEventListener("keydown",event=>{
 const row=event.target.closest("[data-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.trackRef,
  playbackRefs:libraryPlaybackRefs(),
  selectionAware:true
 });
});

libraryGroupTracks.addEventListener("click",event=>{
 const group=currentLibraryDetailGroup();
 if(handleTrackAction(event,{
  playbackRefs:group?.refs??libraryPlaybackRefs()
 }))return;
 const row=event.target.closest("[data-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.trackRef,
  playbackRefs:group?.refs??libraryPlaybackRefs()
 });
});
libraryGroupTracks.addEventListener("dblclick",event=>{
 const group=currentLibraryDetailGroup();
 const row=event.target.closest("[data-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.trackRef,
  playbackRefs:group?.refs??libraryPlaybackRefs()
 });
});
libraryGroupTracks.addEventListener("keydown",event=>{
 const group=currentLibraryDetailGroup();
 const row=event.target.closest("[data-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.trackRef,
  playbackRefs:group?.refs??libraryPlaybackRefs()
 });
});

artistAlbumsToggle.addEventListener("click",()=>{
 artistAlbumsExpanded=!artistAlbumsExpanded;
 renderLibraryGroupDetail();
});

artistAlbumShelf.addEventListener("click",event=>{
 const card=event.target.closest("[data-artist-album-filter]");
 if(!card)return;
 artistAlbumFilterKey=card.dataset.artistAlbumFilter;
 artistSongQuery="";
 renderLibraryGroupDetail();
});

artistSongSearch.addEventListener("input",()=>{
 artistSongQuery=artistSongSearch.value;
 renderLibraryGroupDetail();
});

artistAlbumFilter.addEventListener("click",event=>{
 if(!event.target.closest("[data-artist-album-filter-clear]"))return;
 artistAlbumFilterKey="";
 renderLibraryGroupDetail();
});

artistSortSummary.addEventListener("click",event=>{
 if(!event.target.closest("[data-artist-sort-reset]"))return;
 libraryView=Object.freeze({
  ...libraryView,
  sort:"title",
  sortStack:resetLibrarySortStack()
 });
 renderLibraryGroupDetail();
});

artistSongTable.addEventListener("click",event=>{
 const sortAction=event.target.closest("[data-song-sort]");
 if(sortAction){
  const next=activateLibrarySortStack(
   libraryView.sortStack,
   sortAction.dataset.songSort
  );
  libraryView=Object.freeze({
   ...libraryView,
   sort:librarySortPresetForStack(next),
   sortStack:next
  });
  renderLibraryGroupDetail();
  return;
 }
 const group=currentLibraryDetailGroup();
 const visible=group?artistDetailVisibleTracks(group):[];
 if(handleTrackAction(event,{
  playbackRefs:visible.map(item=>item.ref)
 }))return;
 const row=event.target.closest("[data-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.trackRef,
  playbackRefs:visible.map(item=>item.ref)
 });
});
artistSongTable.addEventListener("dblclick",event=>{
 const group=currentLibraryDetailGroup();
 const visible=group?artistDetailVisibleTracks(group):[];
 const row=event.target.closest("[data-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.trackRef,
  playbackRefs:visible.map(item=>item.ref)
 });
});
artistSongTable.addEventListener("keydown",event=>{
 const group=currentLibraryDetailGroup();
 const visible=group?artistDetailVisibleTracks(group):[];
 const row=event.target.closest("[data-track-ref]");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.trackRef,
  playbackRefs:visible.map(item=>item.ref)
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

libraryGroupShuffle.addEventListener("click",()=>{
 const group=currentLibraryDetailGroup();
 if(!group)return;
 const shuffled=shuffleRefs([...group.refs]);
 if(shuffled.length)void guarded(()=>player.playContext(shuffled,0));
});

libraryGroupPlayNext.addEventListener("click",()=>{
 const group=currentLibraryDetailGroup();
 if(!group)return;
 void guarded(()=>player.playNext(group.refs));
});

libraryGroupRelations.addEventListener("click",event=>{
 const relation=event.target.closest("[data-library-relation-kind][data-library-relation-key]");
 if(!relation)return;
 shellNavigation.openLibraryDetail(
  relation.dataset.libraryRelationKind,
  relation.dataset.libraryRelationKey
 );
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
 if(!action)return;

 if(action.dataset.playlistAction==="play"){
  if(resolved.playableRefs.length){
   void guarded(()=>player.playContext(resolved.playableRefs,0));
  }
  return;
 }
 if(action.dataset.playlistAction==="shuffle"){
  const shuffled=shuffleRefs([...resolved.playableRefs]);
  if(shuffled.length){
   void guarded(()=>player.playContext(shuffled,0));
  }
  return;
 }
 if(action.dataset.playlistAction==="open"){
  playlistNavigation.open(id);
 }
});

playlistDetailBack.addEventListener("click",()=>{
 playlistNavigation.close();
});

playlistDetailPlay.addEventListener("click",()=>{
 const resolved=currentPlaylistDetail();
 if(resolved?.playableRefs.length){
  void guarded(()=>player.playContext(resolved.playableRefs,0));
 }
});

playlistDetailShuffle.addEventListener("click",()=>{
 const resolved=currentPlaylistDetail();
 const shuffled=shuffleRefs([...(resolved?.playableRefs??[])]);
 if(shuffled.length){
  void guarded(()=>player.playContext(shuffled,0));
 }
});

playlistDetailAddSongs.addEventListener("click",()=>{
 const resolved=currentPlaylistDetail();
 if(!resolved)return;
 playlistSelectionTargetID=resolved.playlist.id;
 librarySelection.activate();
 shellNavigation.select(ShellView.SONGS);
 renderTrackChoices();
 bulkPlaylistTarget.value=resolved.playlist.id;
});

playlistDetailRename.addEventListener("click",()=>void guarded(async()=>{
 const resolved=currentPlaylistDetail();
 if(!resolved)return;
 await renamePlaylist(resolved.playlist.id,playlistDetailName.value);
 await refreshPlaylists();
}));

playlistDetailDelete.addEventListener("click",()=>void guarded(async()=>{
 const resolved=currentPlaylistDetail();
 if(!resolved)return;
 await deletePlaylist(resolved.playlist.id);
 await refreshPlaylists();
}));

playlistDetailEntries.addEventListener("dblclick",event=>{
 const resolved=currentPlaylistDetail();
 if(!resolved)return;
 const row=event.target.closest("[data-track-ref][data-available='true']");
 activateTrackRow(event,{
  row,
  ref:row?.dataset.trackRef,
  playbackRefs:resolved.playableRefs
 });
});

playlistDetailEntries.addEventListener("click",event=>{
 const resolved=currentPlaylistDetail();
 if(!resolved)return;
 if(handleTrackAction(event,{playbackRefs:resolved.playableRefs}))return;
 const action=event.target.closest("[data-playlist-entry-action]");
 const entry=event.target.closest("[data-playlist-index]");
 if(!action||!entry)return;
 const index=Number(entry.dataset.playlistIndex);
 const refs=[...resolved.playlist.trackRefs];

 if(action.dataset.playlistEntryAction==="remove"){
  const ref=refs[index];
  void guarded(async()=>{
   await removeTracksFromPlaylist(resolved.playlist.id,[ref]);
   await refreshPlaylists();
  });
  return;
 }

 const target=action.dataset.playlistEntryAction==="up"
  ?index-1
  :index+1;
 if(target<0||target>=refs.length)return;
 [refs[index],refs[target]]=[refs[target],refs[index]];
 void guarded(async()=>{
  await reorderPlaylist(resolved.playlist.id,refs);
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
   await loadImportReview(candidate.url,{selectYouTube:true});
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
 stopImportDuplicatePreview();
 importReviewLoadingState=false;
 importReviewLoadingURL="";
 currentImportReview=null;
 renderImportReview();
});

importDuplicateDetails.addEventListener("click",event=>{
 const button=event.target.closest("[data-import-duplicate-preview]");
 if(!button)return;
 previewImportDuplicate(button.dataset.importDuplicatePreview,button);
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
 await ensureImportRuntimeReady();
 await startImport(plan);
 stopImportDuplicatePreview();
 currentImportReview=null;
 youtubeImportURL.value="";
 await refreshImportJobs();
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
  case "view":
   shellNavigation.select(ShellView.SONGS);
   return;
  }
  await refreshImportJobs();
 });
});

trashRefresh.addEventListener("click",()=>void guarded(()=>refreshTrash()));

trashItems.addEventListener("change",event=>{
 const input=event.target.closest("[data-trash-select]");
 const row=event.target.closest("[data-trash-id]");
 if(!input||!row)return;
 if(input.checked)selectedTrashIDs.add(row.dataset.trashId);
 else selectedTrashIDs.delete(row.dataset.trashId);
 renderTrash();
});

trashItems.addEventListener("click",event=>{
 const action=event.target.closest("[data-trash-action]");
 const row=event.target.closest("[data-trash-id]");
 if(!action||!row)return;
 const id=row.dataset.trashId;

 if(action.dataset.trashAction==="restore"){
  void guarded(async()=>{
   await restoreTrashItems([id]);
   selectedTrashIDs.delete(id);
   await Promise.all([
    refreshTrash(),
    refreshCatalog()
   ]);
  });
  return;
 }

 if(action.dataset.trashAction==="purge"){
  openTrashPurgeConfirmation([id]);
 }
});

restoreSelectedTrashButton.addEventListener("click",()=>void guarded(async()=>{
 const ids=validTrashIDs([...selectedTrashIDs]);
 if(!ids.length)return;
 await restoreTrashItems(ids);
 selectedTrashIDs.clear();
 await Promise.all([
  refreshTrash(),
  refreshCatalog()
 ]);
}));

purgeSelectedTrashButton.addEventListener("click",()=>{
 openTrashPurgeConfirmation([...selectedTrashIDs]);
});

clearTrashSelectionButton.addEventListener("click",()=>{
 selectedTrashIDs.clear();
 renderTrash();
});

trashPurgeCancel.addEventListener("click",closeTrashPurgeConfirmation);

trashPurgeCommit.addEventListener("click",()=>void guarded(async()=>{
 const ids=validTrashIDs(pendingTrashPurgeIDs);
 if(!ids.length){
  closeTrashPurgeConfirmation();
  return;
 }
 await purgeTrashItems(ids,{
  preserveMetadata:preserveDeletedMetadata.checked
 });
 for(const id of ids)selectedTrashIDs.delete(id);
 closeTrashPurgeConfirmation();
 await refreshTrash();
}));

recoverTrashPurgeButton.addEventListener("click",()=>void guarded(async()=>{
 await recoverTrashPurge();
 await refreshTrash();
}));

queueCurrentPane.addEventListener("click",event=>{
 const relation=event.target.closest(
  "[data-now-library-kind][data-now-library-key]"
 );
 if(relation){
  shellNavigation.openLibraryDetail(
   relation.dataset.nowLibraryKind,
   relation.dataset.nowLibraryKey
  );
  return;
 }
 handleTrackAction(event);
});

shuffleQueueUpcoming.addEventListener("click",()=>{
 player.shuffleUpcoming();
});

queueList.addEventListener("dblclick",event=>{
 if(rowTargetOwnsInteraction(event.target))return;
 const row=event.target.closest("[data-queue-index]");
 if(!row)return;
 const index=Number(row.dataset.queueIndex);
 const state=queue.currentProjection.state;
 if(
  state.kind!=="positioned" ||
  index===state.currentIndex
 )return;
 event.preventDefault();
 void guarded(()=>player.playQueueIndex(index));
});

queueList.addEventListener("click",event=>{
 if(handleTrackAction(event,{playbackRefs:libraryPlaybackRefs()}))return;
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
  return;
 }
 if(action.dataset.queueAction==="down"){
  player.moveQueue(index,index+2);
  return;
 }
 if(action.dataset.queueAction==="remove"){
  player.removeQueue(index);
 }
});

function clearQueueDragState(){
 queueDragIndex=-1;
 queueDropIndex=-1;
 queueDropPlaceholder.remove();
 queueDropPlaceholder.hidden=true;
 for(const row of queueList.querySelectorAll(".dragging")){
  row.classList.remove("dragging");
 }
}

function placeQueueDropPreview(boundary,before){
 if(
  boundary===queueDropIndex &&
  queueDropPlaceholder.isConnected
 )return;
 queueDropPlaceholder.remove();
 queueDropIndex=boundary;
 queueDropPlaceholder.dataset.dropIndex=String(boundary);
 queueDropPlaceholder.style.height=`${QUEUE_ROW_HEIGHT}px`;
 queueDropPlaceholder.hidden=false;
 if(before)queueList.insertBefore(queueDropPlaceholder,before);
 else queueList.append(queueDropPlaceholder);
}

function updateQueueDropPreview(event){
 const state=queue.currentProjection.state;
 if(state.kind!=="positioned"||queueDragIndex<0)return;

 if(event.target.closest(".queue-drop-placeholder"))return;

 const row=event.target.closest("[data-queue-index]");
 if(row){
  const index=Number(row.dataset.queueIndex);
  const rect=row.getBoundingClientRect();
  const beforePointer=event.clientY<rect.top+(rect.height/2);
  const boundary=beforePointer?index:index+1;
  if(boundary===queueDropIndex&&queueDropPlaceholder.isConnected)return;
  queueDropPlaceholder.remove();
  const before=beforePointer?row:row.nextElementSibling;
  placeQueueDropPreview(boundary,before);
  return;
 }

 const spacer=event.target.closest("[data-queue-virtual]");
 if(!spacer)return;
 const start=Number(queueList.dataset.windowStart||0);
 const end=Number(queueList.dataset.windowEnd||state.items.length);
 if(spacer.dataset.queueVirtual==="top"){
  placeQueueDropPreview(start,spacer.nextElementSibling);
  return;
 }
 placeQueueDropPreview(end,spacer);
}

queueList.addEventListener("dragstart",event=>{
 if(event.target.closest("button,summary,details,select,input,label,a")){
  event.preventDefault();
  return;
 }
 const row=event.target.closest("[data-queue-index]");
 if(!row||!event.dataTransfer)return;
 queueDragIndex=Number(row.dataset.queueIndex);
 queueDropIndex=-1;
 event.dataTransfer.effectAllowed="move";
 event.dataTransfer.setData("text/plain",String(queueDragIndex));
 row.classList.add("dragging");
});

queueList.addEventListener("dragover",event=>{
 if(queueDragIndex<0)return;
 event.preventDefault();
 if(event.dataTransfer)event.dataTransfer.dropEffect="move";
 updateQueueDropPreview(event);
});

queueList.addEventListener("drop",event=>{
 if(queueDragIndex<0)return;
 event.preventDefault();
 const from=queueDragIndex>=0
  ?queueDragIndex
  :Number(event.dataTransfer?.getData("text/plain"));
 const to=queueDropIndex;
 clearQueueDragState();
 if(!Number.isInteger(from)||!Number.isInteger(to))return;
 if(from===to||from+1===to)return;
 player.moveQueue(from,to);
});

queueList.addEventListener("dragend",clearQueueDragState);

queueTimelinePane.addEventListener("scroll",()=>{
 if(queueDragIndex>=0||queueScrollFrame!==null)return;
 queueScrollFrame=requestAnimationFrame(()=>{
  queueScrollFrame=null;
  const state=queue.currentProjection.state;
  if(
   state.kind!=="positioned" ||
   !queueDrawerOpen ||
   !queueWindowInitialized
  )return;
  const relative=Math.max(
   0,
   queueTimelinePane.scrollTop-queueList.offsetTop
  );
  const firstVisible=Math.max(
   0,
   Math.min(
    state.items.length-1,
    Math.floor(relative/QUEUE_ROW_HEIGHT)
   )
  );
  const nextWindow=queueWindowForScroll({
   length:state.items.length,
   start:queueWindowStart,
   firstVisible,
   policy:QueueWindowPolicy
  });
  if(!nextWindow.changed)return;
  queueWindowStart=nextWindow.start;
  renderQueueWindow(state,{force:true});
 });
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
 observedImportCompletionKey=importCompletionKey(currentImportJobs);
 scheduleImportJobRefresh();

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
 renderTrash();
 renderShell();
 render();
 window.vexV4=Object.freeze({
  ready:true,
  get identity(){return systemIdentity;},
  get systemStatus(){return currentSystemStatus;},
  journey:playerJourney,
  queue,
  playback,
  timeline,
  volume:playbackVolume,
  player,
  media:audio,
  get catalog(){return catalog;},
  get libraryView(){return libraryView;},
  shellNavigation,
  discoverNavigation,
  librarySelection,
  playlistNavigation,
  phoneConnection,
  get sourceConfig(){return sourceConfig;},
  get playlists(){return playlists;},
  get resolvedPlaylists(){return resolvedPlaylists;},
  get trash(){return trashSnapshot;},
  get editingTrackRef(){return currentEditingTrackRef;},
  exploreSession,
  explore,
  get importReview(){return currentImportReview;},
  get importJobs(){return currentImportJobs;},
  get importRuntime(){return importRuntimeState;},
  get sessionRestoration(){return currentSessionRestoration;},
  controls:Object.freeze({PlaybackControl,QueueControl})
 });
 void refreshImportRuntime().catch(failure=>{
  importRuntimeState=Object.freeze({
   searchReady:false,
   ffmpegReady:false,
   importReady:false,
   detail:failure?.message??String(failure)
  });
  renderImportRuntimeStatus();
 });
}

window.addEventListener("resize",()=>{
 requestAnimationFrame(()=>refreshSongMarqueeMetrics());
});

void initialize().catch(failure=>{
 error.textContent=failure?.message??String(failure);
});
