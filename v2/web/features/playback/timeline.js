// [VXG RealForever]
// Playback Timeline answers one provider question:
//
//   where is the current HTMLMediaElement inside the admitted playback window,
//   and what provider position does a human seek request mean?
//
// Library owns the playback-window facts. Queue owns what comes next. This connection owns neither.

const WINDOW_END_TOLERANCE_SECONDS=0.015;

export function projectPlaybackTimeline(media,window){
 const bounds=resolvePlaybackBounds(window,media?.duration);
 if(!bounds){
  return Object.freeze({
   kind:"unavailable",
   seekAvailable:false,
   startSeconds:0,
   endSeconds:0,
   elapsedSeconds:0,
   durationSeconds:0,
   normalized:0,
   windowEndReached:false
  });
 }

 const providerSeconds=finiteNumber(media?.currentTime,bounds.startSeconds);
 const clamped=Math.max(
  bounds.startSeconds,
  Math.min(bounds.endSeconds,providerSeconds)
 );
 const elapsedSeconds=clamped-bounds.startSeconds;
 const normalized=bounds.durationSeconds>0
  ?elapsedSeconds/bounds.durationSeconds
  :0;

 return Object.freeze({
  kind:"ready",
  seekAvailable:bounds.durationSeconds>0,
  startSeconds:bounds.startSeconds,
  endSeconds:bounds.endSeconds,
  elapsedSeconds,
  durationSeconds:bounds.durationSeconds,
  normalized:Math.max(0,Math.min(1,normalized)),
  windowEndReached:
   bounds.hasExplicitEnd&&
   providerSeconds>=bounds.endSeconds-WINDOW_END_TOLERANCE_SECONDS
 });
}

export function createHTMLMediaPlaybackTimelineConnection({
 media,
 currentWindow
}={}){
 if(!media?.addEventListener||!media?.removeEventListener){
  throw new Error("Playback Timeline requires an HTMLMediaElement-like provider");
 }
 if(typeof currentWindow!=="function"){
  throw new Error("Playback Timeline requires currentWindow()");
 }

 const listeners=new Set();
 let lastProjection=null;

 function formProjection(){
  return projectPlaybackTimeline(media,currentWindow());
 }

 function publish(){
  const projection=formProjection();
  lastProjection=projection;
  for(const listener of listeners)listener(projection);
  return projection;
 }

 function seekNormalized(value){
  const projection=formProjection();
  if(projection.kind!=="ready"||!projection.seekAvailable){
   publish();
   return null;
  }
  const normalized=Math.max(0,Math.min(1,finiteNumber(value,0)));
  media.currentTime=
   projection.startSeconds+
   normalized*projection.durationSeconds;
  return publish();
 }

 const providerEvents=[
  "loadedmetadata",
  "durationchange",
  "timeupdate",
  "seeked",
  "emptied"
 ];
 const onProviderEvent=()=>publish();
 for(const event of providerEvents){
  media.addEventListener(event,onProviderEvent);
 }
 publish();

 return Object.freeze({
  get currentProjection(){
   return lastProjection??publish();
  },
  refresh:publish,
  seekNormalized,
  subscribe(listener){
   if(typeof listener!=="function"){
    throw new Error("Playback Timeline subscriber must be a function");
   }
   listeners.add(listener);
   listener(lastProjection??publish());
   return ()=>listeners.delete(listener);
  },
  close(){
   for(const event of providerEvents){
    media.removeEventListener(event,onProviderEvent);
   }
   listeners.clear();
  }
 });
}

function resolvePlaybackBounds(window,mediaDuration){
 if(!window)return null;
 const duration=finitePositive(mediaDuration);
 if(duration===null)return null;

 const start=Math.min(
  duration,
  Math.max(0,finiteNumber(window.startSeconds,0))
 );
 const requestedEnd=finitePositive(window.endSeconds);
 const end=requestedEnd===null
  ?duration
  :Math.min(duration,Math.max(start,requestedEnd));
 const length=end-start;
 if(!(length>0))return null;

 return Object.freeze({
  startSeconds:start,
  endSeconds:end,
  durationSeconds:length,
  hasExplicitEnd:
   requestedEnd!==null&&
   requestedEnd<duration-WINDOW_END_TOLERANCE_SECONDS
 });
}

function finiteNumber(value,fallback){
 const number=Number(value);
 return Number.isFinite(number)?number:fallback;
}

function finitePositive(value){
 const number=Number(value);
 return Number.isFinite(number)&&number>0?number:null;
}
