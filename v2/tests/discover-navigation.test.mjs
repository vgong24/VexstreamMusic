import assert from "node:assert/strict";
import test from "node:test";

import {
 DiscoverMode,
 createDiscoverNavigation
} from "../web/features/discover/navigation.js";

test("Discover starts on the recognizable This PC route",()=>{
 const navigation=createDiscoverNavigation();
 assert.deepEqual(navigation.currentProjection,{mode:DiscoverMode.PC});
});

test("PC YouTube and Explore are human locations not feature truth owners",()=>{
 const navigation=createDiscoverNavigation();
 const observed=[];
 navigation.subscribe(projection=>observed.push(projection.mode));

 navigation.select(DiscoverMode.YOUTUBE);
 navigation.select(DiscoverMode.EXPLORE);
 navigation.select(DiscoverMode.PC);

 assert.deepEqual(observed,["youtube","explore","pc"]);
});

test("reselecting current Discover route is stable",()=>{
 const navigation=createDiscoverNavigation({initialMode:DiscoverMode.YOUTUBE});
 const observed=[];
 navigation.subscribe(projection=>observed.push(projection.mode));
 assert.deepEqual(navigation.select(DiscoverMode.YOUTUBE),{mode:"youtube"});
 assert.deepEqual(observed,[]);
});

test("unknown Discover route fails instead of inventing product navigation",()=>{
 const navigation=createDiscoverNavigation();
 assert.throws(()=>navigation.select("radio"),/Unknown VexStream Discover mode/);
});
