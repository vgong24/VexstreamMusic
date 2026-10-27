// [VXG RealForever]
// Ephemeral multi-track selection for the recognizable Songs terrain.
//
// Selection answers only "which visible Library refs did the human select?".
// Library truth, Playback, Queue, Playlists, Trash, and metadata mutation remain
// with their existing owners.

export function createTrackSelection({active=false}={}){
 let enabled=Boolean(active);
 let selected=[];
 const listeners=new Set();

 function projection(){
  return Object.freeze({
   active:enabled,
   count:selected.length,
   selectedRefs:Object.freeze([...selected])
  });
 }

 function publish(){
  const current=projection();
  for(const listener of listeners)listener(current);
  return current;
 }

 function normalize(refs){
  const seen=new Set();
  const out=[];
  for(const value of Array.isArray(refs)?refs:[]){
   const ref=String(value??"").trim();
   if(!ref||seen.has(ref))continue;
   seen.add(ref);
   out.push(ref);
  }
  return out;
 }

 return Object.freeze({
  get currentProjection(){return projection();},
  isSelected(ref){
   return selected.includes(String(ref??"").trim());
  },
  activate(){
   if(enabled)return projection();
   enabled=true;
   return publish();
  },
  deactivate(){
   if(!enabled&&!selected.length)return projection();
   enabled=false;
   selected=[];
   return publish();
  },
  toggleMode(){
   return enabled?this.deactivate():this.activate();
  },
  toggle(ref,wanted){
   if(!enabled)throw new Error("Track selection is not active");
   const value=String(ref??"").trim();
   if(!value)throw new Error("Track selection ref is required");
   const next=new Set(selected);
   const select=wanted===undefined?!next.has(value):Boolean(wanted);
   if(select)next.add(value);else next.delete(value);
   selected=[...next];
   return publish();
  },
  selectVisible(refs){
   if(!enabled)throw new Error("Track selection is not active");
   const next=new Set(selected);
   for(const ref of normalize(refs))next.add(ref);
   selected=[...next];
   return publish();
  },
  clear(){
   if(!selected.length)return projection();
   selected=[];
   return publish();
  },
  reconcile(availableRefs){
   const available=new Set(normalize(availableRefs));
   const next=selected.filter(ref=>available.has(ref));
   if(next.length===selected.length&&next.every((ref,index)=>ref===selected[index])){
    return projection();
   }
   selected=next;
   return publish();
  },
  subscribe(listener){
   listeners.add(listener);
   return ()=>listeners.delete(listener);
  }
 });
}
