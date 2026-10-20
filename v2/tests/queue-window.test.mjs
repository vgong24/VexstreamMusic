import assert from "node:assert/strict";
import test from "node:test";
import {
 QueueWindowPolicy,
 queueWindowAroundIndex,
 queueWindowForScroll
} from "../web/features/queue/window.js";

test("Queue window materializes at most one hundred rows around current",()=>{
 const projection=queueWindowAroundIndex(738,506);
 assert.equal(projection.end-projection.start,100);
 assert.equal(projection.start,481);
 assert.equal(projection.end,581);
 assert.ok(506>=projection.start&&506<projection.end);
});

test("Queue window advances when first visible row crosses seventy-five percent",()=>{
 const projection=queueWindowForScroll({
  length:738,
  start:0,
  firstVisible:75
 });
 assert.equal(projection.changed,true);
 assert.equal(projection.start,50);
 assert.equal(projection.end,150);
});

test("Queue window stays put below the forward threshold",()=>{
 const projection=queueWindowForScroll({
  length:738,
  start:0,
  firstVisible:60
 });
 assert.equal(projection.changed,false);
 assert.equal(projection.start,0);
 assert.equal(projection.end,100);
});

test("Queue window can retreat without changing Queue truth",()=>{
 const refs=Array.from({length:738},(_,index)=>`track-${index}`);
 const projection=queueWindowForScroll({
  length:refs.length,
  start:300,
  firstVisible:310
 });
 assert.equal(projection.changed,true);
 assert.equal(projection.start,285);
 assert.deepEqual(refs.slice(0,3),["track-0","track-1","track-2"]);
 assert.equal(refs.length,738);
 assert.equal(QueueWindowPolicy.size,100);
});
