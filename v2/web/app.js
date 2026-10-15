// [VXG RealForever]
import { PlaybackControl } from "./features/playback/behavior.js";
import { createHTMLMediaPlaybackConnection } from "./features/playback/connection.js";
import { QueueControl, QueueState, currentQueueRef } from "./features/queue/behavior.js";
import { createQueueConnection } from "./features/queue/connection.js";
import { createPlayerComposition } from "./features/player/composition.js";
import { playerJourney } from "./features/player/journey.js";
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
const trackChoices=document.querySelector("#trackChoices");
const title=document.querySelector("#trackTitle");
const artist=document.querySelector("#trackArtist");
const error=document.querySelector("#error");

const sharedMediaURL=window.__VEXSTREAM_V4_MEDIA_URL??"/media/vexstream-test-tone.mp3";
const catalog=Object.freeze([
 Object.freeze({ref:"media.vexstream.test-tone.1",url:sharedMediaURL,titleRef:"player.track.1.title",artistRef:"player.track.artist",playbackStartSeconds:0}),
 Object.freeze({ref:"media.vexstream.test-tone.2",url:sharedMediaURL,titleRef:"player.track.2.title",artistRef:"player.track.artist",playbackStartSeconds:0}),
 Object.freeze({ref:"media.vexstream.test-tone.3",url:sharedMediaURL,titleRef:"player.track.3.title",artistRef:"player.track.artist",playbackStartSeconds:0})
]);
const byRef=new Map(catalog.map(item=>[item.ref,item]));
const refs=catalog.map(item=>item.ref);

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
  title.textContent=resolveText(locale,item.titleRef);
  artist.textContent=resolveText(locale,item.artistRef);
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
  const currentClass=index===qp.state.currentIndex?"current":"";
  const stateLabel=index<qp.state.currentIndex?"Played":index===qp.state.currentIndex?"Current":"Up next";
  return `<li class="${currentClass}" data-queue-ref="${ref}"><span class="queue-index">${index+1}</span><strong>${resolveText(locale,item.titleRef)}</strong><span class="queue-state">${stateLabel}</span></li>`;
 }).join("");
}

trackChoices.innerHTML=catalog.map(item=>`<button class="track-choice" data-track-ref="${item.ref}"><strong>${resolveText(locale,item.titleRef)}</strong><small>${resolveText(locale,item.artistRef)}</small></button>`).join("");

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
trackChoices.addEventListener("click",event=>{
 const button=event.target.closest("[data-track-ref]");
 if(!button)return;
 void guarded(()=>player.playTrack(button.dataset.trackRef,refs));
});

queue.subscribe(render);
playback.subscribe(render);
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
