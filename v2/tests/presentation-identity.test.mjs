// [VXG RealForever]
import assert from "node:assert/strict";
import test from "node:test";

import {
 appendVexPresentationPath,
 dynamicVexPresentationRef,
 VexPresentationContinuity,
 VEX_PRESENTATION_INSTANCE_ENCODING,
} from "../web/foundation/identity.js";

test("nested presentation identity follows the same normalized parent-child law",()=>{
 assert.equal(
  appendVexPresentationPath("element.gallery.page.content.elements.sample","title"),
  "element.gallery.page.content.elements.sample.title"
 );
 assert.equal(
  appendVexPresentationPath("element.gallery.body"," preview/title "),
  "element.gallery.body.preview.title"
 );
});

test("runtime presentation identity is deterministic across reorder and text changes",()=>{
 const first=dynamicVexPresentationRef(
  "element.vexstream.playlists.card",
  ["playlist-42"]
 );
 const again=dynamicVexPresentationRef(
  "element.vexstream.playlists.card",
  ["playlist-42"]
 );
 assert.equal(first,again);
 assert.equal(
  first,
  "element.vexstream.playlists.card.dynamic.utf8hex.706c61796c6973742d3432"
 );
});

test("presentation scope can distinguish one domain object without changing domain identity",()=>{
 const songs=dynamicVexPresentationRef(
  "element.vexstream.track",
  ["songs","track-7"]
 );
 const search=dynamicVexPresentationRef(
  "element.vexstream.track",
  ["search","track-7"]
 );
 assert.notEqual(songs,search);
});

test("continuity vocabulary keeps stable instances distinct from current occurrences",()=>{
 assert.deepEqual(
  Object.values(VexPresentationContinuity),
  ["STABLE_INSTANCE","CURRENT_OCCURRENCE"]
 );
 assert.equal(VEX_PRESENTATION_INSTANCE_ENCODING,"UTF8_HEX_NUL_PARTS");
});
