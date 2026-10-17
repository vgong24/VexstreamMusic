import assert from "node:assert/strict";
import test from "node:test";

import { presentPhoneConnection } from "../web/features/phone/presentation.js";

const ready=Object.freeze({
  open:true,
  phase:"ready",
  error:"",
  pairing:Object.freeze({
    running:true,
    displayName:"VexStream on Mac",
    pairingCode:"123456",
    pairingWindowOpen:true,
    pairingWindowUntil:"later",
    pending:Object.freeze([
      Object.freeze({requestRef:"pair.one",displayName:"Samsung",requestedAt:"now"}),
    ]),
    paired:Object.freeze([
      Object.freeze({deviceRef:"phone.one",displayName:"Victor phone",approvedAt:"yesterday"}),
    ]),
  }),
});

test("presentation projects the established V3 pairing modal grammar",()=>{
  const view=presentPhoneConnection(ready);
  assert.equal(view.title,"Connect your VexLife phone");
  assert.match(view.bodyHTML,/Ready to pair for 5 minutes/);
  assert.match(view.bodyHTML,/123 456/);
  assert.match(view.bodyHTML,/Waiting for your approval/);
  assert.match(view.bodyHTML,/data-phone-action="approve"/);
  assert.match(view.bodyHTML,/Remembered phones/);
  assert.match(view.bodyHTML,/data-phone-action="forget"/);
  assert.match(view.actionsHTML,/data-modal-close/);
  assert.equal(view.bodyHTML.includes("phone-dialog"),false);
});

test("loading and failure remain inside the same shared modal rather than forming new surfaces",()=>{
  assert.match(
    presentPhoneConnection({open:true,phase:"loading",pairing:null,error:""}).bodyHTML,
    /Checking phone connection/,
  );
  const failed=presentPhoneConnection({
    open:true,
    phase:"failed",
    pairing:null,
    error:"<bad>",
  });
  assert.match(failed.bodyHTML,/&lt;bad&gt;/);
  assert.equal(failed.bodyHTML.includes("phone-dialog"),false);
});

test("closed connection has no presentation",()=>{
  assert.equal(
    presentPhoneConnection({open:false,phase:"closed",pairing:null,error:""}),
    null,
  );
});
