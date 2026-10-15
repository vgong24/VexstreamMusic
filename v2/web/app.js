// [VXG RealForever]
import { PlaybackControl } from "./features/playback/behavior.js";
import { createHTMLMediaPlaybackConnection } from "./features/playback/connection.js";
import { QueueControl, QueueState, currentQueueRef } from "./features/queue/behavior.js";
import { createQueueConnection } from "./features/queue/connection.js";
import { createPlayerComposition } from "./features/player/composition.js";
import { playerJourney } from "./features/player/journey.js";
import { loadLibraryCatalog } from "./features/library/client.js";
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
const title=document.querySelector("#trackTitle");
const artist=document.querySelector("#trackArtist");
const error=document.querySelector("#error");

let catalog=Object.freeze([]);
let byRef=new Map();
let refs=[];

function trackTitle(item){
 return item?.title?.trim()||resolveText(locale,"library.unknown-title");
}

function trackArtist(item){
 return item?.artist?.trim()||resolveText(locale,"library.unknown-artist");
}

function randomShuffle(values){
 const result=[...values];
 for(let i=result.length-1;i>0;i--){const j=Math.floor(Math.random()*(i+1));[result[i],result[j]]=[result[j],result[i]];}
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

function setControl(button,presentation){
 button.disabled=!presentation.enabled;
 button.dataset.textRef=presentation.textRef;
 button.dataset.actRef=presentation.act?.ref??"";
 button.textContent=resolveText(locale,presentation.textRef);
 if("selected" in presentation)button.setAttribute("aria-pressed",String(Boolean(presentation.selected)));
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
 status.textContent=playback.currentProjection.state.requestPending?resolveText(locale,"playback.status.pending"):resolveText(locale,"playback.status.ready");

 const current=currentQueueRef(qp.state);
 if(!current){
  title.textContent=resolveText(locale,"player.nothing.title");
  artist.textContent=resolveText(locale,"player.nothing.artist");
 }
 queueSummary.textContent=qp.state.kind==="empty"?"Empty":`${qp.state.items.length} items · ${qp.state.currentIndex+1} current`;
 queueList.innerHTML=qp.state.kind==="empty"?"":qp.state.items.map((ref,index)=>{
  const item=byRef.get(ref);
  const isCurrent=index===qp.state.currentIndex;
  const currentClass=isCurrent?"current":"";
  const stateLabel=index<qp.state.currentIndex?"Played":isCurrent?"Current":"Up next";
  const upDisabled=isCurrent||index===0;
  const downDisabled=isCurrent||index===qp.state.items.length-1;
  return `<li class="${currentClass}" data-queue-ref="${ref}" data-queue-index="${index}">
    <span class="queue-index">${index+1}</span>
    <span class="queue-title"><strong>${trackTitle(item)}</strong><span class="queue-state">${stateLabel}</span></span>
    <span class="queue-state">${isCurrent?"●":" "}</span>
    <span class="queue-actions">
      <button data-queue-action="play" ${isCurrent?"disabled":""}>Play</button>
      <button data-queue-action="up" ${upDisabled?"disabled":""}>↑</button>
      <button data-queue-action="down" ${downDisabled?"disabled":""}>↓</button>
      <button data-queue-action="remove" ${isCurrent?"disabled":""}>Remove</button>
    </span>
  </li>`;
 }).join("");
}

function renderTrackChoices(){
 trackChoices.innerHTML=catalog.map(item=>`<div class="track-card" data-track-ref="${item.ref}">
  <strong>${trackTitle(item)}</strong>
  <small>${trackArtist(item)}</small>
  <span class="track-actions">
    <button data-track-action="play" data-vex-action="vexstream.player.play-track">Play</button>
    <button data-track-action="play-next" data-vex-action="vexstream.player.play-next">Play next</button>
    <button data-track-action="append" data-vex-action="vexstream.player.add-queue">Add to queue</button>
  </span>
</div>`).join("");
}

async function guarded(effect){
 error.textContent="";
 try{await effect()}catch(failure){error.textContent=failure?.message??String(failure)}
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
 if(action.dataset.trackAction==="play"){void guarded(()=>player.playTrack(ref,refs));return;}
 if(action.dataset.trackAction==="play-next"){void guarded(()=>player.playNext([ref]));return;}
 if(action.dataset.trackAction==="append"){player.append([ref]);render();}
});

queueList.addEventListener("click",event=>{
 const action=event.target.closest("[data-queue-action]");
 const row=event.target.closest("[data-queue-index]");
 if(!action||!row)return;
 const index=Number(row.dataset.queueIndex);
 if(action.dataset.queueAction==="play"){void guarded(()=>player.playQueueIndex(index));return;}
 if(action.dataset.queueAction==="up"){player.moveQueue(index,index-1);render();return;}
 if(action.dataset.queueAction==="down"){player.moveQueue(index,index+2);render();return;}
 if(action.dataset.queueAction==="remove"){player.removeQueue(index);render();}
});

queue.subscribe(render);
playback.subscribe(render);
render();

async function initialize(){
 catalog=await loadLibraryCatalog();
 byRef=new Map(catalog.map(item=>[item.ref,item]));
 refs=catalog.map(item=>item.ref);
 renderTrackChoices();
 render();
 window.vexV4=Object.freeze({
  ready:true,
  journey:playerJourney,
  queue,
  playback,
  player,
  media:audio,
  catalog,
  controls:Object.freeze({PlaybackControl,QueueControl})
 });
}

void initialize().catch(failure=>{
 error.textContent=failure?.message??String(failure);
});
