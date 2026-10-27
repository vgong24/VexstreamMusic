// [VXG RealForever]
// Local Desktop process identity + explicit shutdown browser boundary.

export async function loadSystemIdentity(
  { request = globalThis.fetch } = {},
) {
  const response=await requireResponse(
    "/health",
    {
      method:"GET",
      headers:{Accept:"application/json"}
    },
    request,
    "System identity"
  );
  const payload=await response.json();
  const version=String(payload?.version??"").trim();
  const buildRef=String(payload?.buildRef??"").trim();
  if(payload?.ok!==true||!version||!buildRef){
    throw new Error("System identity response is incomplete");
  }
  return Object.freeze({
    version,
    buildRef,
    warnings:Object.freeze(
      Array.isArray(payload.warnings)
        ? payload.warnings.map(value=>String(value))
        : []
    )
  });
}

export async function loadSessionRetirePlan(
  { request = globalThis.fetch } = {},
) {
  const response=await requireResponse(
    "/api/system/sessions/retire-plan",
    {
      method:"GET",
      headers:{Accept:"application/json"}
    },
    request,
    "Old-session review"
  );
  const payload=await response.json();
  return Object.freeze({
    currentPID:Number(payload?.currentPID??0),
    sessions:Object.freeze(array(payload?.sessions).map(normalizeSession)),
    eligible:Object.freeze(array(payload?.eligible).map(normalizeCandidate)),
    held:Object.freeze(array(payload?.held).map(normalizeCandidate)),
    scanError:String(payload?.scanError??"").trim(),
    warning:String(payload?.warning??"").trim()
  });
}

export async function retireSessions(
  pids,
  { request = globalThis.fetch } = {},
) {
  const selected=[...new Set(array(pids).map(Number).filter(Number.isInteger))];
  const response=await requireResponse(
    "/api/system/sessions/retire",
    {
      method:"POST",
      headers:{
        Accept:"application/json",
        "Content-Type":"application/json"
      },
      body:JSON.stringify({
        confirm:"RETIRE_OLD_IDLE_SESSIONS",
        pids:selected
      })
    },
    request,
    "Retire old sessions"
  );
  const payload=await response.json();
  return Object.freeze({
    retired:Object.freeze(array(payload?.retired).map(Number)),
    failed:Object.freeze(array(payload?.failed).map(item=>Object.freeze({
      pid:Number(item?.pid??0),
      error:String(item?.error??"").trim()
    }))),
    message:String(payload?.message??"").trim()
  });
}

export async function exitVexStream(
  { request = globalThis.fetch } = {},
) {
  const response=await requireResponse(
    "/api/system/exit",
    {
      method:"POST",
      headers:{
        Accept:"application/json",
        "Content-Type":"application/json"
      },
      body:JSON.stringify({confirm:"EXIT_VEXSTREAM"})
    },
    request,
    "Exit VexStream"
  );
  const payload=await response.json();
  if(payload?.ok!==true||payload?.state!=="EXITING"){
    throw new Error("Exit VexStream response is incomplete");
  }
  return Object.freeze({
    ok:true,
    state:"EXITING"
  });
}

function array(value){
  return Array.isArray(value)?value:[];
}

function normalizeSession(value){
  return Object.freeze({
    pid:Number(value?.pid??0),
    version:String(value?.version??"").trim(),
    buildRef:String(value?.buildRef??"").trim(),
    url:String(value?.url??"").trim(),
    startedAt:String(value?.startedAt??"").trim(),
    current:Boolean(value?.current),
    trackCount:Number(value?.trackCount??0),
    activeImports:Number(value?.activeImports??0),
    libraryVerified:Boolean(value?.libraryVerified),
    importsVerified:Boolean(value?.importsVerified)
  });
}

function normalizeCandidate(value){
  return Object.freeze({
    pid:Number(value?.pid??0),
    version:String(value?.version??"").trim(),
    buildRef:String(value?.buildRef??"").trim(),
    url:String(value?.url??"").trim(),
    startedAt:String(value?.startedAt??"").trim(),
    trackCount:Number(value?.trackCount??0),
    activeImports:Number(value?.activeImports??0),
    uniqueTrackCount:Number(value?.uniqueTrackCount??0),
    eligible:Boolean(value?.eligible),
    reason:String(value?.reason??"").trim()
  });
}

async function requireResponse(path,options,request,label){
  if(typeof request!=="function"){
    throw new Error(label+" requires fetch-like request function");
  }
  const response=await request(path,options);
  if(!response?.ok){
    let detail="";
    try{
      const body=await response.json();
      detail=String(body?.error??"").trim();
    }catch{}
    throw new Error(
      detail||`${label} request failed: ${response?.status??"unknown"}`
    );
  }
  return response;
}
