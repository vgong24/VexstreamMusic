// [VXG RealForever]
// Browser client for explicit local Trash custody.

export async function loadTrash(
  {request=globalThis.fetch}={}
){
  const payload=await requestJSON(
    "/api/library/trash",
    {method:"GET",headers:{Accept:"application/json"}},
    request,
    "Trash"
  );
  return Object.freeze({
    trash:Object.freeze(
      Array.isArray(payload?.trash)
        ? payload.trash.map(projectTrashItem)
        : []
    ),
    deletedHistory:Object.freeze(
      Array.isArray(payload?.deletedHistory)
        ? payload.deletedHistory.map(projectDeletedItem)
        : []
    ),
    pendingPurge:
      payload?.pendingPurge&&typeof payload.pendingPurge==="object"
        ? Object.freeze({...payload.pendingPurge})
        : null
  });
}

export async function moveTracksToTrash(ids,{request=globalThis.fetch}={}){
  return mutate(
    "/api/library/trash",
    {
      ids:refs(ids),
      confirm:"MOVE_TO_VEXSTREAM_TRASH"
    },
    request,
    "Move to Trash"
  );
}

export async function restoreTrashItems(ids,{request=globalThis.fetch}={}){
  return mutate(
    "/api/library/trash/restore",
    {ids:refs(ids)},
    request,
    "Restore from Trash"
  );
}

export async function purgeTrashItems(
  ids,
  {
    preserveMetadata=true,
    request=globalThis.fetch
  }={}
){
  return mutate(
    "/api/library/trash/purge",
    {
      ids:refs(ids),
      confirm:"PERMANENTLY_DELETE_TRASH",
      preserveMetadata:Boolean(preserveMetadata)
    },
    request,
    "Permanently delete Trash"
  );
}

export function recoverTrashPurge(
  {request=globalThis.fetch}={}
){
  return mutate(
    "/api/library/trash/purge/recover",
    {confirm:"RECOVER_ADMITTED_TRASH_PURGE"},
    request,
    "Recover Trash purge"
  );
}

async function mutate(path,body,request,label){
  return requestJSON(
    path,
    {
      method:"POST",
      headers:{
        Accept:"application/json",
        "Content-Type":"application/json"
      },
      body:JSON.stringify(body)
    },
    request,
    label
  );
}

async function requestJSON(path,options,request,label){
  if(typeof request!=="function"){
    throw new Error(label+" requires fetch-like request function");
  }
  const response=await request(path,options);
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

function refs(values){
  const seen=new Set();
  const out=[];
  for(const value of Array.isArray(values)?values:[values]){
    const ref=String(value??"").trim();
    if(!ref||seen.has(ref))continue;
    seen.add(ref);
    out.push(ref);
  }
  if(!out.length)throw new Error("Trash action requires track refs");
  return out;
}

function projectTrashItem(value){
  const id=String(value?.id??"").trim();
  if(!id)throw new Error("Trash item is missing id");
  return Object.freeze({
    id,
    track:Object.freeze({...value.track}),
    trashedAt:String(value?.trashedAt??"")
  });
}

function projectDeletedItem(value){
  const id=String(value?.id??"").trim();
  if(!id)throw new Error("Deleted history item is missing id");
  return Object.freeze({
    id,
    track:Object.freeze({...value.track}),
    deletedAt:String(value?.deletedAt??"")
  });
}
