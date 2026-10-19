import assert from "node:assert/strict";
import test from "node:test";

import {
  MarqueeMinimumSeconds,
  marqueeDurationSeconds,
} from "../web/foundation/marquee.js";

test("short overflow keeps the familiar minimum reveal time",()=>{
 assert.equal(marqueeDurationSeconds(24),MarqueeMinimumSeconds);
 assert.equal(marqueeDurationSeconds(96),MarqueeMinimumSeconds);
});

test("long titles receive proportionally more time instead of accelerating",()=>{
 const medium=marqueeDurationSeconds(480);
 const long=marqueeDurationSeconds(960);
 assert.equal(medium,25);
 assert.equal(long,50);
 assert.equal(long/medium,2);
});

test("marquee timing rejects invalid speed or travel ownership",()=>{
 assert.throws(
  ()=>marqueeDurationSeconds(100,{pixelsPerSecond:0}),
  /positive/
 );
 assert.throws(
  ()=>marqueeDurationSeconds(100,{travelFraction:1.5}),
  /fraction/
 );
});
