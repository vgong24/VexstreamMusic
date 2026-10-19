// [VXG RealForever]
// Repository-owned V4 local verification entry.
//
// This composes existing proof owners; it does not redefine their semantics.
// GitHub Actions, VexRelay, a developer terminal, or another caller may invoke
// this same entry instead of re-implementing the proof sequence elsewhere.

import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const HERE=path.dirname(fileURLToPath(import.meta.url));
const V2=path.resolve(HERE,"..");
const ROOT=path.resolve(V2,"..");
const npm=process.platform==="win32"?"npm.cmd":"npm";

function goFiles(root){
 const out=[];
 for(const entry of fs.readdirSync(root,{withFileTypes:true})){
  const full=path.join(root,entry.name);
  if(entry.isDirectory()){
   out.push(...goFiles(full));
   continue;
  }
  if(entry.isFile()&&entry.name.endsWith(".go"))out.push(full);
 }
 return out;
}

function run(name,command,args,cwd){
 process.stdout.write("\n== "+name+" ==\n");
 const result=spawnSync(command,args,{
  cwd,
  env:process.env,
  encoding:"utf8",
  stdio:["ignore","pipe","pipe"],
 });
 if(result.stdout)process.stdout.write(result.stdout);
 if(result.stderr)process.stderr.write(result.stderr);
 if(result.error)throw result.error;
 if(result.status!==0){
  throw new Error(name+" failed with exit "+String(result.status));
 }
}

function checkGoFmt(){
 process.stdout.write("\n== gofmt ==\n");
 const dirty=[];
 for(const file of goFiles(V2).sort()){
  const result=spawnSync("gofmt",["-d",file],{
   cwd:ROOT,
   env:process.env,
   encoding:"utf8",
   stdio:["ignore","pipe","pipe"],
  });
  if(result.error)throw result.error;
  if(result.status!==0){
   if(result.stderr)process.stderr.write(result.stderr);
   throw new Error("gofmt failed for "+path.relative(ROOT,file));
  }
  if(String(result.stdout||"").trim()){
   dirty.push({
    file:path.relative(ROOT,file),
    diff:result.stdout,
   });
  }
 }
 if(dirty.length){
  for(const row of dirty){
   process.stderr.write("\n"+row.file+"\n"+row.diff);
  }
  throw new Error("V4 Go source is not gofmt-clean");
 }
 process.stdout.write("PASS · "+goFiles(V2).length+" Go files clean\n");
}

function main(){
 const started=new Date().toISOString();
 const steps=[];
 const step=(name,fn)=>{
  const at=Date.now();
  try{
   fn();
   steps.push({name,status:"PASS",durationMs:Date.now()-at});
  }catch(error){
   steps.push({
    name,
    status:"FAIL",
    durationMs:Date.now()-at,
    error:error?.message??String(error),
   });
   throw error;
  }
 };
 try{
  step("gofmt",checkGoFmt);
  step("go-test",()=>run("Go V4 tests","go",["test","./v2/..."],ROOT));
  step("go-vet-native",()=>run(
   "Go V4 native-host vet",
   "go",
   ["vet","./v2/..."],
   ROOT,
  ));
  step("node-semantic",()=>run(
   "Node semantic/model tests",
   npm,
   ["test"],
   V2,
  ));
  step("browser-lived",()=>run(
   "Integrated Chromium lived-shell journey",
   npm,
   ["run","test:lived"],
   V2,
  ));
 }catch(error){
  console.error(JSON.stringify({
   schemaVersion:"vexstream.v4.local-verification/v1",
   status:"FAIL",
   hostPlatform:process.platform,
   hostArch:process.arch,
   startedAt:started,
   finishedAt:new Date().toISOString(),
   steps,
   failure:{message:error?.message??String(error)},
  },null,2));
  process.exitCode=1;
  return;
 }
 console.log(JSON.stringify({
  schemaVersion:"vexstream.v4.local-verification/v1",
  status:"PASS",
  hostPlatform:process.platform,
  hostArch:process.arch,
  startedAt:started,
  finishedAt:new Date().toISOString(),
  steps,
  limits:[
   "native-host vet is not cross-platform release qualification",
   "browser-lived uses deterministic API/media fixtures rather than a live YouTube provider",
   "PASS is not human Windows/macOS product acceptance",
  ],
 },null,2));
}

main();
