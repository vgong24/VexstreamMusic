// [VXG RealForever]
// Native VexStream Music -> VexBoard Screen Lens exporter.
// Reads VexStream-owned semantic tokens and the lived HTML/CSS shell.
// Product JavaScript is never exported; data-vex-ref exists only in the inert copy.

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { PlaybackAct, PlaybackControl, PlaybackEffect, VexStreamPlaybackFamily } from "../web/features/playback/behavior.js";
import { QueueAct, QueueControl, QueueEffect, VexStreamQueueFamily } from "../web/features/queue/behavior.js";
import { PhoneConnectionIntent } from "../web/features/phone/connection.js";
import { SystemStatusIntent } from "../web/features/system/status.js";
import { DiscoverMode } from "../web/features/discover/navigation.js";
import { SourceClientIntent } from "../web/features/sources/client.js";
import { ImportClientIntent } from "../web/features/imports/client.js";
import { PlaylistClientIntent } from "../web/features/playlists/client.js";
import { PlaylistNavigationIntent } from "../web/features/playlists/navigation.js";
import { PlayerCompositionIntent } from "../web/features/player/composition.js";
import {
  VexPresentationContinuity,
  VEX_PRESENTATION_INSTANCE_ENCODING,
} from "../web/foundation/identity.js";

export const VEXBOARD_SCREEN_LENS_SCHEMA="vexboard.screen-lens/v1";
export const VEXSTREAM_VEXBOARD_EXPORTER_VERSION="vexstream-vexboard-exporter/1";
const HERE=path.dirname(fileURLToPath(import.meta.url));
const V2=path.resolve(HERE,"..");
const SCREEN_REF="vexstream.desktop.library";
const FRAME_REF="vexstream.desktop.library.frame";
const SHELL_REF="vexstream.desktop.library.region.shell";
const SHELL_TERRAIN_REF="vexstream.desktop.library.terrain";
const HOME_PANEL_REF="vexstream.desktop.library.panel.home";
const LIBRARY_PANEL_REF="vexstream.desktop.library.panel.library";
const DISCOVER_PANEL_REF="vexstream.desktop.library.panel.discover";
const PLAYLISTS_PANEL_REF="vexstream.desktop.library.panel.playlists";
const PLAYLISTS_OVERVIEW_REF="vexstream.desktop.library.playlists.overview";
const PLAYLIST_DETAIL_REF="vexstream.desktop.library.playlists.detail";
const DISCOVER_MODE_REF="vexstream.desktop.library.discover-mode-selection";
const DISCOVER_PC_PANEL_REF="vexstream.desktop.library.discover-panel.pc";
const DISCOVER_YOUTUBE_PANEL_REF="vexstream.desktop.library.discover-panel.youtube";
const DISCOVER_EXPLORE_PANEL_REF="vexstream.desktop.library.discover-panel.explore";
const LIBRARY_MODE_REF="vexstream.desktop.library.mode-selection";
const LIBRARY_MODE_SONGS_REF="vexstream.desktop.library.mode-state.songs";
const LIBRARY_MODE_ALBUMS_REF="vexstream.desktop.library.mode-state.albums";
const LIBRARY_MODE_ARTISTS_REF="vexstream.desktop.library.mode-state.artists";
const TRACK_CHOICES_REF="vexstream.desktop.library.track-choices";
const SHELL_SELECTION_REF="presence.vexstream.shell.primary-terrain";
const LIBRARY_MODE_SELECTION_REF="presence.vexstream.library.mode";
const DISCOVER_MODE_SELECTION_REF="presence.vexstream.discover.mode";
const PLAYLIST_DETAIL_PRESENTATION_REF="presence.vexstream.playlists.detail-open";
const PLAYLIST_CARD_POPULATION_REF="presence.vexstream.playlists.cards";
const PLAYLIST_ENTRY_POPULATION_REF="presence.vexstream.playlists.detail.entries";
const PLAYLIST_CARD_TEMPLATE_REF="template.vexstream.playlists.card";
const PLAYLIST_ENTRY_TEMPLATE_REF="template.vexstream.playlists.detail-entry";
const PLAYLIST_CARD_INSTANCE_BASE="element.vexstream.playlists.card";
const PLAYLIST_ENTRY_INSTANCE_BASE="element.vexstream.playlists.entry";
const PREVIOUS_ACTION="vexstream.player.previous";
const semanticIdentity=token=>token.familyRef+"."+token.ref;
const elementRef=identity=>"element."+identity;
const source=relative=>"v2:"+relative;

const RELATIONSHIP_ATTRIBUTES=Object.freeze([
  "data-vex-control",
  "data-vex-action",
  "data-shell-view",
  "data-library-mode",
  "data-discover-mode",
  "data-phone-action",
  "data-system-action",
  "data-track-action",
]);

function attributeValue(raw,attribute){
  return String(raw).match(new RegExp("\\b"+attribute+"=\"([^\"]+)\""))?.[1]??null;
}

function relationshipIdentity(attribute,value){
  if(attribute==="data-vex-control"||attribute==="data-vex-action")return value;
  if(attribute==="data-shell-view")return "vexstream.shell.view."+value;
  if(attribute==="data-library-mode")return "vexstream.library.mode."+value;
  if(attribute==="data-discover-mode")return "vexstream.discover.mode."+value;
  if(attribute==="data-phone-action")return "vexstream.phone.action."+value;
  if(attribute==="data-system-action")return "vexstream.system.action."+value;
  if(attribute==="data-track-action")return "vexstream.track.action."+value;
  return null;
}

function sourceRelationshipForTag(tagSource){
  const tag=String(tagSource).match(/^<\s*([A-Za-z][\w:-]*)/)?.[1]?.toLowerCase()??"button";
  for(const attribute of RELATIONSHIP_ATTRIBUTES){
    const value=attributeValue(tagSource,attribute);
    if(!value)continue;
    return Object.freeze({attribute,value,identity:relationshipIdentity(attribute,value),tag});
  }
  return null;
}

function stableSourceRelationships(html){
  const found=[];
  const byIdentity=new Map();
  for(const match of String(html).matchAll(/<[A-Za-z][^>]*>/g)){
    const relation=sourceRelationshipForTag(match[0]);
    if(!relation)continue;
    if(!byIdentity.has(relation.identity)){
      byIdentity.set(relation.identity,relation);
      found.push(relation);
    }
  }
  return Object.freeze(found);
}

function identityLabel(identity){
  return identity.split(".").slice(-2).join(" / ").replaceAll("-"," ");
}

function relationLabel(relation){
  const title=String(relation.value).replaceAll("-"," ").replace(/\b\w/g,(char)=>char.toUpperCase());
  if(relation.attribute==="data-shell-view")return "Shell view: "+title;
  if(relation.attribute==="data-library-mode")return "Library mode: "+title;
  if(relation.attribute==="data-discover-mode")return "Discover mode: "+title;
  if(relation.attribute==="data-track-action")return "Track action: "+title;
  if(relation.attribute==="data-phone-action")return "Phone action: "+title;
  if(relation.attribute==="data-system-action")return "System action: "+title;
  return identityLabel(relation.identity);
}

function viewHTML(sourceHTML){
  let html=String(sourceHTML);
  html=html.replace('<body data-vex-screen="vexstream.desktop.library" data-shell-current-view="home">','<body data-vex-screen="vexstream.desktop.library" data-shell-current-view="home" data-vex-structural-ref="'+FRAME_REF+'">');
  html=html.replace('<main class="shell">','<main class="shell" data-vex-structural-ref="'+SHELL_REF+'">');
  html=html.replace(/<section([^>]*\bid="homeView"[^>]*\bdata-shell-panel="home"[^>]*)>/,
    '<section$1 data-vex-structural-ref="'+HOME_PANEL_REF+'">');
  html=html.replace(/<section([^>]*\bdata-shell-panel="library"[^>]*)>/,
    '<section$1 data-vex-structural-ref="'+LIBRARY_PANEL_REF+'">');
  html=html.replace(/<section([^>]*\bdata-shell-panel="discover"[^>]*)>/,
    '<section$1 data-vex-structural-ref="'+DISCOVER_PANEL_REF+'">');
  html=html.replace(/<section([^>]*\bdata-shell-panel="playlists"[^>]*)>/,
    '<section$1 data-vex-structural-ref="'+PLAYLISTS_PANEL_REF+'">');
  html=html.replace(/<div([^>]*\bid="playlistOverview"[^>]*)>/,
    '<div$1 data-vex-structural-ref="'+PLAYLISTS_OVERVIEW_REF+'">');
  html=html.replace(/<section([^>]*\bid="playlistDetail"[^>]*)>/,
    '<section$1 data-vex-structural-ref="'+PLAYLIST_DETAIL_REF+'">');
  html=html.replace(/<section([^>]*\bdata-discover-panel="pc"[^>]*)>/,
    '<section$1 data-vex-structural-ref="'+DISCOVER_PC_PANEL_REF+'">');
  html=html.replace(/<section([^>]*\bdata-discover-panel="youtube"[^>]*)>/,
    '<section$1 data-vex-structural-ref="'+DISCOVER_YOUTUBE_PANEL_REF+'">');
  html=html.replace(/<section([^>]*\bdata-discover-panel="explore"[^>]*)>/,
    '<section$1 data-vex-structural-ref="'+DISCOVER_EXPLORE_PANEL_REF+'">');
  html=html.replace(/<div([^>]*\bid="trackChoices"[^>]*)>/,
    '<div$1 data-vex-structural-ref="'+TRACK_CHOICES_REF+'">');
  html=html.replace(/<[A-Za-z][^>]*>/g,(tagSource)=>{
    if(/\bdata-vex-ref=/.test(tagSource))return tagSource;
    const relation=sourceRelationshipForTag(tagSource);
    if(!relation)return tagSource;
    return tagSource.replace(/>$/,' data-vex-ref="'+elementRef(relation.identity)+'">');
  });
  return html;
}

function node(ref,face,role,label,options={}){
  return {ref,face,role,label,
    ...(options.symbol?{symbol:options.symbol}:{}),
    ...(options.summary?{summary:options.summary}:{}),
    ...(options.sourcePath?{source:source(options.sourcePath)}:{}),
  };
}
const edge=(from,to,label=null,kind="FLOW")=>({from,to,...(label?{label}:{}),kind});

function playbackPathway(){
  const p="pathway.vexstream.playback.play-pause";
  const ctl=p+".control", state=p+".state", act=p+".act", effect=p+".effect", projection=p+".projection", method=p+".player", connection=p+".connection", observation=p+".observation";
  return {
    pathwayRef:p,label:"Play / Pause",disposition:"SCAE_FAMILY",
    summary:"Playback owns Play/Pause meaning. Player composition may seed or load listening context before the same Playback Control enters the Family; provider success remains HTMLMediaElement truth.",
    returns:"Observed media truth -> current Playback State -> current Play/Pause presentation",returnRef:projection,
    nodes:[
      node(ctl,"BEHAVIOR","CONTROL","PlaybackControl.PlayPause",{symbol:semanticIdentity(PlaybackControl.PlayPause),sourcePath:"web/features/playback/behavior.js"}),
      node(state,"BEHAVIOR","STATE","PlaybackState",{symbol:VexStreamPlaybackFamily,summary:"no-media / media / ended semantic truth",sourcePath:"web/features/playback/behavior.js"}),
      node(act,"BEHAVIOR","ACT","Play / Pause",{symbol:PlaybackAct.Play.ref+" | "+PlaybackAct.Pause.ref,sourcePath:"web/features/playback/behavior.js"}),
      node(effect,"BEHAVIOR","EFFECT","Request play / pause / restart current",{symbol:PlaybackEffect.RequestPlay.ref+" | "+PlaybackEffect.RequestPause.ref+" | "+PlaybackEffect.RestartCurrent.ref,sourcePath:"web/features/playback/behavior.js"}),
      node(projection,"BEHAVIOR","PROJECTION","Play / Pause presentation",{sourcePath:"web/features/playback/presentation.js"}),
      node(method,"DEPTH","METHOD","Player.activatePlayPause()",{summary:"Higher composition seeds Queue or media only when listening context requires it.",sourcePath:"web/features/player/composition.js"}),
      node(connection,"DEPTH","CONNECTION","HTML media Playback connection",{symbol:"createHTMLMediaPlaybackConnection",summary:"Re-resolves Control and keeps request-pending distinct from provider acknowledgement.",sourcePath:"web/features/playback/connection.js"}),
      node(observation,"DEPTH","OBSERVATION","HTMLMediaElement truth",{summary:"paused, ended and play acknowledgement remain provider observations.",sourcePath:"web/features/playback/connection.js"}),
    ],
    edges:[
      edge(ctl,act,"resolves with current State"),edge(state,act,null,"DATA"),edge(act,effect,"Family resolves"),
      edge(effect,method,"realized through composition"),edge(method,connection),edge(connection,observation),
      edge(observation,state,"observed truth","RETURN"),edge(state,projection,"projects"),
    ],
  };
}

function queuePathway(key,control,actLabel,effectLabel,methodLabel){
  const p="pathway.vexstream.queue."+key;
  const ctl=p+".control", state=p+".state", act=p+".act", effect=p+".effect", projection=p+".projection", method=p+".player", realization=p+".queue-connection";
  return {
    pathwayRef:p,label:key[0].toUpperCase()+key.slice(1),disposition:"SCAE_FAMILY",
    summary:"Queue owns the Control/State/Act/Effect relationship; Player composition may carry current Queue position into Playback realization without becoming Queue meaning.",
    returns:"Current Queue State -> current control presentation",returnRef:projection,
    nodes:[
      node(ctl,"BEHAVIOR","CONTROL","QueueControl."+control.ref,{symbol:semanticIdentity(control),sourcePath:"web/features/queue/behavior.js"}),
      node(state,"BEHAVIOR","STATE","QueueState",{symbol:VexStreamQueueFamily,summary:"empty or positioned Queue truth plus shuffle/repeat flags",sourcePath:"web/features/queue/behavior.js"}),
      node(act,"BEHAVIOR","ACT",actLabel,{sourcePath:"web/features/queue/behavior.js"}),
      node(effect,"BEHAVIOR","EFFECT",effectLabel,{sourcePath:"web/features/queue/behavior.js"}),
      node(projection,"BEHAVIOR","PROJECTION",key+" presentation",{sourcePath:"web/features/queue/presentation.js"}),
      node(method,"DEPTH","METHOD",methodLabel,{summary:"Human listening composition delegates Queue meaning to QueueConnection.",sourcePath:"web/features/player/composition.js"}),
      node(realization,"DEPTH","REALIZATION","QueueConnection.activate()",{symbol:"createQueueConnection",sourcePath:"web/features/queue/connection.js"}),
    ],
    edges:[
      edge(ctl,act,"resolves with current State"),edge(state,act,null,"DATA"),edge(act,effect,"Family resolves"),
      edge(effect,method,"realized through composition"),edge(method,realization),edge(realization,state,"publishes current Queue","RETURN"),edge(state,projection,"projects"),
    ],
  };
}

function previousPathway(){
  const p="pathway.vexstream.player.previous";
  const intent=p+".intent", dispatch=p+".dispatch", playback=p+".playback-state", queue=p+".queue-state", method=p+".method", restart=p+".restart", move=p+".queue-previous", realization=p+".realization";
  return {
    pathwayRef:p,label:"Previous listening action",disposition:"DIRECT_REGISTRATION",
    summary:"The fixed Previous affordance belongs to Player composition, not unconditionally to QueueControl.Previous. Playback time and Queue position choose which already-owned relationship is used.",
    returns:"Current Playback / Queue truth after the selected composition branch",returnRef:realization,
    nodes:[
      node(intent,"BEHAVIOR","INTENT","vexstream.player.previous",{symbol:PREVIOUS_ACTION,sourcePath:"web/index.html"}),
      node(dispatch,"BEHAVIOR","DISPATCH","player.previous()",{summary:"Top-level listening composition.",sourcePath:"web/app.js"}),
      node(playback,"BEHAVIOR","STATE","Current Playback time",{summary:"Elapsed time may make Previous mean restart-current.",sourcePath:"web/features/player/composition.js"}),
      node(queue,"BEHAVIOR","STATE","Current Queue position",{summary:"Near the start, Previous may delegate to QueueControl.Previous when lawful.",sourcePath:"web/features/queue/behavior.js"}),
      node(method,"DEPTH","METHOD","Player.previous()",{sourcePath:"web/features/player/composition.js"}),
      node(restart,"DEPTH","METHOD","Restart current Playback branch",{sourcePath:"web/features/player/composition.js"}),
      node(move,"DEPTH","METHOD","Queue Previous branch",{sourcePath:"web/features/player/composition.js"}),
      node(realization,"DEPTH","REALIZATION","Restart or load current media",{sourcePath:"web/features/player/composition.js"}),
    ],
    edges:[
      edge(intent,dispatch),edge(playback,dispatch,null,"DATA"),edge(queue,dispatch,null,"DATA"),edge(dispatch,method),
      edge(method,restart,"elapsed beyond restart threshold","SIDE"),edge(method,move,"near start with prior Queue item","SIDE"),
      edge(restart,realization),edge(move,realization),
    ],
  };
}

function phoneConnectionPathway(){
  const p="pathway.vexstream.phone.connection.open";
  const intent=p+".intent", dispatch=p+".dispatch", connection=p+".connection", load=p+".load", start=p+".start", projection=p+".projection";
  return {
    pathwayRef:p,label:"Open Phone Connection",disposition:"DIRECT_REGISTRATION",
    summary:"The phone button opens the existing Phone Connection walk. The Connection observes remembered/pending pairing and may start the five-minute pairing window when no phone is remembered.",
    returns:"Current Phone Connection projection shown in the shared modal shell",returnRef:projection,
    nodes:[
      node(intent,"BEHAVIOR","INTENT","Open Phone Connection",{symbol:PhoneConnectionIntent.Open,sourcePath:"web/features/phone/connection.js"}),
      node(dispatch,"BEHAVIOR","DISPATCH","phoneConnection.open()",{sourcePath:"web/app.js"}),
      node(projection,"BEHAVIOR","PROJECTION","Phone Connection presentation",{summary:"Loading / ready / failed pairing projection rendered in the existing modal shell.",sourcePath:"web/features/phone/presentation.js"}),
      node(connection,"DEPTH","CONNECTION","Phone Connection",{symbol:"createPhoneConnection",summary:"Owns open/refresh/polling/action presentation lifetime without owning the localhost pairing contract.",sourcePath:"web/features/phone/connection.js"}),
      node(load,"DEPTH","METHOD","loadPhonePairing()",{summary:"GET current localhost pairing status.",sourcePath:"web/features/phone/client.js"}),
      node(start,"DEPTH","METHOD","startPhonePairing()",{summary:"POST a five-minute pairing window only when open() observes no remembered phone and no window already open.",sourcePath:"web/features/phone/client.js"}),
    ],
    edges:[
      edge(intent,dispatch),
      edge(dispatch,connection),
      edge(connection,load,"observe"),
      edge(connection,start,"auto-start when needed","SIDE"),
      edge(start,load,"refresh after start"),
      edge(load,projection,"projects current pairing","RETURN"),
    ],
  };
}

function systemStatusPathway(){
  const p="pathway.vexstream.system.status.open";
  const intent=p+".intent", dispatch=p+".dispatch", method=p+".projection-method", projection=p+".projection";
  return {
    pathwayRef:p,label:"Open System status",disposition:"DIRECT_REGISTRATION",
    summary:"The bell opens a read-only System status projection composed from truths already owned by System identity, Library, Sources, Player session, Imports and Trash. It performs no repair or lifecycle effect.",
    returns:"Current System status drawer projection",returnRef:projection,
    nodes:[
      node(intent,"BEHAVIOR","INTENT","Open System status",{symbol:SystemStatusIntent.Open,sourcePath:"web/features/system/status.js"}),
      node(dispatch,"BEHAVIOR","DISPATCH","renderSystemStatus()",{sourcePath:"web/app.js"}),
      node(projection,"BEHAVIOR","PROJECTION","System status drawer",{summary:"Healthy/attention summary plus current build, library, session, import and trash facts.",sourcePath:"web/features/system/status.js"}),
      node(method,"DEPTH","METHOD","projectSystemStatus()",{summary:"Pure read-only composition over already-owned current facts.",sourcePath:"web/features/system/status.js"}),
    ],
    edges:[
      edge(intent,dispatch),
      edge(dispatch,method),
      edge(method,projection,"projects","RETURN"),
    ],
  };
}

function playlistDetailClosePathway(){
  const p="pathway.vexstream.playlists.detail.close";
  const intent=p+".intent", dispatch=p+".dispatch", method=p+".method", projection=p+".projection";
  return {
    pathwayRef:p,label:"Close Playlist detail",disposition:"DIRECT_REGISTRATION",
    summary:"PlaylistNavigation owns whether the human is browsing all playlists or one selected durable playlist.",
    returns:"Playlists overview presentation",returnRef:projection,
    nodes:[
      node(intent,"BEHAVIOR","INTENT","Close Playlist detail",{symbol:PlaylistNavigationIntent.CloseDetail,sourcePath:"web/features/playlists/navigation.js"}),
      node(dispatch,"BEHAVIOR","DISPATCH","playlistNavigation.close()",{sourcePath:"web/app.js"}),
      node(projection,"BEHAVIOR","PROJECTION","Playlists overview",{summary:"Selected-playlist detail closes and overview returns.",sourcePath:"web/features/playlists/navigation.js"}),
      node(method,"DEPTH","METHOD","PlaylistNavigation.close()",{sourcePath:"web/features/playlists/navigation.js"}),
    ],
    edges:[edge(intent,dispatch),edge(dispatch,method),edge(method,projection,"projects","RETURN")],
  };
}

function playlistPlayPathway(){
  const p="pathway.vexstream.player.play-playlist";
  const intent=p+".intent", state=p+".state", dispatch=p+".dispatch", method=p+".method", realization=p+".realization";
  return {
    pathwayRef:p,label:"Play selected Playlist",disposition:"DIRECT_REGISTRATION",
    summary:"Player composition consumes the selected Playlist's currently playable refs without taking durable Playlist ownership.",
    returns:"Current Queue/Playback listening state",returnRef:realization,
    nodes:[
      node(intent,"BEHAVIOR","INTENT","Play selected Playlist",{symbol:PlayerCompositionIntent.PlayPlaylist,sourcePath:"web/features/player/composition.js"}),
      node(state,"BEHAVIOR","STATE","Resolved Playlist playable refs",{summary:"Playlist owns durable refs; Library availability decides which are playable now.",sourcePath:"web/features/playlists/client.js"}),
      node(dispatch,"BEHAVIOR","DISPATCH","player.playContext(resolved.playableRefs, 0)",{sourcePath:"web/app.js"}),
      node(method,"DEPTH","METHOD","Player.playContext()",{sourcePath:"web/features/player/composition.js"}),
      node(realization,"DEPTH","REALIZATION","Queue + Playback composition",{summary:"Player composition forms the temporary listening context and realizes current media through existing Queue/Playback owners.",sourcePath:"web/features/player/composition.js"}),
    ],
    edges:[edge(intent,dispatch),edge(state,dispatch,null,"DATA"),edge(dispatch,method),edge(method,realization)],
  };
}

function playlistShufflePathway(){
  const p="pathway.vexstream.player.shuffle-playlist";
  const intent=p+".intent", state=p+".state", dispatch=p+".dispatch", shuffle=p+".shuffle", realization=p+".realization";
  return {
    pathwayRef:p,label:"Shuffle selected Playlist",disposition:"DIRECT_REGISTRATION",
    summary:"Player composition shuffles the selected Playlist's currently playable refs, then starts that temporary listening context.",
    returns:"Current shuffled Queue/Playback listening state",returnRef:realization,
    nodes:[
      node(intent,"BEHAVIOR","INTENT","Shuffle selected Playlist",{symbol:PlayerCompositionIntent.ShufflePlaylist,sourcePath:"web/features/player/composition.js"}),
      node(state,"BEHAVIOR","STATE","Resolved Playlist playable refs",{sourcePath:"web/features/playlists/client.js"}),
      node(dispatch,"BEHAVIOR","DISPATCH","shuffleRefs([...resolved.playableRefs])",{sourcePath:"web/app.js"}),
      node(shuffle,"DEPTH","METHOD","shuffleRefs()",{sourcePath:"web/app.js"}),
      node(realization,"DEPTH","REALIZATION","Player.playContext(shuffled, 0)",{sourcePath:"web/features/player/composition.js"}),
    ],
    edges:[edge(intent,dispatch),edge(state,dispatch,null,"DATA"),edge(dispatch,shuffle),edge(shuffle,realization)],
  };
}

function playlistAddSongsPathway(){
  const p="pathway.vexstream.playlists.detail.add-songs";
  const intent=p+".intent", state=p+".state", dispatch=p+".dispatch", select=p+".selection", navigate=p+".navigate", projection=p+".projection";
  return {
    pathwayRef:p,label:"Add songs to selected Playlist",disposition:"DIRECT_REGISTRATION",
    summary:"The selected Playlist ID becomes the target for the existing Library track-selection flow; PlaylistNavigation does not absorb Library selection ownership.",
    returns:"Library Songs terrain with selection active and target Playlist retained",returnRef:projection,
    nodes:[
      node(intent,"BEHAVIOR","INTENT","Add songs",{symbol:PlaylistNavigationIntent.AddSongs,sourcePath:"web/features/playlists/navigation.js"}),
      node(state,"BEHAVIOR","STATE","Current selected Playlist ID",{sourcePath:"web/features/playlists/navigation.js"}),
      node(dispatch,"BEHAVIOR","DISPATCH","playlistSelectionTargetID = resolved.playlist.id",{sourcePath:"web/app.js"}),
      node(projection,"BEHAVIOR","PROJECTION","Library selection targeting Playlist",{sourcePath:"web/app.js"}),
      node(select,"DEPTH","METHOD","librarySelection.activate()",{sourcePath:"web/features/library/selection.js"}),
      node(navigate,"DEPTH","METHOD","ShellNavigation.select(Songs)",{sourcePath:"web/features/shell/navigation.js"}),
    ],
    edges:[edge(intent,dispatch),edge(state,dispatch,null,"DATA"),edge(dispatch,select),edge(select,navigate),edge(navigate,projection,"projects","RETURN")],
  };
}

function playlistRenamePathway(){
  const p="pathway.vexstream.playlists.rename";
  const intent=p+".intent", state=p+".state", dispatch=p+".dispatch", method=p+".method", projection=p+".projection";
  return {
    pathwayRef:p,label:"Rename selected Playlist",disposition:"DIRECT_REGISTRATION",
    summary:"Playlist client owns the durable rename mutation; the detail name field supplies the proposed name and does not become a second semantic owner.",
    returns:"Refreshed selected Playlist detail",returnRef:projection,
    nodes:[
      node(intent,"BEHAVIOR","INTENT","Rename Playlist",{symbol:PlaylistClientIntent.Rename,sourcePath:"web/features/playlists/client.js"}),
      node(state,"BEHAVIOR","STATE","Selected Playlist + proposed name",{symbol:PlaylistClientIntent.RenameName,sourcePath:"web/features/playlists/client.js"}),
      node(dispatch,"BEHAVIOR","DISPATCH","renamePlaylist(id, playlistDetailName.value)",{sourcePath:"web/app.js"}),
      node(projection,"BEHAVIOR","PROJECTION","Renamed Playlist detail",{sourcePath:"web/app.js"}),
      node(method,"DEPTH","METHOD","POST /api/playlists/rename",{sourcePath:"web/features/playlists/client.js"}),
    ],
    edges:[edge(intent,dispatch),edge(state,dispatch,null,"DATA"),edge(dispatch,method),edge(method,projection,"RETURN")],
  };
}

function playlistDeletePathway(){
  const p="pathway.vexstream.playlists.delete";
  const intent=p+".intent", state=p+".state", dispatch=p+".dispatch", method=p+".method", projection=p+".projection";
  return {
    pathwayRef:p,label:"Delete selected Playlist",disposition:"DIRECT_REGISTRATION",
    summary:"Playlist client deletes only the durable Playlist record; Library media truth is not deleted by this action.",
    returns:"Refreshed Playlists terrain; PlaylistNavigation reconciliation closes a missing detail",returnRef:projection,
    nodes:[
      node(intent,"BEHAVIOR","INTENT","Delete Playlist",{symbol:PlaylistClientIntent.Delete,sourcePath:"web/features/playlists/client.js"}),
      node(state,"BEHAVIOR","STATE","Selected Playlist ID",{sourcePath:"web/features/playlists/navigation.js"}),
      node(dispatch,"BEHAVIOR","DISPATCH","deletePlaylist(resolved.playlist.id)",{sourcePath:"web/app.js"}),
      node(projection,"BEHAVIOR","PROJECTION","Playlists terrain after refresh",{sourcePath:"web/app.js"}),
      node(method,"DEPTH","METHOD","POST /api/playlists/delete",{sourcePath:"web/features/playlists/client.js"}),
    ],
    edges:[edge(intent,dispatch),edge(state,dispatch,null,"DATA"),edge(dispatch,method),edge(method,projection,"RETURN")],
  };
}

function mappedPathwayRef(identity){
  if(identity===PREVIOUS_ACTION)return "pathway.vexstream.player.previous";
  if(identity===PhoneConnectionIntent.Open)return "pathway.vexstream.phone.connection.open";
  if(identity===SystemStatusIntent.Open)return "pathway.vexstream.system.status.open";
  if(identity===semanticIdentity(PlaybackControl.PlayPause))return "pathway.vexstream.playback.play-pause";
  if(identity===semanticIdentity(QueueControl.Next))return "pathway.vexstream.queue.next";
  if(identity===semanticIdentity(QueueControl.Shuffle))return "pathway.vexstream.queue.shuffle";
  if(identity===semanticIdentity(QueueControl.Repeat))return "pathway.vexstream.queue.repeat";
  if(identity===PlaylistNavigationIntent.CloseDetail)return "pathway.vexstream.playlists.detail.close";
  if(identity===PlayerCompositionIntent.PlayPlaylist)return "pathway.vexstream.player.play-playlist";
  if(identity===PlayerCompositionIntent.ShufflePlaylist)return "pathway.vexstream.player.shuffle-playlist";
  if(identity===PlaylistNavigationIntent.AddSongs)return "pathway.vexstream.playlists.detail.add-songs";
  if(identity===PlaylistClientIntent.RenameName||identity===PlaylistClientIntent.Rename)return "pathway.vexstream.playlists.rename";
  if(identity===PlaylistClientIntent.Delete)return "pathway.vexstream.playlists.delete";
  return null;
}

function relationshipParentRef(relation){
  if(relation.attribute==="data-library-mode")return LIBRARY_MODE_REF;
  if(relation.attribute==="data-discover-mode")return DISCOVER_MODE_REF;
  if(Object.values(SourceClientIntent).includes(relation.identity))return DISCOVER_PC_PANEL_REF;
  if(Object.values(ImportClientIntent).includes(relation.identity))return DISCOVER_YOUTUBE_PANEL_REF;
  if(relation.identity===PlaylistClientIntent.CreateName||relation.identity===PlaylistClientIntent.Create)return PLAYLISTS_OVERVIEW_REF;
  if([
    PlaylistClientIntent.RenameName,PlaylistClientIntent.Rename,PlaylistClientIntent.Delete,
    PlaylistNavigationIntent.CloseDetail,PlaylistNavigationIntent.AddSongs,
    PlayerCompositionIntent.PlayPlaylist,PlayerCompositionIntent.ShufflePlaylist
  ].includes(relation.identity))return PLAYLIST_DETAIL_REF;
  if(relation.identity==="vexstream.explore.back"||relation.identity==="vexstream.explore.clear")return DISCOVER_EXPLORE_PANEL_REF;
  return SHELL_REF;
}

function interactionKind(relation){
  return ["input","select","textarea"].includes(relation.tag)?"FIELD":"BUTTON";
}

function interactionElement(relation,label,pathwayRef,parentRef=SHELL_REF){
  const declaredValue=relation.value??relation.identity;
  return {ref:elementRef(relation.identity),kind:interactionKind(relation),label,parentRef,symbol:relation.identity,declared:relation.attribute+'="'+declaredValue+'"',intent:relation.identity,...(pathwayRef?{pathwayRef}:{}),layer:"CHROME",interactions:["ACTIVATE"]};
}

export function buildVexStreamVexBoardLens({html,css}){
  const play=semanticIdentity(PlaybackControl.PlayPause);
  const next=semanticIdentity(QueueControl.Next);
  const shuffle=semanticIdentity(QueueControl.Shuffle);
  const repeat=semanticIdentity(QueueControl.Repeat);
  const relationships=stableSourceRelationships(html);
  const explicitLabels=new Map([
    [PREVIOUS_ACTION,"Previous"],
    [play,"Play / Pause"],
    [next,"Next"],
    [shuffle,"Shuffle"],
    [repeat,"Repeat"],
    [PhoneConnectionIntent.Open,"Connect VexLife phone"],
    [SystemStatusIntent.Open,"System status"],
    [PlaylistClientIntent.CreateName,"New playlist name"],
    [PlaylistClientIntent.Create,"Create playlist"],
    [PlaylistNavigationIntent.CloseDetail,"Back to all playlists"],
    [PlaylistClientIntent.RenameName,"Playlist name"],
    [PlayerCompositionIntent.PlayPlaylist,"Play playlist"],
    [PlayerCompositionIntent.ShufflePlaylist,"Shuffle playlist"],
    [PlaylistNavigationIntent.AddSongs,"Add songs"],
    [PlaylistClientIntent.Rename,"Rename playlist"],
    [PlaylistClientIntent.Delete,"Delete playlist"],
  ]);
  const stableElements=relationships.map((relation)=>
    interactionElement(
      relation,
      explicitLabels.get(relation.identity)??relationLabel(relation),
      mappedPathwayRef(relation.identity),
      relationshipParentRef(relation)
    ));
  const libraryModeButtons=Object.freeze({
    songs:elementRef("vexstream.library.mode.songs"),
    albums:elementRef("vexstream.library.mode.albums"),
    artists:elementRef("vexstream.library.mode.artists"),
  });
  const modeOperations=(mode)=>[
    {op:"SET_ATTRIBUTE",targetRef:SCREEN_REF,attribute:"data-shell-current-view",value:mode},
    {op:"SET_ATTRIBUTE",targetRef:TRACK_CHOICES_REF,attribute:"data-library-terrain",value:mode},
    ...Object.entries(libraryModeButtons).map(([key,ref])=>({
      op:"SET_ATTRIBUTE",targetRef:ref,attribute:"aria-pressed",value:String(key===mode)
    })),
  ];
  const discoverModeButtons=Object.freeze({
    pc:elementRef("vexstream.discover.mode.pc"),
    youtube:elementRef("vexstream.discover.mode.youtube"),
    explore:elementRef("vexstream.discover.mode.explore"),
  });
  const discoverModeOperations=(mode)=>[
    {op:"SET_ATTRIBUTE",targetRef:SCREEN_REF,attribute:"data-shell-current-view",value:"discover"},
    {op:"SET_ATTRIBUTE",targetRef:DISCOVER_PANEL_REF,attribute:"data-discover-terrain",value:mode},
    ...Object.entries(discoverModeButtons).map(([key,ref])=>({
      op:"SET_ATTRIBUTE",targetRef:ref,attribute:"aria-pressed",value:String(key===mode)
    })),
  ];
  return {
    schemaVersion:VEXBOARD_SCREEN_LENS_SCHEMA,lensRef:"lens.vexstream-music.desktop",title:"VexStream Music",
    summary:"Projection of the persistent V4 desktop shell with zero-blocker default-visible coverage plus source-authored Library, Discover, and Playlists terrain expressions.",
    entryScreenRef:SCREEN_REF,entryElementRef:elementRef(play),relatedBoardRefs:[],
    sourceSnapshot:{repositoryRef:"repo.vexstream-music",repositoryUrl:"https://github.com/vgong24/VexstreamMusic.git",commit:null,observedAt:null,roots:{v2:"v2/"},notes:[
      "Generated by "+VEXSTREAM_VEXBOARD_EXPORTER_VERSION+" from source-owned semantic modules and the lived shell.",
      "Product JavaScript is excluded from the VexBoard static view.",
      "Every source-stable relationship attribute in the bounded shell becomes addressable; repeated affordances may share one Lens relationship ref. Home/Library/Discover/Playlists plus Library and Discover nested modes are projected as browser-local source-authored expressions; deeper runtime/data states remain bounded.",
    ]},
    chrome:[],
    screens:[{
      screenRef:SCREEN_REF,label:"VexStream Music desktop",placement:"PRIMARY",symbol:"VexStream V4 lived shell",source:source("web/index.html"),mapped:"MAPPED",
      summary:"Persistent V3-shaped V4 desktop shell. ShellNavigation changes visible terrain while fixed Playback / Queue transport remains available.",
      view:{kind:"STATIC_HTML_CSS",html:viewHTML(html),css:String(css),scriptPolicy:"NONE",htmlSource:source("web/index.html"),cssSource:source("web/styles.css"),anchors:{screenAttribute:"data-vex-screen",elementAttribute:"data-vex-ref",structuralAttribute:"data-vex-structural-ref"},stateBindings:[
        {presenceRef:SHELL_SELECTION_REF,value:HOME_PANEL_REF,operations:[
          {op:"SET_ATTRIBUTE",targetRef:SCREEN_REF,attribute:"data-shell-current-view",value:"home"}
        ]},
        {presenceRef:SHELL_SELECTION_REF,value:LIBRARY_PANEL_REF,operations:[
          {op:"SET_ATTRIBUTE",targetRef:SCREEN_REF,attribute:"data-shell-current-view",value:"songs"}
        ]},
        {presenceRef:LIBRARY_MODE_SELECTION_REF,value:LIBRARY_MODE_SONGS_REF,operations:modeOperations("songs")},
        {presenceRef:LIBRARY_MODE_SELECTION_REF,value:LIBRARY_MODE_ALBUMS_REF,operations:modeOperations("albums")},
        {presenceRef:LIBRARY_MODE_SELECTION_REF,value:LIBRARY_MODE_ARTISTS_REF,operations:modeOperations("artists")},
        {presenceRef:SHELL_SELECTION_REF,value:DISCOVER_PANEL_REF,operations:[
          {op:"SET_ATTRIBUTE",targetRef:SCREEN_REF,attribute:"data-shell-current-view",value:"discover"}
        ]},
        {presenceRef:DISCOVER_MODE_SELECTION_REF,value:DISCOVER_PC_PANEL_REF,operations:discoverModeOperations("pc")},
        {presenceRef:DISCOVER_MODE_SELECTION_REF,value:DISCOVER_YOUTUBE_PANEL_REF,operations:discoverModeOperations("youtube")},
        {presenceRef:DISCOVER_MODE_SELECTION_REF,value:DISCOVER_EXPLORE_PANEL_REF,operations:discoverModeOperations("explore")},
        {presenceRef:SHELL_SELECTION_REF,value:PLAYLISTS_PANEL_REF,operations:[
          {op:"SET_ATTRIBUTE",targetRef:SCREEN_REF,attribute:"data-shell-current-view",value:"playlists"}
        ]},
        {presenceRef:PLAYLIST_DETAIL_PRESENTATION_REF,value:"OPEN",operations:[
          {op:"SET_ATTRIBUTE",targetRef:PLAYLISTS_OVERVIEW_REF,attribute:"hidden",value:""},
          {op:"REMOVE_ATTRIBUTE",targetRef:PLAYLIST_DETAIL_REF,attribute:"hidden"}
        ]},
        {presenceRef:PLAYLIST_DETAIL_PRESENTATION_REF,value:"CLOSED",operations:[
          {op:"REMOVE_ATTRIBUTE",targetRef:PLAYLISTS_OVERVIEW_REF,attribute:"hidden"},
          {op:"SET_ATTRIBUTE",targetRef:PLAYLIST_DETAIL_REF,attribute:"hidden",value:""}
        ]}
      ],interactionBindings:[]},
      elements:[
        {ref:FRAME_REF,kind:"FRAME",label:"Desktop shell frame",parentRef:null,symbol:"body[data-vex-screen]",declared:'data-vex-screen="'+SCREEN_REF+'"'},
        {ref:SHELL_REF,kind:"REGION",label:"V3-shaped V4 shell",parentRef:FRAME_REF,symbol:"main.shell",declared:'<main class="shell">'},
        {ref:SHELL_TERRAIN_REF,kind:"REGION",label:"Admitted Shell terrain selection",parentRef:SHELL_REF,symbol:"ShellNavigation.currentProjection.panel",declared:"bounded Home / Library / Discover / Playlists projection"},
        {ref:HOME_PANEL_REF,kind:"REGION",label:"Home terrain",parentRef:SHELL_TERRAIN_REF,symbol:'[data-shell-panel="home"]',declared:'data-shell-panel="home"'},
        {ref:LIBRARY_PANEL_REF,kind:"REGION",label:"Library terrain",parentRef:SHELL_TERRAIN_REF,symbol:'[data-shell-panel="library"]',declared:'data-shell-panel="library"'},
        {ref:DISCOVER_PANEL_REF,kind:"REGION",label:"Discover terrain",parentRef:SHELL_TERRAIN_REF,symbol:'[data-shell-panel="discover"]',declared:'data-shell-panel="discover"'},
        {ref:PLAYLISTS_PANEL_REF,kind:"REGION",label:"Playlists terrain",parentRef:SHELL_TERRAIN_REF,symbol:'[data-shell-panel="playlists"]',declared:'data-shell-panel="playlists"'},
        {ref:PLAYLISTS_OVERVIEW_REF,kind:"REGION",label:"Playlists overview",parentRef:PLAYLISTS_PANEL_REF,symbol:"#playlistOverview",declared:'id="playlistOverview"'},
        {ref:PLAYLIST_DETAIL_REF,kind:"REGION",label:"Selected Playlist detail",parentRef:PLAYLISTS_PANEL_REF,symbol:"#playlistDetail",declared:'id="playlistDetail"'},
        {ref:DISCOVER_MODE_REF,kind:"REGION",label:"Discover mode selection",parentRef:DISCOVER_PANEL_REF,symbol:"DiscoverNavigation.currentProjection.mode",declared:"This PC / YouTube / Explore selection"},
        {ref:DISCOVER_PC_PANEL_REF,kind:"REGION",label:"This PC mode",parentRef:DISCOVER_MODE_REF,symbol:'[data-discover-panel="pc"]',declared:'data-discover-panel="pc"'},
        {ref:DISCOVER_YOUTUBE_PANEL_REF,kind:"REGION",label:"YouTube mode",parentRef:DISCOVER_MODE_REF,symbol:'[data-discover-panel="youtube"]',declared:'data-discover-panel="youtube"'},
        {ref:DISCOVER_EXPLORE_PANEL_REF,kind:"REGION",label:"Explore mode",parentRef:DISCOVER_MODE_REF,symbol:'[data-discover-panel="explore"]',declared:'data-discover-panel="explore"'},
        {ref:LIBRARY_MODE_REF,kind:"REGION",label:"Library mode selection",parentRef:LIBRARY_PANEL_REF,symbol:"libraryView.mode",declared:"Songs / Albums / Artists selection"},
        {ref:LIBRARY_MODE_SONGS_REF,kind:"REGION",label:"Songs mode",parentRef:LIBRARY_MODE_REF,symbol:"songs",declared:"libraryView.mode = songs"},
        {ref:LIBRARY_MODE_ALBUMS_REF,kind:"REGION",label:"Albums mode",parentRef:LIBRARY_MODE_REF,symbol:"albums",declared:"libraryView.mode = albums"},
        {ref:LIBRARY_MODE_ARTISTS_REF,kind:"REGION",label:"Artists mode",parentRef:LIBRARY_MODE_REF,symbol:"artists",declared:"libraryView.mode = artists"},
        {ref:TRACK_CHOICES_REF,kind:"REGION",label:"Library results terrain",parentRef:LIBRARY_PANEL_REF,symbol:"#trackChoices",declared:'id="trackChoices"'},
        ...stableElements,
      ],
      layout:{kind:"FRAME",symbol:"VexStream desktop shell",source:source("web/index.html"),root:FRAME_REF,frames:[{ref:FRAME_REF,seams:[],regions:[{ref:SHELL_REF,start:"FRAME_START",end:"FRAME_END",top:"FRAME_TOP",bottom:"FRAME_BOTTOM"}],chrome:[]}],overlays:[]},
      realization:{
        status:"PROJECTED",
        entry:{symbol:"ShellNavigation + Library view + DiscoverNavigation + PlaylistNavigation",source:source("web/app.js")},
        rendersRegisteredTree:SCREEN_REF,
        realizers:["ShellNavigation","Library view projection","DiscoverNavigation","PlaylistNavigation"],
        presence:[
          {
            ref:SHELL_SELECTION_REF,
            kind:"SELECTION",
            subjectRef:SHELL_TERRAIN_REF,
            owner:"ShellNavigation",
            defaultRef:HOME_PANEL_REF,
            exhaustive:false,
            source:source("web/features/shell/navigation.js"),
            choices:[
              {condition:"projection.panel === 'home'",targetRef:HOME_PANEL_REF,realizedBy:"ShellNavigation.project",realizes:[]},
              {condition:"projection.panel === 'library'",targetRef:LIBRARY_PANEL_REF,realizedBy:"ShellNavigation.project",realizes:[LIBRARY_MODE_REF,TRACK_CHOICES_REF]},
              {condition:"projection.panel === 'discover'",targetRef:DISCOVER_PANEL_REF,realizedBy:"ShellNavigation.project",realizes:[DISCOVER_MODE_REF]},
              {condition:"projection.panel === 'playlists'",targetRef:PLAYLISTS_PANEL_REF,realizedBy:"ShellNavigation.project",realizes:[]}
            ]
          },
          {
            ref:LIBRARY_MODE_SELECTION_REF,
            kind:"SELECTION",
            subjectRef:LIBRARY_MODE_REF,
            owner:"ShellNavigation",
            defaultRef:LIBRARY_MODE_SONGS_REF,
            exhaustive:true,
            source:source("web/features/shell/navigation.js"),
            gate:{presenceRef:SHELL_SELECTION_REF,choice:1},
            choices:[
              {condition:"projection.libraryMode === 'songs'",targetRef:LIBRARY_MODE_SONGS_REF,realizedBy:"ShellNavigation.project",realizes:[TRACK_CHOICES_REF]},
              {condition:"projection.libraryMode === 'albums'",targetRef:LIBRARY_MODE_ALBUMS_REF,realizedBy:"ShellNavigation.project",realizes:[TRACK_CHOICES_REF]},
              {condition:"projection.libraryMode === 'artists'",targetRef:LIBRARY_MODE_ARTISTS_REF,realizedBy:"ShellNavigation.project",realizes:[TRACK_CHOICES_REF]}
            ]
          },
          {
            ref:DISCOVER_MODE_SELECTION_REF,
            kind:"SELECTION",
            subjectRef:DISCOVER_MODE_REF,
            owner:"DiscoverNavigation",
            defaultRef:DISCOVER_PC_PANEL_REF,
            exhaustive:true,
            source:source("web/features/discover/navigation.js"),
            gate:{presenceRef:SHELL_SELECTION_REF,choice:2},
            choices:[
              {condition:"projection.mode === '"+DiscoverMode.PC+"'",targetRef:DISCOVER_PC_PANEL_REF,realizedBy:"DiscoverNavigation.project",realizes:[]},
              {condition:"projection.mode === '"+DiscoverMode.YOUTUBE+"'",targetRef:DISCOVER_YOUTUBE_PANEL_REF,realizedBy:"DiscoverNavigation.project",realizes:[]},
              {condition:"projection.mode === '"+DiscoverMode.EXPLORE+"'",targetRef:DISCOVER_EXPLORE_PANEL_REF,realizedBy:"DiscoverNavigation.project",realizes:[]}
            ]
          },
          {
            ref:PLAYLIST_DETAIL_PRESENTATION_REF,
            kind:"PRESENTATION",
            subjectRef:PLAYLIST_DETAIL_REF,
            owner:"PlaylistNavigation",
            condition:"projection.openPlaylistID !== ''",
            realizedBy:"PlaylistNavigation.open/close",
            source:source("web/features/playlists/navigation.js"),
            gate:{presenceRef:SHELL_SELECTION_REF,choice:3},
            realizes:[
              elementRef(PlaylistNavigationIntent.CloseDetail),
              elementRef(PlaylistClientIntent.RenameName),
              elementRef(PlayerCompositionIntent.PlayPlaylist),
              elementRef(PlayerCompositionIntent.ShufflePlaylist),
              elementRef(PlaylistNavigationIntent.AddSongs),
              elementRef(PlaylistClientIntent.Rename),
              elementRef(PlaylistClientIntent.Delete)
            ]
          },
          {
            ref:PLAYLIST_CARD_POPULATION_REF,
            kind:"POPULATION",
            subjectRef:PLAYLISTS_OVERVIEW_REF,
            owner:"renderPlaylistChoices",
            list:{ref:"runtime.vexstream.playlists.cards",expression:"resolvedPlaylists",identity:"RUNTIME"},
            items:"resolvedPlaylists",
            itemKey:"playlist.id",
            contentType:"playlist-card",
            realizedBy:"renderPlaylistChoices",
            source:source("web/app.js"),
            templateRef:PLAYLIST_CARD_TEMPLATE_REF,
            gate:{presenceRef:SHELL_SELECTION_REF,choice:3},
            empty:[],
            emptyRealization:"SOURCE_AUTHORED"
          },
          {
            ref:PLAYLIST_ENTRY_POPULATION_REF,
            kind:"POPULATION",
            subjectRef:PLAYLIST_DETAIL_REF,
            owner:"renderPlaylistChoices",
            list:{ref:"runtime.vexstream.playlists.detail.entries",expression:"resolved.entries",identity:"RUNTIME"},
            items:"resolved.entries",
            itemKey:"playlist.id + entry.ref + occurrenceWithinPlaylist",
            contentType:"playlist-detail-entry",
            realizedBy:"playlistEntryHTML",
            source:source("web/app.js"),
            templateRef:PLAYLIST_ENTRY_TEMPLATE_REF,
            gate:{presenceRef:PLAYLIST_DETAIL_PRESENTATION_REF,choice:0},
            empty:[],
            emptyRealization:"SOURCE_AUTHORED"
          }
        ],
        templates:[
          {
            ref:PLAYLIST_CARD_TEMPLATE_REF,
            symbol:"playlist-card",
            source:source("web/app.js"),
            summary:"One durable Playlist rendered in the overview; identity survives ordering and wording changes.",
            instanceFormation:'dynamicVexPresentationRef("'+PLAYLIST_CARD_INSTANCE_BASE+'",[playlist.id])',
            instanceIdentity:{
              kind:"RUNTIME_INSTANCE",
              baseRef:PLAYLIST_CARD_INSTANCE_BASE,
              partSources:["playlist.id"],
              encoding:VEX_PRESENTATION_INSTANCE_ENCODING,
              continuity:VexPresentationContinuity.STABLE_INSTANCE
            },
            realizes:[
              elementRef(PlayerCompositionIntent.PlayPlaylist),
              elementRef(PlayerCompositionIntent.ShufflePlaylist)
            ],
            usedBy:[PLAYLIST_CARD_POPULATION_REF],
            slots:[
              {path:"card",kind:"CARD",primitive:"article.playlist-card",instanceFormation:"playlistCardRef",instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistCardRef",localPath:"card"},depth:0},
              {path:"actions.play",kind:"BUTTON",primitive:"button[data-playlist-action=play]",realizes:elementRef(PlayerCompositionIntent.PlayPlaylist),instanceFormation:'appendVexPresentationPath(playlistCardRef,"actions/play")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistCardRef",localPath:"actions/play"},depth:1},
              {path:"actions.shuffle",kind:"BUTTON",primitive:"button[data-playlist-action=shuffle]",realizes:elementRef(PlayerCompositionIntent.ShufflePlaylist),instanceFormation:'appendVexPresentationPath(playlistCardRef,"actions/shuffle")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistCardRef",localPath:"actions/shuffle"},depth:1},
              {path:"actions.open",kind:"BUTTON",primitive:"button[data-playlist-action=open]",instanceFormation:'appendVexPresentationPath(playlistCardRef,"actions/open")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistCardRef",localPath:"actions/open"},depth:1}
            ]
          },
          {
            ref:PLAYLIST_ENTRY_TEMPLATE_REF,
            symbol:"playlist-entry",
            source:source("web/app.js"),
            summary:"One current Playlist entry occurrence. Duplicate refs and reorder prevent a stronger durable entry claim without a source-owned entry ID.",
            instanceFormation:'dynamicVexPresentationRef("'+PLAYLIST_ENTRY_INSTANCE_BASE+'",[playlist.id,entry.ref,occurrenceWithinPlaylist])',
            instanceIdentity:{
              kind:"RUNTIME_INSTANCE",
              baseRef:PLAYLIST_ENTRY_INSTANCE_BASE,
              partSources:["playlist.id","entry.ref","occurrenceWithinPlaylist"],
              encoding:VEX_PRESENTATION_INSTANCE_ENCODING,
              continuity:VexPresentationContinuity.CURRENT_OCCURRENCE
            },
            realizes:[
              elementRef("vexstream.player.add-queue"),
              elementRef("vexstream.player.play-next"),
              elementRef("vexstream.explore.from-track"),
              elementRef("vexstream.track.action.edit"),
              elementRef("vexstream.track.action.trash")
            ],
            usedBy:[PLAYLIST_ENTRY_POPULATION_REF],
            slots:[
              {path:"entry",kind:"ROW",primitive:"li.playlist-entry",instanceFormation:"playlistEntryRef",instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistEntryRef",localPath:"entry"},depth:0},
              {path:"actions.play",kind:"BUTTON",primitive:"button[data-track-action=play]",instanceFormation:'appendVexPresentationPath(playlistEntryRef,"actions/play")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistEntryRef",localPath:"actions/play"},depth:1},
              {path:"actions.add-queue",kind:"BUTTON",primitive:"button[data-track-action=append]",realizes:elementRef("vexstream.player.add-queue"),instanceFormation:'appendVexPresentationPath(playlistEntryRef,"actions/add-queue")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistEntryRef",localPath:"actions/add-queue"},depth:1},
              {path:"actions.play-next",kind:"BUTTON",primitive:"button[data-track-action=play-next]",realizes:elementRef("vexstream.player.play-next"),instanceFormation:'appendVexPresentationPath(playlistEntryRef,"actions/play-next")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistEntryRef",localPath:"actions/play-next"},depth:1},
              {path:"actions.explore",kind:"BUTTON",primitive:"button[data-track-action=explore]",realizes:elementRef("vexstream.explore.from-track"),instanceFormation:'appendVexPresentationPath(playlistEntryRef,"actions/explore")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistEntryRef",localPath:"actions/explore"},depth:1},
              {path:"actions.edit",kind:"BUTTON",primitive:"button[data-track-action=edit]",realizes:elementRef("vexstream.track.action.edit"),instanceFormation:'appendVexPresentationPath(playlistEntryRef,"actions/edit")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistEntryRef",localPath:"actions/edit"},depth:1},
              {path:"actions.trash",kind:"BUTTON",primitive:"button[data-track-action=trash]",realizes:elementRef("vexstream.track.action.trash"),instanceFormation:'appendVexPresentationPath(playlistEntryRef,"actions/trash")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistEntryRef",localPath:"actions/trash"},depth:1},
              {path:"actions.up",kind:"BUTTON",primitive:"button[data-playlist-entry-action=up]",instanceFormation:'appendVexPresentationPath(playlistEntryRef,"actions/up")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistEntryRef",localPath:"actions/up"},depth:1},
              {path:"actions.down",kind:"BUTTON",primitive:"button[data-playlist-entry-action=down]",instanceFormation:'appendVexPresentationPath(playlistEntryRef,"actions/down")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistEntryRef",localPath:"actions/down"},depth:1},
              {path:"actions.remove",kind:"BUTTON",primitive:"button[data-playlist-entry-action=remove]",instanceFormation:'appendVexPresentationPath(playlistEntryRef,"actions/remove")',instanceIdentity:{kind:"DERIVED_CHILD",parentSource:"playlistEntryRef",localPath:"actions/remove"},depth:1}
            ]
          }
        ],elements:[],opaque:[],held:[
          "Deleted Shell terrain remains outside the current expression slice.",
          "Current runtime Playlist instances remain unobserved; their source-owned identity formation is projected as POPULATION + template relationships without enumerating live IDs.",
          "Playlist entry identity is CURRENT_OCCURRENCE because the Product does not yet own a durable per-entry ID across arbitrary duplicate-ref reorder/rebuild.",
          "Other catalog rows/groups remain runtime data populations outside this bounded dynamic-identity slice."
        ]
      },
    }],
    pathways:[
      previousPathway(),phoneConnectionPathway(),systemStatusPathway(),playbackPathway(),
      playlistDetailClosePathway(),playlistPlayPathway(),playlistShufflePathway(),playlistAddSongsPathway(),playlistRenamePathway(),playlistDeletePathway(),
      queuePathway("next",QueueControl.Next,QueueAct.Next.ref,QueueEffect.Advance.ref+" / "+QueueEffect.Wrap.ref,"Player.next()"),
      queuePathway("shuffle",QueueControl.Shuffle,QueueAct.EnableShuffle.ref+" / "+QueueAct.DisableShuffle.ref,QueueEffect.ShuffleUpcoming.ref+" / "+QueueEffect.DisableShuffle.ref,"Player.toggleShuffle()"),
      queuePathway("repeat",QueueControl.Repeat,QueueAct.EnableRepeat.ref+" / "+QueueAct.DisableRepeat.ref,QueueEffect.EnableRepeat.ref+" / "+QueueEffect.DisableRepeat.ref,"Player.toggleRepeat()"),
    ],
    annotations:[],examples:[],
    examplesHeld:[{ref:"examples.vexstream-music.dynamic-shell",source:source("tests/player-browser-regression.py"),reason:"Home, Library modes, Discover modes, Playlists overview/create, selected-Playlist detail, and Playlist runtime identity templates are projected. Current live Playlist instances are never claimed; nested Library/Import states, other dynamic catalog populations, and Deleted remain separately admitted."}],
  };
}

export function readCurrentSources(){
  return {html:fs.readFileSync(path.join(V2,"web","index.html"),"utf8"),css:fs.readFileSync(path.join(V2,"web","styles.css"),"utf8")};
}
export const stableJSON=value=>JSON.stringify(value,null,2)+"\n";

if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
  const output=stableJSON(buildVexStreamVexBoardLens(readCurrentSources()));
  if(process.argv.includes("--stdout"))process.stdout.write(output);
  else fs.writeFileSync(path.join(HERE,"vexboard-screen-lens.json"),output);
}
