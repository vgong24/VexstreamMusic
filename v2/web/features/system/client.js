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
