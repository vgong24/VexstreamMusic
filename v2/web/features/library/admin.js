// [VXG RealForever]
// Browser client for reversible local Library display and playback-window edits.

export async function updateTrackDisplay(
  ref,
  fields,
  {request=globalThis.fetch}={}
){
  const id=requiredRef(ref,"Track display update");
  const body={id};
  for(const key of ["title","artist","album","genre"]){
    if(Object.prototype.hasOwnProperty.call(fields??{},key)){
      body[key]=String(fields[key]??"");
    }
  }
  if(Object.keys(body).length===1){
    throw new Error("Track display update requires at least one field");
  }
  return postJSON("/api/library/overlay",body,request,"Track display update");
}

export function restoreTrackDisplay(ref,{request=globalThis.fetch}={}){
  return postJSON(
    "/api/library/overlay",
    {id:requiredRef(ref,"Track display restore"),restore:true},
    request,
    "Track display restore"
  );
}

export async function updatePlaybackWindow(
  ref,
  {
    start=0,
    end=null
  }={},
  {request=globalThis.fetch}={}
){
  const body={
    id:requiredRef(ref,"Playback window update"),
    start:finiteNonNegative(start)
  };
  if(end!==null&&end!==""&&end!==undefined){
    const value=Number(end);
    if(!Number.isFinite(value)){
      throw new Error("Playback window end must be numeric");
    }
    body.end=value;
  }
  return postJSON(
    "/api/library/playback-window",
    body,
    request,
    "Playback window update"
  );
}

export function restorePlaybackWindow(ref,{request=globalThis.fetch}={}){
  return postJSON(
    "/api/library/playback-window",
    {id:requiredRef(ref,"Playback window restore"),restore:true},
    request,
    "Playback window restore"
  );
}

async function postJSON(path,body,request,label){
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

function requiredRef(value,label){
  const ref=String(value??"").trim();
  if(!ref)throw new Error(label+" requires track ref");
  return ref;
}

function finiteNonNegative(value){
  const number=Number(value);
  if(!Number.isFinite(number)||number<0){
    throw new Error("Playback window start must be a non-negative number");
  }
  return number;
}
