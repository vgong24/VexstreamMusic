// [VXG RealForever]
// Temporary ProcessMap proof lab: observe one exact VexStream candidate without changing its bytes.

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {spawn} from 'node:child_process';
import {pathToFileURL} from 'node:url';

const sleep=ms=>new Promise(r=>setTimeout(r,ms));

function parseArgs(argv=process.argv.slice(2)){
  const out={target:'',output:'',browser:'',timeout:30000};
  for(let i=0;i<argv.length;i++){
    const key=argv[i];
    if(key==='--target'){out.target=path.resolve(String(argv[++i]??''));continue;}
    if(key==='--output'){out.output=path.resolve(String(argv[++i]??''));continue;}
    if(key==='--browser'){out.browser=String(argv[++i]??'');continue;}
    if(key==='--timeout-ms'){out.timeout=Number(argv[++i]);continue;}
    throw new Error(`unknown argument: ${key}`);
  }
  if(!out.target)throw new Error('--target is required');
  if(!out.output)throw new Error('--output is required');
  if(!Number.isInteger(out.timeout)||out.timeout<1000||out.timeout>120000)throw new Error('invalid --timeout-ms');
  return out;
}

function browserExecutable(explicit=''){
  const home=os.homedir();
  const candidates=[
    explicit,
    process.env.VEXSTREAM_CHROMIUM,
    '/usr/bin/google-chrome',
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser',
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Chromium.app/Contents/MacOS/Chromium',
    '/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge',
    '/Applications/Brave Browser.app/Contents/MacOS/Brave Browser',
    path.join(home,'Applications/Google Chrome.app/Contents/MacOS/Google Chrome'),
    path.join(home,'Applications/Chromium.app/Contents/MacOS/Chromium'),
    path.join(home,'Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge'),
    path.join(home,'Applications/Brave Browser.app/Contents/MacOS/Brave Browser'),
  ];
  for(const value of candidates){
    if(!value||!path.isAbsolute(value))continue;
    try{if(fs.statSync(value).isFile())return value;}catch{}
  }
  return null;
}

class CDP{
  constructor(ws){
    this.ws=ws;this.id=0;this.pending=new Map();this.events=[];this.listeners=new Map();this.listenerErrors=[];
    ws.addEventListener('message',event=>{
      const message=JSON.parse(String(event.data));
      if(message.id&&this.pending.has(message.id)){
        const pending=this.pending.get(message.id);this.pending.delete(message.id);
        message.error?pending.reject(new Error(JSON.stringify(message.error))):pending.resolve(message.result||{});
        return;
      }
      this.events.push(message);
      for(const listener of this.listeners.get(message.method)||[]){
        Promise.resolve(listener(message.params||{},message.sessionId||null)).catch(error=>this.listenerErrors.push(error));
      }
    });
  }
  send(method,params={},sessionId){
    return new Promise((resolve,reject)=>{
      const id=++this.id;this.pending.set(id,{resolve,reject});
      this.ws.send(JSON.stringify({id,method,params,...(sessionId?{sessionId}:{})}));
    });
  }
  on(method,listener){if(!this.listeners.has(method))this.listeners.set(method,new Set());this.listeners.get(method).add(listener);}
  close(){try{this.ws.close()}catch{}}
}

async function connect(url){
  const ws=new WebSocket(url);
  await new Promise((resolve,reject)=>{ws.addEventListener('open',resolve,{once:true});ws.addEventListener('error',reject,{once:true});});
  return new CDP(ws);
}

async function activePort(file,child,stderr,timeout){
  const end=Date.now()+Math.min(timeout,8000);
  while(Date.now()<end){
    if(fs.existsSync(file)){const value=fs.readFileSync(file,'utf8').trim();if(value)return value;}
    if(child.exitCode!==null)throw new Error(`browser exited ${child.exitCode}: ${stderr()}`);
    await sleep(40);
  }
  throw new Error(`DevTools timeout: ${stderr()}`);
}

function kill(child,signal){
  if(child.exitCode!==null)return;
  try{process.platform==='win32'?child.kill(signal):process.kill(-child.pid,signal);}catch{}
}

function headerValue(headers,name){
  const wanted=String(name).toLowerCase();
  for(const [key,value] of Object.entries(headers||{}))if(String(key).toLowerCase()===wanted)return String(value);
  return '';
}

function mediaReply(bytes,request){
  const method=String(request.method||'GET').toUpperCase();
  const common=[
    {name:'Accept-Ranges',value:'bytes'},
    {name:'Content-Type',value:'audio/mpeg'},
    {name:'X-Content-Type-Options',value:'nosniff'},
  ];
  if(method==='HEAD')return{responseCode:200,responseHeaders:[...common,{name:'Content-Length',value:String(bytes.length)}],body:''};
  const range=headerValue(request.headers,'range').trim();
  const match=/^bytes=(\d+)-(\d*)$/i.exec(range);
  if(match){
    const start=Number(match[1]);
    const requestedEnd=match[2]?Number(match[2]):bytes.length-1;
    const end=Math.min(requestedEnd,bytes.length-1);
    if(Number.isInteger(start)&&Number.isInteger(end)&&start>=0&&start<=end&&start<bytes.length){
      const body=bytes.subarray(start,end+1);
      return{responseCode:206,responseHeaders:[...common,{name:'Content-Range',value:`bytes ${start}-${end}/${bytes.length}`},{name:'Content-Length',value:String(body.length)}],body:body.toString('base64')};
    }
  }
  return{responseCode:200,responseHeaders:[...common,{name:'Content-Length',value:String(bytes.length)}],body:bytes.toString('base64')};
}

function instrumentationScript(){
  return `<script>(()=>{
    const trace=[];const captures=[];const captured=new Set();
    const clone=value=>{try{return JSON.parse(JSON.stringify(value))}catch{return String(value)}};
    const shell=()=>window.vexV4?.shellNavigation?.currentProjection??null;
    const library=()=>window.vexV4?.libraryView??null;
    const describe=event=>{const target=event?.target;const action=target?.closest?.('[data-library-group-action]')??null;const card=target?.closest?.('[data-library-group]')??null;return{target:target?.tagName??null,targetId:target?.id??null,action:action?.dataset?.libraryGroupAction??null,groupKey:card?.dataset?.libraryGroup??null,groupKind:card?.dataset?.libraryGroupKind??null,defaultPrevented:Boolean(event?.defaultPrevented),cancelBubble:Boolean(event?.cancelBubble)};};
    const record=(kind,detail={})=>trace.push({at:performance.now(),kind,...clone(detail),shell:clone(shell()),libraryView:clone(library())});
    const capture=name=>{if(captured.has(name))return;captured.add(name);captures.push({name,at:performance.now(),shell:clone(shell()),libraryView:clone(library())});};
    const originalClick=HTMLElement.prototype.click;
    HTMLElement.prototype.click=function(){const relevant=this.matches?.('[data-library-group-action]');if(relevant)record('programmatic-click-before',{element:{action:this.dataset.libraryGroupAction,text:this.textContent}});const result=originalClick.call(this);if(relevant)record('programmatic-click-after',{element:{action:this.dataset.libraryGroupAction,text:this.textContent}});return result;};
    const originalAdd=EventTarget.prototype.addEventListener;
    EventTarget.prototype.addEventListener=function(type,listener,options){
      if(type==='click'&&this instanceof Element&&this.id==='trackChoices'&&typeof listener==='function'){
        const registration=trace.filter(x=>x.kind==='trackChoices-listener-registered').length+1;
        record('trackChoices-listener-registered',{registration});
        const wrapped=function(event){record('trackChoices-listener-before',{registration,event:describe(event)});try{return listener.call(this,event);}finally{record('trackChoices-listener-after',{registration,event:describe(event)});}};
        return originalAdd.call(this,type,wrapped,options);
      }
      return originalAdd.call(this,type,listener,options);
    };
    document.addEventListener('click',event=>{const d=describe(event);if(d.action||d.groupKey)record('document-click-capture',{event:d});},true);
    document.addEventListener('click',event=>{const d=describe(event);if(d.action||d.groupKey)record('document-click-bubble',{event:d});});
    let subscribed=false;
    const timer=setInterval(()=>{
      if(!window.vexV4?.ready)return;
      if(!subscribed){subscribed=true;window.vexV4.shellNavigation.subscribe(projection=>record('shell-publish',{projection:clone(projection)}));record('shell-subscriber-attached');}
      const projection=shell();
      if(projection?.view==='home'&&!document.querySelector('#homeView')?.hidden)capture('library-home');
      if(projection?.view==='songs'&&document.querySelectorAll('#trackChoices [data-track-ref]').length)capture('songs');
      if(projection?.view==='albums'&&document.querySelectorAll('#trackChoices [data-library-group]').length)capture('albums');
      if(projection?.libraryDetail?.kind==='album')capture('album-detail');
      if(projection?.view==='artists'&&document.querySelectorAll('#trackChoices [data-library-group]').length)capture('artists');
      if(projection?.libraryDetail?.kind==='artist')capture('artist-detail');
      if(projection?.view==='playlists')capture(window.vexV4.playlistNavigation?.currentProjection?.openPlaylistID?'playlist-detail':'playlists');
      if(projection?.view==='discover')capture('discover-'+(window.vexV4.discoverNavigation?.currentProjection?.mode??'unknown'));
      if(document.querySelector('#vexSmokeResult'))clearInterval(timer);
    },10);
    window.__vexProcessMap={trace,captures,record,capture};
  })();</script>`;
}

async function evaluate(c,sessionId,expression){
  const result=await c.send('Runtime.evaluate',{expression,returnByValue:true},sessionId);
  return result.result?.value;
}

async function captureScreenshot(c,sessionId,file){
  const shot=await c.send('Page.captureScreenshot',{format:'png',captureBeyondViewport:false},sessionId);
  fs.writeFileSync(file,Buffer.from(shot.data,'base64'));
}

async function run({exe,markup,profile,mediaBytes,timeout,output}){
  const child=spawn(exe,[
    '--headless=new','--no-sandbox','--disable-gpu','--disable-background-networking','--disable-component-update','--no-first-run',
    '--autoplay-policy=no-user-gesture-required','--remote-debugging-port=0',`--user-data-dir=${profile}`,'about:blank'
  ],{stdio:['ignore','pipe','pipe'],detached:process.platform!=='win32'});
  let stderr='';child.stderr.on('data',b=>stderr+=String(b));
  let c;
  try{
    const active=await activePort(path.join(profile,'DevToolsActivePort'),child,()=>stderr.slice(-2500),timeout);
    const [port,route]=active.split(/\r?\n/);
    c=await connect(`ws://127.0.0.1:${port}${route}`);
    const version=await c.send('Browser.getVersion');
    const {targetId}=await c.send('Target.createTarget',{url:'about:blank'});
    const {sessionId}=await c.send('Target.attachToTarget',{targetId,flatten:true});
    await c.send('Page.enable',{},sessionId);
    await c.send('Runtime.enable',{},sessionId);
    await c.send('Emulation.setDeviceMetricsOverride',{width:1440,height:900,deviceScaleFactor:1,mobile:false},sessionId);
    await c.send('Fetch.enable',{patterns:[{urlPattern:'https://vexstream.local/media/*',requestStage:'Request'}]},sessionId);
    const mediaRequests=[];
    c.on('Fetch.requestPaused',async(params,eventSessionId)=>{
      if(eventSessionId!==sessionId)return;
      const request=params.request||{},url=new URL(request.url);
      mediaRequests.push({method:String(request.method||'GET').toUpperCase(),path:url.pathname,range:headerValue(request.headers,'range')||null});
      const reply=mediaReply(mediaBytes,request);
      await c.send('Fetch.fulfillRequest',{requestId:params.requestId,responseCode:reply.responseCode,responseHeaders:reply.responseHeaders,body:reply.body},sessionId);
    });
    const tree=await c.send('Page.getFrameTree',{},sessionId);const frameId=tree.frameTree?.frame?.id;if(!frameId)throw new Error('no root frame');
    await c.send('Page.setDocumentContent',{frameId,html:markup},sessionId);

    const screens=path.join(output,'screens');fs.mkdirSync(screens,{recursive:true});
    const captured=new Set();let encoded=null;
    const end=Date.now()+timeout;
    while(Date.now()<end){
      if(c.listenerErrors.length)throw c.listenerErrors[0];
      const queue=await evaluate(c,sessionId,'window.__vexProcessMap?.captures?.splice(0)??[]')||[];
      for(const item of queue){
        if(captured.has(item.name))continue;captured.add(item.name);
        await captureScreenshot(c,sessionId,path.join(screens,`${String(captured.size).padStart(2,'0')}-${item.name}.png`));
      }
      encoded=await evaluate(c,sessionId,'document.querySelector("#vexSmokeResult")?.dataset.smokeResult||null');
      if(encoded)break;
      await sleep(30);
    }
    await captureScreenshot(c,sessionId,path.join(screens,'99-final.png'));
    const trace=await evaluate(c,sessionId,'window.__vexProcessMap?.trace??[]')||[];
    const finalState=await evaluate(c,sessionId,'({shell:window.vexV4?.shellNavigation?.currentProjection??null,libraryView:window.vexV4?.libraryView??null,groupHTML:[...document.querySelectorAll("#trackChoices [data-library-group]")].map(x=>x.outerHTML),result:document.querySelector("#vexSmokeResult")?.dataset.smokeResult||null})');
    if(!encoded)throw new Error('no smoke result');
    for(let i=0;i<25&&!mediaRequests.length;i++)await sleep(10);
    return{
      browserVersion:version.product,
      result:JSON.parse(Buffer.from(encoded,'base64').toString()),
      mediaRequests,
      trace,
      finalState,
      exceptions:c.events.filter(e=>e.method==='Runtime.exceptionThrown').map(e=>e.params?.exceptionDetails?.text||'exception'),
      consoleErrors:c.events.filter(e=>e.method==='Runtime.consoleAPICalled'&&e.params?.type==='error').map(e=>(e.params.args||[]).map(a=>a.value||a.description||'').join(' ')),
    };
  }finally{
    c?.close();kill(child,'SIGTERM');await Promise.race([new Promise(r=>child.once('exit',r)),sleep(1000)]);kill(child,'SIGKILL');
  }
}

async function main(){
  const opt=parseArgs();fs.mkdirSync(opt.output,{recursive:true});
  const fixturePath=path.join(opt.target,'v2','tests','lived-shell-browser-fixture.mjs');
  const mediaPath=path.join(opt.target,'tests','fixtures','vexstream-test-tone.mp3');
  if(!fs.existsSync(fixturePath))throw new Error(`target fixture missing: ${fixturePath}`);
  if(!fs.existsSync(mediaPath))throw new Error(`target media missing: ${mediaPath}`);
  const {buildSmokeHTML}=await import(pathToFileURL(fixturePath).href);
  let markup=buildSmokeHTML({webRoot:path.join(opt.target,'v2','web'),schema:'vexstream.v4.processmap-diagnostic/v1'});
  const marker='<script type="module">';
  if(!markup.includes(marker))throw new Error('module marker missing from target fixture');
  markup=markup.replace(marker,`${instrumentationScript()}${marker}`);
  const exe=browserExecutable(opt.browser);if(!exe)throw new Error('No Chromium-family browser found');
  const root=fs.mkdtempSync(path.join(os.tmpdir(),'vexstream-processmap-'));const profile=path.join(root,'profile');fs.mkdirSync(profile,{recursive:true});
  try{
    const observation=await run({exe,markup,profile,mediaBytes:fs.readFileSync(mediaPath),timeout:opt.timeout,output:opt.output});
    fs.writeFileSync(path.join(opt.output,'event-trace.json'),JSON.stringify(observation.trace,null,2)+'\n');
    fs.writeFileSync(path.join(opt.output,'final-state.json'),JSON.stringify(observation.finalState,null,2)+'\n');
    const summary={schemaVersion:'vexstream.v4.processmap-diagnostic/v1',status:observation.result.status,browserExecutable:exe,browserVersion:observation.browserVersion,result:observation.result,mediaRequests:observation.mediaRequests,exceptions:observation.exceptions,consoleErrors:observation.consoleErrors,traceCount:observation.trace.length,screenshots:fs.readdirSync(path.join(opt.output,'screens')).sort()};
    fs.writeFileSync(path.join(opt.output,'summary.json'),JSON.stringify(summary,null,2)+'\n');
    console.log(JSON.stringify(summary,null,2));
    if(summary.status!=='PASS')process.exitCode=1;
  }finally{await sleep(250);fs.rmSync(root,{recursive:true,force:true,maxRetries:20,retryDelay:100});}
}

main().catch(error=>{console.error(error?.stack||String(error));process.exitCode=1;});
