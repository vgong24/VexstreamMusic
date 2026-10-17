import assert from "node:assert/strict";
import test from "node:test";

import {
  restorePlaybackWindow,
  restoreTrackDisplay,
  updatePlaybackWindow,
  updateTrackDisplay,
  updateTrackDisplays,
} from "../web/features/library/admin.js";

function okJSON(payload={ok:true}){
 return {ok:true,status:200,json:async()=>payload};
}

test("display edit sends only VexStream overlay fields",async()=>{
 let body;
 await updateTrackDisplay(
  " track-a ",
  {title:"New",artist:"",album:"Album"},
  {request:async(_url,options)=>{
   body=JSON.parse(options.body);
   return okJSON();
  }}
 );
 assert.deepEqual(body,{
  id:"track-a",
  title:"New",
  artist:"",
  album:"Album"
 });
});

test("shared display edit uses one multi-track overlay mutation and excludes Title",async()=>{
 let body;
 await updateTrackDisplays(
  [" a ","b","a"],
  {artist:"Shared Artist",genre:"",title:"Do not bulk-title"},
  {request:async(_url,options)=>{body=JSON.parse(options.body);return okJSON({ok:true,updated:2});}}
 );
 assert.deepEqual(body,{ids:["a","b"],artist:"Shared Artist",genre:""});
});

test("display restore is explicit",async()=>{
 let body;
 await restoreTrackDisplay("track-a",{
  request:async(_url,options)=>{
   body=JSON.parse(options.body);
   return okJSON();
  }
 });
 assert.deepEqual(body,{id:"track-a",restore:true});
});

test("playback window update and restore stay separate from metadata",async()=>{
 const seen=[];
 const request=async(url,options)=>{
  seen.push([url,JSON.parse(options.body)]);
  return okJSON();
 };
 await updatePlaybackWindow("a",{start:12.5,end:60},{request});
 await restorePlaybackWindow("a",{request});
 assert.deepEqual(seen,[
  ["/api/library/playback-window",{id:"a",start:12.5,end:60}],
  ["/api/library/playback-window",{id:"a",restore:true}]
 ]);
});

test("invalid edits fail before request",async()=>{
 let calls=0;
 const request=async()=>{calls++;return okJSON()};
 await assert.rejects(
  ()=>updateTrackDisplay("",{title:"x"},{request}),
  /track ref/
 );
 await assert.rejects(
  ()=>updateTrackDisplay("a",{}, {request}),
  /at least one/
 );
 await assert.rejects(
  ()=>updatePlaybackWindow("a",{start:-1},{request}),
  /non-negative/
 );
 await assert.rejects(()=>updateTrackDisplays([], {artist:"x"},{request}),/track refs/);
 await assert.rejects(()=>updateTrackDisplays(["a"], {title:"not admitted"},{request}),/shared field/);
 assert.equal(calls,0);
});
