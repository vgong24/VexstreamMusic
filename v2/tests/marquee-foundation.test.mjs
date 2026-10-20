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
 assert.ok(Math.abs(medium-25)<1e-9);
 assert.ok(Math.abs(long-50)<1e-9);
 assert.ok(Math.abs(long/medium-2)<1e-12);
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
