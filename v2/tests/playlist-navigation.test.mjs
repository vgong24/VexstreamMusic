import assert from "node:assert/strict";
import test from "node:test";

import { createPlaylistNavigation } from "../web/features/playlists/navigation.js";

test("Playlists starts at the browse terrain",()=>{
 const navigation=createPlaylistNavigation();
 assert.deepEqual(navigation.currentProjection,{openPlaylistID:""});
});

test("opening and closing one playlist is Experience location only",()=>{
 const navigation=createPlaylistNavigation();
 const observed=[];
 navigation.subscribe(value=>observed.push(value.openPlaylistID));

 navigation.open("road-trip");
 assert.deepEqual(navigation.currentProjection,{openPlaylistID:"road-trip"});
 navigation.close();

 assert.deepEqual(observed,["road-trip",""]);
});

test("reselecting current playlist is stable",()=>{
 const navigation=createPlaylistNavigation({initialPlaylistID:"focus"});
 const observed=[];
 navigation.subscribe(value=>observed.push(value.openPlaylistID));
 assert.deepEqual(navigation.open("focus"),{openPlaylistID:"focus"});
 assert.deepEqual(observed,[]);
});

test("reconciliation closes detail when durable playlist disappears",()=>{
 const navigation=createPlaylistNavigation({initialPlaylistID:"gone"});
 navigation.reconcile(["other"]);
 assert.deepEqual(navigation.currentProjection,{openPlaylistID:""});
});

test("blank playlist identity fails instead of inventing terrain",()=>{
 const navigation=createPlaylistNavigation();
 assert.throws(()=>navigation.open(" "),/requires id/);
});
