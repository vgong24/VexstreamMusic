// [VXG RealForever]
import { PlaybackControl } from "./features/playback/behavior.js";
import { createHTMLMediaPlaybackConnection } from "./features/playback/connection.js";
import { playbackJourney } from "./features/playback/journey.js";
import { resolveText } from "./foundation/text.js";
import { en } from "./locales/en.js";

const audio=document.querySelector("#audio");
const playPause=document.querySelector("#playPause");
const status=document.querySelector("#status");
const semanticState=document.querySelector("#semanticState");
const title=document.querySelector("#trackTitle");
const artist=document.querySelector("#trackArtist");
const error=document.querySelector("#error");
const locale=en;

const currentMedia=Object.freeze({
 ref:"media.vexstream.test-tone",
 url:window.__VEXSTREAM_V4_MEDIA_URL??"/media/vexstream-test-tone.mp3",
 titleRef:"playback.track.title",
 artistRef:"playback.track.artist"
});

audio.src=currentMedia.url;
audio.dataset.mediaRef=currentMedia.ref;
title.textContent=resolveText(locale,currentMedia.titleRef);
artist.textContent=resolveText(locale,currentMedia.artistRef);

const connection=createHTMLMediaPlaybackConnection({
 media:audio,
 hasCurrentMedia:()=>Boolean(audio.dataset.mediaRef&&audio.currentSrc||audio.getAttribute("src")),
 restartCurrent:async media=>{media.currentTime=0;}
});

function render(projection){
 const control=projection.control;
 playPause.disabled=!control.enabled;
 playPause.dataset.textRef=control.textRef;
 playPause.dataset.actRef=control.act?.ref??"";
 playPause.textContent=resolveText(locale,control.textRef);
 semanticState.textContent=projection.state.kind;
 status.textContent=projection.state.requestPending?resolveText(locale,"playback.status.pending"):resolveText(locale,"playback.status.ready");
}

connection.subscribe(render);

playPause.addEventListener("click",async()=>{
 error.textContent="";
 try{await connection.activate(PlaybackControl.PlayPause)}catch(failure){error.textContent=failure?.message??String(failure)}
});

window.vexV4=Object.freeze({
 ready:true,
 journey:playbackJourney,
 connection,
 media:audio,
 PlaybackControl,
 currentMedia
});
