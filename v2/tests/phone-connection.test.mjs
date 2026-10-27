import assert from "node:assert/strict";
import test from "node:test";

import {
  PhoneConnectionAction,
  createPhoneConnection,
} from "../web/features/phone/connection.js";

function state({
  open=false,
  pending=[],
  paired=[],
}={}){
  return Object.freeze({
    running:open,
    displayName:"VexStream on Mac",
    pairingCode:"123456",
    pairingWindowOpen:open,
    pairingWindowUntil:"2026-10-17T12:00:00Z",
    pending:Object.freeze(pending),
    paired:Object.freeze(paired),
  });
}

function harness({remembered=false,confirmForget=()=>true}={}){
  const operations=[];
  let current=state({
    paired:remembered
      ?[Object.freeze({deviceRef:"phone.one",displayName:"Victor phone",approvedAt:"now"})]
      :[],
  });
  let scheduled=null;
  const connection=createPhoneConnection({
    loadPairing:async()=>{operations.push("load");return current;},
    startPairing:async()=>{operations.push("start");current=state({open:true,paired:current.paired});},
    decidePairing:async(ref,decision)=>{
      operations.push(`decision:${ref}:${decision}`);
      current=state({
        open:true,
        paired:decision==="APPROVE"
          ?[Object.freeze({deviceRef:"phone.one",displayName:"Victor phone",approvedAt:"now"})]
          :[],
      });
    },
    forgetPairedPhone:async ref=>{
      operations.push(`forget:${ref}`);
      current=state();
    },
    confirmForget,
    schedule:fn=>{scheduled=fn;return 1;},
    cancel:()=>{scheduled=null;},
    pollMs:1000,
  });
  return {
    connection,
    operations,
    get scheduled(){return scheduled;},
    setCurrent(value){current=value;},
  };
}

test("open follows the human walk: observe then advertise only when no phone is remembered",async()=>{
  const h=harness();
  await h.connection.open();
  assert.deepEqual(h.operations,["load","start","load"]);
  assert.equal(h.connection.currentProjection.phase,"ready");
  assert.equal(h.connection.currentProjection.pairing.pairingWindowOpen,true);
  assert.equal(typeof h.scheduled,"function");
});

test("remembered phone opens management without unnecessarily reopening pairing",async()=>{
  const h=harness({remembered:true});
  await h.connection.open();
  assert.deepEqual(h.operations,["load"]);
  assert.equal(h.connection.currentProjection.pairing.paired.length,1);
  assert.equal(h.connection.currentProjection.pairing.pairingWindowOpen,false);
});

test("pending phone approval routes through one named connection action then refreshes",async()=>{
  const h=harness();
  await h.connection.open();
  h.setCurrent(state({
    open:true,
    pending:[Object.freeze({requestRef:"pair.one",displayName:"Samsung",requestedAt:"now"})],
  }));
  await h.connection.refresh();
  await h.connection.activate({
    kind:PhoneConnectionAction.Approve,
    requestRef:"pair.one",
  });
  assert.equal(h.operations.includes("decision:pair.one:APPROVE"),true);
  assert.equal(h.connection.currentProjection.pairing.paired.length,1);
});

test("Forget preserves the V3 human confirmation boundary",async()=>{
  let confirmed=false;
  const h=harness({
    remembered:true,
    confirmForget:()=>confirmed,
  });
  await h.connection.open();
  await h.connection.activate({
    kind:PhoneConnectionAction.Forget,
    deviceRef:"phone.one",
  });
  assert.equal(h.operations.some(value=>value.startsWith("forget:")),false);
  confirmed=true;
  await h.connection.activate({
    kind:PhoneConnectionAction.Forget,
    deviceRef:"phone.one",
  });
  assert.equal(h.operations.includes("forget:phone.one"),true);
});

test("close owns transient presentation lifetime and cancels polling",async()=>{
  const h=harness();
  await h.connection.open();
  assert.equal(typeof h.scheduled,"function");
  h.connection.close();
  assert.equal(h.connection.currentProjection.open,false);
  assert.equal(h.connection.currentProjection.phase,"closed");
  assert.equal(h.scheduled,null);
});

test("connection failure is projected as feature state instead of escaping into callback choreography",async()=>{
  const connection=createPhoneConnection({
    loadPairing:async()=>{throw new Error("discovery unavailable");},
    startPairing:async()=>{},
    decidePairing:async()=>{},
    forgetPairedPhone:async()=>{},
    schedule:()=>1,
    cancel:()=>{},
  });
  await connection.open();
  assert.equal(connection.currentProjection.phase,"failed");
  assert.match(connection.currentProjection.error,/discovery unavailable/);
});
