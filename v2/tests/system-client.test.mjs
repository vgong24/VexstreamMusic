import assert from "node:assert/strict";
import test from "node:test";

import {
  exitVexStream,
  loadSessionRetirePlan,
  loadSystemIdentity,
  retireSessions,
} from "../web/features/system/client.js";

function okJSON(payload){
 return {
  ok:true,
  status:200,
  json:async()=>payload
 };
}

test("system identity requires exact version and buildRef",async()=>{
 let seen;
 const identity=await loadSystemIdentity({
  request:async(url,options)=>{
   seen={url,options};
   return okJSON({
    ok:true,
    version:"4.0.0",
    buildRef:"source.bin-abc",
    warnings:["one"]
   });
  }
 });
 assert.equal(seen.url,"/health");
 assert.equal(seen.options.method,"GET");
 assert.equal(identity.version,"4.0.0");
 assert.equal(identity.buildRef,"source.bin-abc");
 assert.deepEqual(identity.warnings,["one"]);
});

test("old-session review preserves eligible and held safety state",async()=>{
 const plan=await loadSessionRetirePlan({
  request:async(url,options)=>{
   assert.equal(url,"/api/system/sessions/retire-plan");
   assert.equal(options.method,"GET");
   return okJSON({
    currentPID:10,
    sessions:[
     {pid:10,version:"4.0.0",current:true,trackCount:3,libraryVerified:true,importsVerified:true},
     {pid:20,version:"3.0.0",current:false,trackCount:3,libraryVerified:true,importsVerified:true}
    ],
    eligible:[{pid:20,version:"3.0.0",eligible:true,reason:"idle"}],
    held:[{pid:30,version:"3.0.0",eligible:false,reason:"active import"}],
    warning:"safe retirement only"
   });
  }
 });
 assert.equal(plan.currentPID,10);
 assert.equal(plan.sessions[0].current,true);
 assert.equal(plan.eligible[0].pid,20);
 assert.equal(plan.eligible[0].eligible,true);
 assert.equal(plan.held[0].reason,"active import");
 assert.equal(plan.warning,"safe retirement only");
});

test("session retirement sends one explicit confirmation and exact selected PIDs",async()=>{
 let body;
 const result=await retireSessions([20,20,30],{
  request:async(url,options)=>{
   assert.equal(url,"/api/system/sessions/retire");
   assert.equal(options.method,"POST");
   body=JSON.parse(options.body);
   return okJSON({
    retired:[20],
    failed:[{pid:30,error:"held for safety"}],
    message:"Retired 1 older VexStream session(s)."
   });
  }
 });
 assert.deepEqual(body,{
  confirm:"RETIRE_OLD_IDLE_SESSIONS",
  pids:[20,30]
 });
 assert.deepEqual(result.retired,[20]);
 assert.deepEqual(result.failed,[{pid:30,error:"held for safety"}]);
});

test("Exit is one explicit confirmed process action",async()=>{
 let body;
 const result=await exitVexStream({
  request:async(url,options)=>{
   assert.equal(url,"/api/system/exit");
   assert.equal(options.method,"POST");
   body=JSON.parse(options.body);
   return okJSON({ok:true,state:"EXITING"});
  }
 });
 assert.deepEqual(body,{confirm:"EXIT_VEXSTREAM"});
 assert.deepEqual(result,{ok:true,state:"EXITING"});
});

test("identity and Exit fail closed on incomplete success payloads",async()=>{
 await assert.rejects(
  ()=>loadSystemIdentity({
   request:async()=>okJSON({ok:true,version:"4.0.0"})
  }),
  /incomplete/
 );
 await assert.rejects(
  ()=>exitVexStream({
   request:async()=>okJSON({ok:true,state:"MAYBE"})
  }),
  /incomplete/
 );
});

test("server error detail survives browser boundary",async()=>{
 await assert.rejects(
  ()=>exitVexStream({
   request:async()=>({
    ok:false,
    status:409,
    json:async()=>({error:"shutdown held"})
   })
  }),
  /shutdown held/
 );
});
