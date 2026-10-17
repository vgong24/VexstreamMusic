// [VXG RealForever]
// Temporary ProcessMap visual-oracle lab.
//
// V3 remains the lived-product oracle. V4 remains the candidate.
// Controlled provider facts make the two real product UIs comparable without claiming
// those fixture facts are canonical product truth.

import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import {spawn} from "node:child_process";
import {pathToFileURL} from "node:url";

const sleep=ms=>new Promise(resolve=>setTimeout(resolve,ms));

function args(argv=process.argv.slice(2)){
 const out={v3:"",v4:"",output:"",browser:"",timeout:30000};
 for(let i=0;i<argv.length;i++){
  const key=argv[i];
  if(key==="--v3"){out.v3=path.resolve(String(argv[++i]??""));continue;}
  if(key==="--v4"){out.v4=path.resolve(String(argv[++i]??""));continue;}
  if(key==="--output"){out.output=path.resolve(String(argv[++i]??""));continue;}
  if(key==="--browser"){out.browser=String(argv[++i]??"");continue;}
  if(key==="--timeout-ms"){out.timeout=Number(argv[++i]);continue;}
  throw new Error(`unknown argument: ${key}`);
 }
 if(!out.v3||!out.v4||!out.output)throw new Error("--v3, --v4 and --output are required");
 if(!Number.isInteger(out.timeout)||out.timeout<1000||out.timeout>120000)throw new Error("invalid --timeout-ms");
 return out;
}

function executable(explicit=""){
 const home=os.homedir();
 for(const value of [
  explicit,
  process.env.VEXSTREAM_CHROMIUM,
  "/usr/bin/google-chrome",
  "/usr/bin/chromium",
  "/usr/bin/chromium-browser",
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
  "/Applications/Chromium.app/Contents/MacOS/Chromium",
  "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
  "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
  path.join(home,"Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
  path.join(home,"Applications/Chromium.app/Contents/MacOS/Chromium"),
  path.join(home,"Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"),
  path.join(home,"Applications/Brave Browser.app/Contents/MacOS/Brave Browser")
 ]){
  if(!value||!path.isAbsolute(value))continue;
  try{if(fs.statSync(value).isFile())return value;}catch{}
 }
 return null;
}

class CDP{
 constructor(ws){
  this.ws=ws;this.id=0;this.pending=new Map();this.events=[];
  ws.addEventListener("message",event=>{
   const message=JSON.parse(String(event.data));
   if(message.id&&this.pending.has(message.id)){
    const pending=this.pending.get(message.id);
    this.pending.delete(message.id);
    message.error?pending.reject(new Error(JSON.stringify(message.error))):pending.resolve(message.result||{});
    return;
   }
   this.events.push(message);
  });
 }
 send(method,params={},sessionId){
  return new Promise((resolve,reject)=>{
   const id=++this.id;
   this.pending.set(id,{resolve,reject});
   this.ws.send(JSON.stringify({id,method,params,...(sessionId?{sessionId}:{})}));
  });
 }
 close(){try{this.ws.close()}catch{}}
}

async function connect(url){
 const ws=new WebSocket(url);
 await new Promise((resolve,reject)=>{
  ws.addEventListener("open",resolve,{once:true});
  ws.addEventListener("error",reject,{once:true});
 });
 return new CDP(ws);
}

async function activePort(file,child,stderr,timeout){
 const end=Date.now()+Math.min(timeout,15000);
 while(Date.now()<end){
  if(fs.existsSync(file)){
   const value=fs.readFileSync(file,"utf8").trim();
   if(value)return value;
  }
  if(child.exitCode!==null)throw new Error(`browser exited ${child.exitCode}: ${stderr()}`);
  await sleep(40);
 }
 throw new Error(`DevTools timeout: ${stderr()}`);
}

function kill(child,signal){
 if(child.exitCode!==null)return;
 try{process.platform==="win32"?child.kill(signal):process.kill(-child.pid,signal);}catch{}
}

async function evaluate(c,sessionId,expression,{awaitPromise=true}={}){
 const response=await c.send("Runtime.evaluate",{
  expression,
  returnByValue:true,
  awaitPromise
 },sessionId);
 if(response.exceptionDetails){
  throw new Error(response.exceptionDetails.exception?.description||response.exceptionDetails.text||"browser evaluation failed");
 }
 return response.result?.value;
}

async function waitFor(c,sessionId,expression,label,timeout=10000){
 const end=Date.now()+timeout;
 let last;
 while(Date.now()<end){
  try{
   last=await evaluate(c,sessionId,`Boolean(${expression})`);
   if(last)return;
  }catch(error){last=error.message}
  await sleep(25);
 }
 throw new Error(`timeout waiting for ${label}; last=${String(last)}`);
}

async function settle(c,sessionId){
 await evaluate(c,sessionId,"new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>setTimeout(resolve,35))))");
}

async function screenshot(c,sessionId,file){
 await settle(c,sessionId);
 const shot=await c.send("Page.captureScreenshot",{
  format:"png",
  captureBeyondViewport:false
 },sessionId);
 fs.writeFileSync(file,Buffer.from(shot.data,"base64"));
}

async function pageSession(c,markup){
 const {targetId}=await c.send("Target.createTarget",{url:"about:blank"});
 const {sessionId}=await c.send("Target.attachToTarget",{targetId,flatten:true});
 await c.send("Page.enable",{},sessionId);
 await c.send("Runtime.enable",{},sessionId);
 await c.send("Emulation.setDeviceMetricsOverride",{
  width:1440,
  height:900,
  deviceScaleFactor:1,
  mobile:false
 },sessionId);
 const tree=await c.send("Page.getFrameTree",{},sessionId);
 const frameId=tree.frameTree?.frame?.id;
 if(!frameId)throw new Error("root frame missing");
 await c.send("Page.setDocumentContent",{frameId,html:markup},sessionId);
 return {targetId,sessionId};
}

const tracks=[
 {id:"media.vexstream.test-tone.1",title:"VexStream Test Tone One",artist:"VexStream QA",album:"V4 Practicum",genre:"Test",year:"2026",duration:75,addedAt:"2026-10-12T00:00:00Z",indexedAt:"2026-10-12T00:00:00Z",filename:"tone-one.mp3",playbackStart:0},
 {id:"media.vexstream.test-tone.2",title:"VexStream Test Tone Two",artist:"VexStream QA",album:"V4 Practicum",genre:"Test",year:"2026",duration:150,addedAt:"2026-10-14T00:00:00Z",indexedAt:"2026-10-14T00:00:00Z",filename:"tone-two.mp3",playbackStart:0},
 {id:"media.vexstream.test-tone.3",title:"VexStream Test Tone Three",artist:"Other Artist",album:"Other Album",genre:"Ambient",year:"2025",duration:225,addedAt:"2026-10-13T00:00:00Z",indexedAt:"2026-10-13T00:00:00Z",filename:"tone-three.mp3",playbackStart:0}
];

function v3Fixture(){
 const radio={
  schemaVersion:"vexstream.discovery-candidates/v2",
  seed:{kind:"library",trackId:"media.vexstream.test-tone.1",title:"VexStream Test Tone One",artist:"VexStream QA"},
  basis:{provider:"fixture",method:"SEARCH_DERIVED_MULTI_AXIS",queries:[{label:"Closer"}],creatorCap:2,selection:"ROUND_ROBIN"},
  candidates:[{providerId:"candidate.one",url:"https://www.youtube.com/watch?v=candidate.one",title:"Candidate One",channel:"Creator One",duration:120,axis:"CLOSER",axisLabel:"Closer",libraryMatch:{state:"NOT_IN_LIBRARY"}}],
  generatedAt:"2026-10-16T00:00:00Z"
 };
 const source={path:"/music",name:"Music"};
 const playlist={id:"road-trip",name:"Road Trip",trackIds:["media.vexstream.test-tone.1","media.vexstream.test-tone.3"]};
 return `<script>(()=>{const tracks=${JSON.stringify(tracks)},source=${JSON.stringify(source)},playlist=${JSON.stringify(playlist)},radio=${JSON.stringify(radio)};const json=(value,status=200)=>new Response(JSON.stringify(value),{status,headers:{"Content-Type":"application/json"}});globalThis.fetch=async(input,opt={})=>{const url=new URL(typeof input==="string"?input:input.url,"https://vexstream-v3.local/"),method=String(opt.method||"GET").toUpperCase(),p=url.pathname;if(p==="/api/library")return json({tracks,sources:[source],candidates:[],ignored:[]});if(p==="/api/library/trash")return json({trash:[],deletedHistory:[]});if(p==="/api/playlists")return json({playlists:[playlist]});if(p==="/api/import/jobs")return json({active:[],ready:[],completed:[]});if(p==="/api/import/status")return json({ready:true,searchReady:true,ffmpegReady:true,importReady:true,detail:"Deterministic visual oracle fixture."});if(p==="/api/system/preflight")return json({version:"3.0.0",compatibility:"Protected V3 lived-product oracle",issues:[],issueCount:0,scan:{running:false},library:{tracks:3,sources:1},liveInstances:[],recoveryProjection:false,recoveredNow:false});if(p==="/api/diagnostics/session")return json({listening:false,sessionId:"visual-oracle",eventCount:0,sessionStartedAt:"2026-10-16T00:00:00Z"});if(p==="/api/discovery/radio"&&method==="POST")return json(radio);if(p==="/api/diagnostics/event"&&method==="POST")return json({ok:true});return json({});};})();</script>`;
}

function v3Markup(root){
 const file=path.join(root,"ui","index.html");
 let html=fs.readFileSync(file,"utf8");
 html=html.replace("<head>","<head><base href=\"https://vexstream-v3.local/\">");
 const marker="<script>";
 if(!html.includes(marker))throw new Error("V3 script marker missing");
 html=html.replace(marker,`${v3Fixture()}${marker}`);
 return html;
}

async function v4Markup(root){
 const fixturePath=path.join(root,"v2","tests","lived-shell-browser-fixture.mjs");
 const {buildSmokeHTML}=await import(pathToFileURL(fixturePath).href+"?visual="+Date.now());
 let html=buildSmokeHTML({webRoot:path.join(root,"v2","web"),schema:"vexstream.v4.visual-oracle/v1"});
 const marker="<script>(()=>{const checks=[]";
 const start=html.lastIndexOf(marker);
 if(start<0)throw new Error("V4 lived driver marker missing");
 const end=html.indexOf("</script>",start);
 if(end<0)throw new Error("V4 lived driver end missing");
 return html.slice(0,start)+html.slice(end+"</script>".length);
}

async function recordMeta(c,sessionId,version,name,file,meta){
 const snapshot=await evaluate(c,sessionId,`(()=>({title:document.title,bodyClass:document.body.className,bodyDataset:{...document.body.dataset},visibleText:document.body.innerText.replace(/\\s+/g," ").trim().slice(0,1400)}))()`);
 meta.push({version,name,file:path.basename(file),snapshot});
}

async function captureState(c,sessionId,dir,version,name,meta){
 const file=path.join(dir,`${name}.png`);
 await screenshot(c,sessionId,file);
 await recordMeta(c,sessionId,version,name,file,meta);
}

async function captureV3(c,markup,dir,meta,timeout){
 fs.mkdirSync(dir,{recursive:true});
 const {sessionId}=await pageSession(c,markup);
 await waitFor(c,sessionId,'document.querySelector("#statSongs")?.textContent==="3"',"V3 library fixture",timeout);

 const states=[
  ["home",'goLibraryHome()', 'document.querySelector("#homeView")&&!document.querySelector("#homeView").classList.contains("hidden")'],
  ["songs",'navigateLibraryScope("songs")','document.querySelector("#songsView")&&!document.querySelector("#songsView").classList.contains("hidden")'],
  ["albums",'navigateLibraryScope("albums")','document.querySelectorAll("#albumGrid [data-album-open]").length===2'],
  ["album-detail",'openAlbum(encodeURIComponent("V4 Practicum"))','document.querySelector("#albumDetail")&&!document.querySelector("#albumDetail").classList.contains("hidden")&&document.querySelector("#albumDetail").innerText.includes("V4 Practicum")'],
  ["artist-detail",'navigateLibraryScope("artists");openArtist(encodeURIComponent("VexStream QA"))','document.querySelector("#artistDetail")&&!document.querySelector("#artistDetail").classList.contains("hidden")&&document.querySelector("#artistDetail").innerText.includes("VexStream QA")'],
  ["playlist-detail",'openPlaylist("road-trip")','document.querySelector("#playlistDetail")?.innerText.includes("Road Trip")'],
  ["discover-pc",'topScreen("discover");discoverPane("pc")','document.querySelector("#pcDiscoverPane")&&!document.querySelector("#pcDiscoverPane").classList.contains("hidden")'],
  ["discover-youtube",'topScreen("discover");discoverPane("youtube")','document.querySelector("#youtubeDiscoverPane")&&!document.querySelector("#youtubeDiscoverPane").classList.contains("hidden")'],
  ["discover-explore",'await startDiscoveryFromTrack("media.vexstream.test-tone.1")','document.querySelector("#radioWorkspace")&&!document.querySelector("#radioWorkspace").classList.contains("hidden")&&document.querySelectorAll("#radioCandidates .radio-card").length===1']
 ];
 for(const [name,action,predicate] of states){
  await evaluate(c,sessionId,`(async()=>{${action}})()`);
  await waitFor(c,sessionId,predicate,`V3 ${name}`,timeout);
  await captureState(c,sessionId,dir,"v3",name,meta);
 }
}

async function captureV4(c,markup,dir,meta,timeout){
 fs.mkdirSync(dir,{recursive:true});
 const {sessionId}=await pageSession(c,markup);
 await waitFor(c,sessionId,"window.vexV4?.ready===true","V4 ready",timeout);
 const states=[
  ["home",'window.vexV4.shellNavigation.select("home")','window.vexV4.shellNavigation.currentProjection.view==="home"&&document.querySelector("#homeView")&&!document.querySelector("#homeView").hidden'],
  ["songs",'window.vexV4.shellNavigation.select("songs")','window.vexV4.shellNavigation.currentProjection.view==="songs"&&document.querySelectorAll("#trackChoices [data-track-ref]").length===3'],
  ["albums",'window.vexV4.shellNavigation.select("albums")','window.vexV4.shellNavigation.currentProjection.view==="albums"&&document.querySelectorAll("#trackChoices [data-library-group]").length===2'],
  ["album-detail",'window.vexV4.shellNavigation.openLibraryDetail("album","v4 practicum")','window.vexV4.shellNavigation.currentProjection.libraryDetail?.kind==="album"&&document.querySelector("#libraryGroupTitle")?.textContent==="V4 Practicum"'],
  ["artist-detail",'window.vexV4.shellNavigation.openLibraryDetail("artist","vexstream qa")','window.vexV4.shellNavigation.currentProjection.libraryDetail?.kind==="artist"&&document.querySelector("#libraryGroupTitle")?.textContent==="VexStream QA"'],
  ["playlist-detail",'window.vexV4.shellNavigation.select("playlists");window.vexV4.playlistNavigation.open("road-trip")','window.vexV4.playlistNavigation.currentProjection.openPlaylistID==="road-trip"&&document.querySelector("#playlistDetailName")?.value==="Road Trip"'],
  ["discover-pc",'window.vexV4.shellNavigation.select("discover");window.vexV4.discoverNavigation.select("pc")','window.vexV4.shellNavigation.currentProjection.view==="discover"&&window.vexV4.discoverNavigation.currentProjection.mode==="pc"'],
  ["discover-youtube",'window.vexV4.discoverNavigation.select("youtube")','window.vexV4.discoverNavigation.currentProjection.mode==="youtube"'],
  ["discover-explore",'window.vexV4.discoverNavigation.select("explore");await window.vexV4.explore.replaceFromTrack("media.vexstream.test-tone.1")','window.vexV4.discoverNavigation.currentProjection.mode==="explore"&&window.vexV4.exploreSession.currentProjection.kind==="active"&&document.querySelectorAll("#exploreCandidates [data-explore-index]").length===1']
 ];
 for(const [name,action,predicate] of states){
  await evaluate(c,sessionId,`(async()=>{${action}})()`);
  await waitFor(c,sessionId,predicate,`V4 ${name}`,timeout);
  await captureState(c,sessionId,dir,"v4",name,meta);
 }
}

async function main(){
 const opt=args();
 const exe=executable(opt.browser);
 if(!exe)throw new Error("No Chromium-family browser found");
 fs.mkdirSync(opt.output,{recursive:true});
 const v3Dir=path.join(opt.output,"v3");
 const v4Dir=path.join(opt.output,"v4");
 const meta=[];
 const root=fs.mkdtempSync(path.join(os.tmpdir(),"vexstream-visual-oracle-"));
 const profile=path.join(root,"profile");
 fs.mkdirSync(profile,{recursive:true});
 const child=spawn(exe,[
  "--headless=new",
  "--no-sandbox",
  "--disable-gpu",
  "--disable-background-networking",
  "--disable-component-update",
  "--no-first-run",
  "--autoplay-policy=no-user-gesture-required",
  "--remote-debugging-port=0",
  `--user-data-dir=${profile}`,
  "about:blank"
 ],{stdio:["ignore","pipe","pipe"],detached:process.platform!=="win32"});
 let stderr="";child.stderr.on("data",bytes=>stderr+=String(bytes));
 let c;
 try{
  const active=await activePort(path.join(profile,"DevToolsActivePort"),child,()=>stderr.slice(-2500),opt.timeout);
  const [port,route]=active.split(/\r?\n/);
  c=await connect(`ws://127.0.0.1:${port}${route}`);
  const version=await c.send("Browser.getVersion");
  await captureV3(c,v3Markup(opt.v3),v3Dir,meta,opt.timeout);
  await captureV4(c,await v4Markup(opt.v4),v4Dir,meta,opt.timeout);
  const manifest={
   schemaVersion:"vexstream.processmap.visual-oracle/v1",
   status:"CAPTURED_NOT_ACCEPTED",
   browserVersion:version.product,
   viewport:{width:1440,height:900,deviceScaleFactor:1},
   oracle:"protected V3 ui/index.html with deterministic provider-boundary facts",
   candidate:"exact V4 target with its committed deterministic browser fixture",
   states:["home","songs","albums","album-detail","artist-detail","playlist-detail","discover-pc","discover-youtube","discover-explore"],
   note:"Screenshots are perception evidence. They are not automatically accepted visual goldens and do not replace Victor lived-product acceptance.",
   captures:meta
  };
  fs.writeFileSync(path.join(opt.output,"manifest.json"),JSON.stringify(manifest,null,2)+"\n");
  console.log(JSON.stringify({status:manifest.status,browserVersion:manifest.browserVersion,states:manifest.states,captures:meta.length},null,2));
 }finally{
  c?.close();
  kill(child,"SIGTERM");
  await Promise.race([new Promise(resolve=>child.once("exit",resolve)),sleep(1000)]);
  kill(child,"SIGKILL");
  await sleep(100);
  fs.rmSync(root,{recursive:true,force:true,maxRetries:20,retryDelay:100});
 }
}

main().catch(error=>{
 console.error(error?.stack||String(error));
 process.exitCode=1;
});
