import assert from "node:assert/strict";
import test from "node:test";

import { createTrackSelection } from "../web/features/library/selection.js";

test("selection starts inactive and becomes explicit Experience state",()=>{
 const selection=createTrackSelection();
 assert.deepEqual(selection.currentProjection,{
  active:false,
  count:0,
  selectedRefs:[]
 });
 selection.activate();
 selection.toggle("a",true);
 selection.toggle("b",true);
 selection.toggle("a",false);
 assert.deepEqual(selection.currentProjection,{
  active:true,
  count:1,
  selectedRefs:["b"]
 });
});

test("Select visible deduplicates while preserving existing human selection order",()=>{
 const selection=createTrackSelection({active:true});
 selection.toggle("b",true);
 selection.selectVisible(["a","b","c","a"]);
 assert.deepEqual(selection.currentProjection.selectedRefs,["b","a","c"]);
});

test("catalog reconciliation drops unavailable refs without inventing Library ownership",()=>{
 const selection=createTrackSelection({active:true});
 selection.selectVisible(["a","b","c"]);
 selection.reconcile(["c","a"]);
 assert.deepEqual(selection.currentProjection.selectedRefs,["a","c"]);
 assert.equal(selection.currentProjection.count,2);
});

test("Done deactivates and clears the ephemeral selection",()=>{
 const selection=createTrackSelection({active:true});
 selection.selectVisible(["a","b"]);
 selection.deactivate();
 assert.deepEqual(selection.currentProjection,{
  active:false,
  count:0,
  selectedRefs:[]
 });
});

test("selection changes fail when selection mode is not active",()=>{
 const selection=createTrackSelection();
 assert.throws(()=>selection.toggle("a",true),/not active/);
 assert.throws(()=>selection.selectVisible(["a"]),/not active/);
});
