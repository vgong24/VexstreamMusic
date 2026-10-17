import assert from "node:assert/strict";
import test from "node:test";

import {
  decidePhonePairing,
  forgetPairedPhone,
  loadPhonePairing,
  projectPhonePairing,
  startPhonePairing,
} from "../web/features/phone/client.js";

function okJSON(payload){
  return {ok:true,status:200,json:async()=>payload};
}

test("phone pairing status projects the established V3 contract",async()=>{
  let seen;
  const state=await loadPhonePairing({
    request:async(url,options)=>{
      seen={url,options};
      return okJSON({
        running:true,
        displayName:"VexStream on Mac",
        pairingCode:"123456",
        pairingWindowOpen:true,
        pairingWindowUntil:"2026-10-17T12:00:00Z",
        pending:[{requestRef:"pair.one",displayName:"Samsung",requestedAt:"2026-10-17T11:59:00Z"}],
        paired:[{deviceRef:"vexlife.phone",displayName:"Victor phone",approvedAt:"2026-10-16T00:00:00Z"}],
      });
    },
  });
  assert.equal(seen.url,"/api/mobile/pairing");
  assert.equal(seen.options.method,"GET");
  assert.equal(state.displayName,"VexStream on Mac");
  assert.equal(state.pairingCode,"123456");
  assert.equal(state.pending[0].requestRef,"pair.one");
  assert.equal(state.paired[0].deviceRef,"vexlife.phone");
  assert.equal(Object.isFrozen(state),true);
});

test("start, approve/deny, and forget preserve the established localhost admin API",async()=>{
  const seen=[];
  const request=async(url,options)=>{
    seen.push({url,method:options.method,body:JSON.parse(options.body||"{}")});
    if(url.endsWith("/start"))return okJSON({ok:true,pairingCode:"123456",expiresAt:"2026-10-17T12:00:00Z"});
    if(url.endsWith("/decision")){
      const body=JSON.parse(options.body);
      return okJSON({ok:true,state:body.decision==="APPROVE"?"APPROVED":"DENIED"});
    }
    return okJSON({ok:true});
  };
  await startPhonePairing({request});
  await decidePhonePairing("pair.one","APPROVE",{request});
  await decidePhonePairing("pair.two","DENY",{request});
  await forgetPairedPhone("vexlife.phone",{request});
  assert.deepEqual(seen,[
    {url:"/api/mobile/pairing/start",method:"POST",body:{}},
    {url:"/api/mobile/pairing/decision",method:"POST",body:{requestRef:"pair.one",decision:"APPROVE"}},
    {url:"/api/mobile/pairing/decision",method:"POST",body:{requestRef:"pair.two",decision:"DENY"}},
    {url:"/api/mobile/pairing/revoke",method:"POST",body:{deviceRef:"vexlife.phone"}},
  ]);
});

test("phone client fails closed on malformed state and unsupported decisions",async()=>{
  assert.throws(
    ()=>projectPhonePairing({displayName:"VexStream",pairingCode:"12"}),
    /incomplete/,
  );
  await assert.rejects(
    ()=>decidePhonePairing("pair.one","MAYBE",{request:async()=>okJSON({})}),
    /APPROVE or DENY/,
  );
});

test("server error detail survives the phone client boundary",async()=>{
  await assert.rejects(
    ()=>loadPhonePairing({
      request:async()=>({
        ok:false,
        status:409,
        json:async()=>({error:"pairing listener unavailable"}),
      }),
    }),
    /pairing listener unavailable/,
  );
});
