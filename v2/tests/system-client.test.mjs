import assert from "node:assert/strict";
import test from "node:test";

import {
  exitVexStream,
  loadSystemIdentity,
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
