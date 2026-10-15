// [VXG RealForever]
// Browser consumer for durable desktop Playlist truth.
//
// Playlist owns durable ordered references. Library owns whether a ref resolves today.
// Queue owns the playable listening context formed from those truths right now.

export function projectPlaylists(payload) {
  if (!payload || !Array.isArray(payload.playlists)) {
    throw new Error("playlist response requires playlists[]");
  }

  return Object.freeze(
    payload.playlists.map((raw, index) => {
      const id = String(raw?.id ?? "").trim();
      const name = String(raw?.name ?? "").trim();
      if (!id) throw new Error(`playlist ${index} is missing id`);
      if (!name) throw new Error(`playlist ${index} is missing name`);
      if (!Array.isArray(raw.trackIds)) {
        throw new Error(`playlist ${index} requires trackIds[]`);
      }

      const trackRefs = Object.freeze(
        raw.trackIds.map((value, refIndex) => {
          const ref = String(value ?? "").trim();
          if (!ref) {
            throw new Error(`playlist ${index} trackIds[${refIndex}] is empty`);
          }
          return ref;
        }),
      );

      return Object.freeze({
        id,
        name,
        trackRefs,
        createdAt: stringValue(raw.createdAt),
        updatedAt: stringValue(raw.updatedAt),
      });
    }),
  );
}

export function resolvePlaylistAvailability(playlist, catalog) {
  const byRef = new Map(catalog.map(track => [track.ref, track]));
  const entries = Object.freeze(
    playlist.trackRefs.map(ref => {
      const track = byRef.get(ref) ?? null;
      return Object.freeze({
        ref,
        available: track !== null,
        track,
      });
    }),
  );

  return Object.freeze({
    playlist,
    entries,
    playableRefs: Object.freeze(
      entries.filter(entry => entry.available).map(entry => entry.ref),
    ),
    missingRefs: Object.freeze(
      entries.filter(entry => !entry.available).map(entry => entry.ref),
    ),
  });
}

export async function loadPlaylists({ request = globalThis.fetch } = {}) {
  if (typeof request !== "function") {
    throw new Error("playlists require fetch-like request function");
  }
  const response = await request("/api/playlists", {
    method: "GET",
    headers: { Accept: "application/json" },
  });
  if (!response?.ok) {
    throw new Error(`playlist request failed: ${response?.status ?? "unknown"}`);
  }
  return projectPlaylists(await response.json());
}

export async function createPlaylist(
  name,
  trackRefs=[],
  {request=globalThis.fetch}={}
) {
  const payload=await playlistMutation(
    "/api/playlists/create",
    {
      name:String(name??"").trim(),
      trackIds:normalizeRefs(trackRefs)
    },
    request,
    "Create playlist"
  );
  return projectOne(payload?.playlist);
}

export async function addTracksToPlaylist(
  id,
  trackRefs,
  {request=globalThis.fetch}={}
) {
  const payload=await playlistMutation(
    "/api/playlists/add",
    {
      id:requiredID(id),
      trackIds:normalizeRefs(trackRefs)
    },
    request,
    "Add to playlist"
  );
  return Object.freeze({
    playlist:projectOne(payload?.playlist),
    added:Number(payload?.added)||0
  });
}

export async function reorderPlaylist(
  id,
  trackRefs,
  {request=globalThis.fetch}={}
) {
  await playlistMutation(
    "/api/playlists/order",
    {
      id:requiredID(id),
      trackIds:normalizeRefs(trackRefs,{allowEmpty:true})
    },
    request,
    "Reorder playlist"
  );
  return true;
}

export async function removeTracksFromPlaylist(
  id,
  trackRefs,
  {request=globalThis.fetch}={}
) {
  const payload=await playlistMutation(
    "/api/playlists/remove",
    {
      id:requiredID(id),
      trackIds:normalizeRefs(trackRefs)
    },
    request,
    "Remove from playlist"
  );
  return projectOne(payload?.playlist);
}

export async function renamePlaylist(
  id,
  name,
  {request=globalThis.fetch}={}
) {
  const payload=await playlistMutation(
    "/api/playlists/rename",
    {
      id:requiredID(id),
      name:String(name??"").trim()
    },
    request,
    "Rename playlist"
  );
  return projectOne(payload?.playlist);
}

export async function deletePlaylist(
  id,
  {request=globalThis.fetch}={}
) {
  await playlistMutation(
    "/api/playlists/delete",
    {
      id:requiredID(id),
      confirm:"DELETE_PLAYLIST_ONLY"
    },
    request,
    "Delete playlist"
  );
  return true;
}

async function playlistMutation(path,body,request,label){
  if(typeof request!=="function"){
    throw new Error(label+" requires fetch-like request function");
  }
  const response=await request(path,{
    method:"POST",
    headers:{
      Accept:"application/json",
      "Content-Type":"application/json"
    },
    body:JSON.stringify(body)
  });
  if(!response?.ok){
    let detail="";
    try{
      const payload=await response.json();
      detail=String(payload?.error??"").trim();
    }catch{}
    throw new Error(
      detail||`${label} request failed: ${response?.status??"unknown"}`
    );
  }
  return response.json();
}

function projectOne(value){
  return projectPlaylists({playlists:[value]})[0];
}

function requiredID(value){
  const id=String(value??"").trim();
  if(!id)throw new Error("playlist id required");
  return id;
}

function normalizeRefs(values,{allowEmpty=false}={}){
  const refs=[];
  for(const value of Array.isArray(values)?values:[values]){
    const ref=String(value??"").trim();
    if(ref)refs.push(ref);
  }
  if(!allowEmpty&&!refs.length){
    throw new Error("playlist track refs required");
  }
  return refs;
}

function stringValue(value) {
  return typeof value === "string" ? value : "";
}
