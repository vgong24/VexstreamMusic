import assert from "node:assert/strict";
import test from "node:test";

import {
 groupLibrary,
 libraryGenres,
 projectLibraryHome,
 projectLibraryView
} from "../web/features/library/projection.js";

const catalog=Object.freeze([
 Object.freeze({
  ref:"c",
  title:"Night Drive",
  artist:"Beta",
  album:"Road",
  genre:"Synthwave",
  year:"2025",
  channel:"Beta Channel",
  durationSeconds:200,
  addedAt:"2026-10-12T00:00:00Z",
  tags:Object.freeze(["neon"])
 }),
 Object.freeze({
  ref:"a",
  title:"Blue Morning",
  artist:"Alpha",
  album:"Sky",
  genre:"Ambient",
  year:"2024",
  durationSeconds:100,
  addedAt:"2026-10-14T00:00:00Z",
  categories:Object.freeze(["calm"])
 }),
 Object.freeze({
  ref:"b",
  title:"Amber",
  artist:"Alpha",
  album:"Sky",
  genre:"Ambient",
  year:"2023",
  durationSeconds:300,
  addedAt:"2026-10-13T00:00:00Z"
 })
]);

test("search spans human/provider metadata without mutating catalog",()=>{
 const original=catalog.map(item=>item.ref);
 assert.deepEqual(
  projectLibraryView(catalog,{query:"neon"}).map(item=>item.ref),
  ["c"]
 );
 assert.deepEqual(
  projectLibraryView(catalog,{query:"calm"}).map(item=>item.ref),
  ["a"]
 );
 assert.deepEqual(
  projectLibraryView(catalog,{query:"beta channel"}).map(item=>item.ref),
  ["c"]
 );
 assert.deepEqual(catalog.map(item=>item.ref),original);
});

test("genre filter and sort are current view projections",()=>{
 assert.deepEqual(
  projectLibraryView(catalog,{genre:"Ambient",sort:"title"})
   .map(item=>item.ref),
  ["b","a"]
 );
 assert.deepEqual(
  projectLibraryView(catalog,{sort:"recent"}).map(item=>item.ref),
  ["a","b","c"]
 );
 assert.deepEqual(
  projectLibraryView(catalog,{sort:"duration-desc"}).map(item=>item.ref),
  ["b","c","a"]
 );
});

test("artist and album sorting use title as stable tie break",()=>{
 assert.deepEqual(
  projectLibraryView(catalog,{sort:"artist"}).map(item=>item.ref),
  ["b","a","c"]
 );
 assert.deepEqual(
  projectLibraryView(catalog,{sort:"album"}).map(item=>item.ref),
  ["c","b","a"]
 );
});

test("genre projection deduplicates case-insensitively",()=>{
 const rows=[
  ...catalog,
  {...catalog[0],ref:"d",genre:"synthWAVE"}
 ];
 assert.deepEqual(libraryGenres(rows),["Ambient","Synthwave"]);
});

test("grouping keeps ordered refs and never becomes Library ownership",()=>{
 const albums=groupLibrary(catalog,"album");
 const sky=albums.find(group=>group.name==="Sky");
 assert.deepEqual(sky.refs,["a","b"]);
 assert.equal(sky.tracks[0].ref,"a");

 const artists=groupLibrary(catalog,"artist");
 const alpha=artists.find(group=>group.name==="Alpha");
 assert.deepEqual(alpha.refs,["a","b"]);
});

test("unknown grouping fails instead of inventing semantics",()=>{
 assert.throws(()=>groupLibrary(catalog,"genre"),/album or artist/);
});


test("Library home is a read-only projection of current catalog truth",()=>{
 const home=projectLibraryHome(catalog,{recentLimit:2});
 assert.equal(home.songCount,3);
 assert.equal(home.albumCount,2);
 assert.equal(home.artistCount,2);
 assert.deepEqual(home.recent.map(item=>item.ref),["a","b"]);
 assert.equal(Object.isFrozen(home),true);
 assert.equal(Object.isFrozen(home.recent),true);
});
