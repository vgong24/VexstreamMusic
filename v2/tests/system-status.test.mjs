import assert from "node:assert/strict";
import test from "node:test";

import { projectSystemStatus } from "../web/features/system/status.js";

test("healthy System status projects current V4 truths without new ownership",()=>{
 const status=projectSystemStatus({
  identity:{version:"4.0.0",buildRef:"build-a",warnings:[]},
  catalog:[{ref:"a"},{ref:"b"}],
  sourceConfig:{sources:["/music"]},
  sessionRestoration:{kind:"restored"},
  importJobs:{active:[],ready:[],issues:[],recovery:{}},
  trash:{trash:[],pendingPurge:null}
 });

 assert.deepEqual(status,{
  version:"4.0.0",
  buildRef:"build-a",
  warnings:[],
  trackCount:2,
  sourceCount:1,
  sessionKind:"restored",
  sessionHeld:false,
  activeImports:0,
  readyImports:0,
  importIssues:0,
  importRecovery:0,
  trashCount:0,
  pendingPurge:false,
  attentionCount:0,
  state:"healthy"
 });
});

test("warnings and held recovery states become attention without effects",()=>{
 const status=projectSystemStatus({
  identity:{warnings:["cache rebuilt","cache rebuilt","provider warning"]},
  catalog:[{ref:"a"}],
  sourceConfig:{sources:["/music","/archive"]},
  sessionRestoration:{kind:"held-schema"},
  importJobs:{
   active:[{id:"active"}],
   ready:[{id:"ready"}],
   issues:[{code:"import-issue"}],
   recovery:{job:"COMMIT_UNCERTAIN"}
  },
  trash:{trash:[{id:"t"}],pendingPurge:{id:"purge"}}
 });

 assert.deepEqual(status.warnings,["cache rebuilt","provider warning"]);
 assert.equal(status.activeImports,1);
 assert.equal(status.readyImports,1);
 assert.equal(status.importIssues,1);
 assert.equal(status.importRecovery,1);
 assert.equal(status.trashCount,1);
 assert.equal(status.pendingPurge,true);
 assert.equal(status.sessionHeld,true);
 assert.equal(status.attentionCount,7);
 assert.equal(status.state,"attention");
});

test("missing optional inputs remain a healthy empty projection",()=>{
 const status=projectSystemStatus();
 assert.equal(status.trackCount,0);
 assert.equal(status.sourceCount,0);
 assert.equal(status.sessionKind,"fresh");
 assert.equal(status.attentionCount,0);
 assert.equal(status.state,"healthy");
});
