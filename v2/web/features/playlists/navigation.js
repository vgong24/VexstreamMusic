// [VXG RealForever]
// Human location inside the recognizable Playlists terrain.
//
// PlaylistNavigation owns only whether the human is browsing all playlists or
// one durable playlist. Playlist data, availability, mutations, Playback,
// Queue, and Library truth remain with their existing owners.

export function createPlaylistNavigation({
 initialPlaylistID=""
}={}){
 let openPlaylistID=optionalID(initialPlaylistID);
 const listeners=new Set();

 function projection(){
  return Object.freeze({openPlaylistID});
 }

 function publish(){
  const current=projection();
  for(const listener of listeners)listener(current);
  return current;
 }

 return Object.freeze({
  get currentProjection(){return projection();},
  open(id){
   const next=requiredID(id);
   if(next===openPlaylistID)return projection();
   openPlaylistID=next;
   return publish();
  },
  close(){
   if(!openPlaylistID)return projection();
   openPlaylistID="";
   return publish();
  },
  reconcile(ids){
   if(!openPlaylistID)return projection();
   const available=new Set(
    (Array.isArray(ids)?ids:[])
     .map(optionalID)
     .filter(Boolean)
   );
   if(available.has(openPlaylistID))return projection();
   openPlaylistID="";
   return publish();
  },
  subscribe(listener){
   listeners.add(listener);
   return ()=>listeners.delete(listener);
  }
 });
}

function optionalID(value){
 return String(value??"").trim();
}

function requiredID(value){
 const id=optionalID(value);
 if(!id)throw new Error("Playlist navigation requires id");
 return id;
}
