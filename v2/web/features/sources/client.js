// [VXG RealForever]
// Browser consumer for desktop source-folder configuration and temporary discovery.
//
// Picking/discovering a folder never admits it. Only addSource mutates the accepted source set.

export async function loadSourceConfig(
 {request=globalThis.fetch}={}
){
 const payload=await requestJSON(
  "/api/source/config",
  {method:"GET",headers:{Accept:"application/json"}},
  request,
  "Source configuration"
 );
 return projectConfig(payload);
}

export async function chooseSourceFolder(
 {request=globalThis.fetch}={}
){
 const payload=await requestJSON(
  "/api/source/choose",
  {
   method:"POST",
   headers:jsonHeaders(),
   body:"{}"
  },
  request,
  "Choose music folder"
 );
 const path=String(payload?.path??"").trim();
 if(!path)throw new Error("Folder picker returned no path");
 return path;
}

export async function addSource(path,options={}){
 return sourceMutation(
  "/api/source/add",
  path,
  options,
  "Add music folder"
 );
}

export async function removeSource(path,options={}){
 return sourceMutation(
  "/api/source/remove",
  path,
  options,
  "Remove music folder"
 );
}

export async function discoverSources(
 {deep=false,request=globalThis.fetch}={}
){
 const payload=await requestJSON(
  "/api/discover",
  {
   method:"POST",
   headers:jsonHeaders(),
   body:JSON.stringify({deep:Boolean(deep)})
  },
  request,
  "Music folder discovery"
 );
 return Object.freeze({
  candidates:Object.freeze(
   Array.isArray(payload?.candidates)
    ? payload.candidates.map(projectCandidate)
    : []
  ),
  scannedDirs:finiteNonNegative(payload?.scannedDirs)
 });
}

export async function ignoreSourceCandidate(path,options={}){
 return sourceMutation(
  "/api/candidate/ignore",
  path,
  options,
  "Ignore music folder"
 );
}

export async function unignoreSource(path,options={}){
 return sourceMutation(
  "/api/candidate/unignore",
  path,
  options,
  "Restore ignored music folder"
 );
}

export async function clearIgnoredSources(
 {request=globalThis.fetch}={}
){
 const payload=await requestJSON(
  "/api/ignored/clear",
  {
   method:"POST",
   headers:jsonHeaders(),
   body:JSON.stringify({
    confirm:"CLEAR_IGNORED_FOLDERS"
   })
  },
  request,
  "Clear ignored music folders"
 );
 return projectMutation(payload);
}

async function sourceMutation(path,value,options,label){
 const sourcePath=String(value??"").trim();
 if(!sourcePath)throw new Error(label+" requires path");
 const payload=await requestJSON(
  path,
  {
   method:"POST",
   headers:jsonHeaders(),
   body:JSON.stringify({path:sourcePath})
  },
  options.request??globalThis.fetch,
  label
 );
 return projectMutation(payload);
}

function projectMutation(payload){
 return Object.freeze({
  ok:payload?.ok===true,
  config:projectConfig(payload?.config??{}),
  libraryRefresh:String(payload?.libraryRefresh??""),
  warning:String(payload?.warning??"")
 });
}

function projectConfig(payload){
 return Object.freeze({
  sources:Object.freeze(stringList(payload?.sources)),
  ignored:Object.freeze(stringList(payload?.ignored))
 });
}

function projectCandidate(value){
 const path=String(value?.path??"").trim();
 if(!path)throw new Error("Discovered music folder is missing path");
 return Object.freeze({
  name:String(value?.name??"").trim(),
  path,
  mp3Count:finiteNonNegative(value?.mp3Count)
 });
}

async function requestJSON(path,options,request,label){
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
 return response.json();
}

function jsonHeaders(){
 return {
  Accept:"application/json",
  "Content-Type":"application/json"
 };
}

function stringList(value){
 const seen=new Set();
 const out=[];
 for(const item of Array.isArray(value)?value:[]){
  const text=String(item??"").trim();
  if(!text||seen.has(text.toLocaleLowerCase()))continue;
  seen.add(text.toLocaleLowerCase());
  out.push(text);
 }
 return out;
}

function finiteNonNegative(value){
 const number=Number(value);
 return Number.isFinite(number)&&number>0?number:0;
}
