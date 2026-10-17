// [VXG RealForever]
// Playback Volume owns one provider question:
//
//   what output level does the HTMLMediaElement currently expose,
//   and what provider level does a human volume request mean?
//
// It does not own Playback play/pause, Queue, Library, or persistence.

export function projectPlaybackVolume(media){
 const volume=clampVolume(media?.volume);
 return Object.freeze({
  volume,
  percent:Math.round(volume*100)
 });
}

export function createHTMLMediaPlaybackVolumeConnection({
 media,
 initialVolume=null
}={}){
 if(!media?.addEventListener||!media?.removeEventListener){
  throw new Error("Playback Volume requires an HTMLMediaElement-like provider");
 }

 const listeners=new Set();
 let lastProjection=null;

 function formProjection(){
  return projectPlaybackVolume(media);
 }

 function publish(){
  const projection=formProjection();
  lastProjection=projection;
  for(const listener of listeners)listener(projection);
  return projection;
 }

 function setVolume(value){
  const requested=Number(value);
  if(!Number.isFinite(requested)){
   throw new Error("Playback Volume requires a finite level");
  }
  media.volume=clampVolume(requested);
  return publish();
 }

 const onProviderVolume=()=>publish();
 media.addEventListener("volumechange",onProviderVolume);

 if(initialVolume!==null&&initialVolume!==undefined){
  setVolume(initialVolume);
 }else{
  publish();
 }

 return Object.freeze({
  get currentProjection(){
   return lastProjection??publish();
  },
  setVolume,
  refresh:publish,
  subscribe(listener){
   if(typeof listener!=="function"){
    throw new Error("Playback Volume subscriber must be a function");
   }
   listeners.add(listener);
   listener(lastProjection??publish());
   return ()=>listeners.delete(listener);
  },
  close(){
   media.removeEventListener("volumechange",onProviderVolume);
   listeners.clear();
  }
 });
}

function clampVolume(value){
 const number=Number(value);
 if(!Number.isFinite(number))return 1;
 return Math.max(0,Math.min(1,number));
}
