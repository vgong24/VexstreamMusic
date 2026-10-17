// [VXG RealForever]
// Fast lived-product browser smoke for the VexStream V4 shell.
// Owns proof mechanics only; product semantics remain in the feature owners it executes.

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {buildSmokeHTML} from './lived-shell-browser-fixture.mjs';
import {browserRun} from './lived-shell-browser-cdp.mjs';

const SCHEMA='vexstream.v4.lived-shell-browser-smoke/v1';
const HERE=path.dirname(fileURLToPath(import.meta.url));
const V2=path.resolve(HERE,'..');
const ROOT=path.resolve(V2,'..');
const WEB=path.join(V2,'web');
const MEDIA_FIXTURE=path.join(ROOT,'tests','fixtures','vexstream-test-tone.mp3');
const TIMEOUT=25000;
const sleep=ms=>new Promise(r=>setTimeout(r,ms));

function args(argv=process.argv.slice(2)){
  const out={browser:'',timeout:TIMEOUT};
  for(let i=0;i<argv.length;i++){
    if(argv[i]==='--browser-executable'){out.browser=String(argv[++i]??'');continue;}
    if(argv[i]==='--timeout-ms'){out.timeout=Number(argv[++i]);if(!Number.isInteger(out.timeout)||out.timeout<1000||out.timeout>120000)throw new Error('invalid --timeout-ms');continue;}
    throw new Error(`unknown argument: ${argv[i]}`);
  }
  return out;
}
function executable(explicit=''){
  const home=os.homedir();
  for(const value of [explicit,process.env.VEXSTREAM_CHROMIUM,'/usr/bin/chromium','/usr/bin/google-chrome','/opt/homebrew/bin/chromium','/usr/local/bin/chromium','/Applications/Google Chrome.app/Contents/MacOS/Google Chrome','/Applications/Chromium.app/Contents/MacOS/Chromium',path.join(home,'Applications/Google Chrome.app/Contents/MacOS/Google Chrome')]){
    if(!value||!path.isAbsolute(value))continue;
    try{if(fs.statSync(value).isFile())return value;}catch{}
  }
  return null;
}
async function main(){const opt=args(),exe=executable(opt.browser);if(!exe){const e=new Error('No Chromium-family browser found');e.code='BROWSER_EXECUTABLE_NOT_FOUND';throw e;}if(!fs.existsSync(MEDIA_FIXTURE)){const e=new Error(`media fixture missing: ${MEDIA_FIXTURE}`);e.code='MEDIA_FIXTURE_MISSING';throw e;}const root=fs.mkdtempSync(path.join(os.tmpdir(),'vexstream-smoke-')),profile=path.join(root,'profile');fs.mkdirSync(profile,{recursive:true});try{const run=await browserRun(exe,buildSmokeHTML({webRoot:WEB,schema:SCHEMA}),profile,fs.readFileSync(MEDIA_FIXTURE),opt.timeout);const requiredMedia=run.mediaRequests.some(r=>r.method==='GET'&&r.path==='/media/media.vexstream.test-tone.1');if(run.result.status==='PASS'&&!requiredMedia)throw new Error('PASS without real media GET');if(run.result.status==='PASS'&&(run.exceptions.length||run.consoleErrors.length))throw new Error('PASS with browser runtime errors');const result={schemaVersion:SCHEMA,status:run.result.status,browserExecutable:exe,browserVersion:run.browserVersion,result:run.result,mediaRequests:run.mediaRequests,exceptions:run.exceptions,consoleErrors:run.consoleErrors};console.log(JSON.stringify(result,null,2));if(result.status!=='PASS')process.exitCode=1;}finally{await sleep(250);fs.rmSync(root,{recursive:true,force:true,maxRetries:20,retryDelay:100});}}
main().catch(e=>{console.error(JSON.stringify({schemaVersion:SCHEMA,status:'FAIL',failure:{code:e.code||'LIVED_SHELL_SMOKE_FAILED',message:e.message}},null,2));process.exitCode=1;});
