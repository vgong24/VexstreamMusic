// [VXG RealForever]
// Human location inside the recognizable V3 Discover terrain.
//
// DiscoverNavigation owns only which human route is visible. Source-folder
// configuration, Import, Explore, provider runtime, and Library truth remain
// with their existing owners.

export const DiscoverMode=Object.freeze({
 PC:"pc",
 YOUTUBE:"youtube",
 EXPLORE:"explore"
});

const MODES=new Set(Object.values(DiscoverMode));

export function createDiscoverNavigation({
 initialMode=DiscoverMode.PC
}={}){
 let mode=resolveMode(initialMode);
 const listeners=new Set();

 function projection(){
  return Object.freeze({mode});
 }

 function publish(){
  const current=projection();
  for(const listener of listeners)listener(current);
  return current;
 }

 return Object.freeze({
  get currentProjection(){return projection();},
  select(next){
   const resolved=resolveMode(next);
   if(resolved===mode)return projection();
   mode=resolved;
   return publish();
  },
  subscribe(listener){
   listeners.add(listener);
   return ()=>listeners.delete(listener);
  }
 });
}

function resolveMode(value){
 const mode=String(value??"").trim();
 if(!MODES.has(mode)){
  throw new Error(`Unknown VexStream Discover mode: ${mode||"(empty)"}`);
 }
 return mode;
}
