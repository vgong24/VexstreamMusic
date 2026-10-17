import assert from "node:assert/strict";
import test from "node:test";

import {
 addSource,
 chooseSourceFolder,
 clearIgnoredSources,
 discoverSources,
 ignoreSourceCandidate,
 loadSourceConfig,
 removeSource,
 unignoreSource
} from "../web/features/sources/client.js";

function okJSON(payload){
 return {ok:true,status:200,json:async()=>payload};
}

test("source config remains only sources plus ignored",async()=>{
 const config=await loadSourceConfig({
  request:async()=>okJSON({
   sources:[" /music ","/music","/archive"],
   ignored:["/skip"]
  })
 });
 assert.deepEqual(config.sources,["/music","/archive"]);
 assert.deepEqual(config.ignored,["/skip"]);
});

test("native folder selection returns evidence only and does not add",async()=>{
 const calls=[];
 const request=async(url,options)=>{
  calls.push([url,options.method]);
  return okJSON({path:" /picked "});
 };
 const picked=await chooseSourceFolder({request});
 assert.equal(picked,"/picked");
 assert.deepEqual(calls,[["/api/source/choose","POST"]]);
});

test("explicit add and remove own source admission mutations",async()=>{
 const seen=[];
 const request=async(url,options)=>{
  seen.push([url,JSON.parse(options.body)]);
  return okJSON({
   ok:true,
   config:{sources:["/music"],ignored:[]},
   libraryRefresh:"CURRENT"
  });
 };
 const added=await addSource(" /music ",{request});
 await removeSource("/music",{request});
 assert.deepEqual(seen,[
  ["/api/source/add",{path:"/music"}],
  ["/api/source/remove",{path:"/music"}]
 ]);
 assert.equal(added.libraryRefresh,"CURRENT");
});

test("discovery is temporary projection and carries no config mutation",async()=>{
 let body;
 const result=await discoverSources({
  deep:true,
  request:async(_url,options)=>{
   body=JSON.parse(options.body);
   return okJSON({
    candidates:[
     {name:"Music",path:"/found",mp3Count:12}
    ],
    scannedDirs:42
   });
  }
 });
 assert.deepEqual(body,{deep:true});
 assert.equal(result.candidates[0].path,"/found");
 assert.equal(result.candidates[0].mp3Count,12);
 assert.equal(result.scannedDirs,42);
});

test("ignore restore and clear are explicit config actions",async()=>{
 const seen=[];
 const request=async(url,options)=>{
  seen.push([url,JSON.parse(options.body)]);
  return okJSON({
   ok:true,
   config:{sources:[],ignored:[]}
  });
 };
 await ignoreSourceCandidate("/candidate",{request});
 await unignoreSource("/candidate",{request});
 await clearIgnoredSources({request});
 assert.deepEqual(seen,[
  ["/api/candidate/ignore",{path:"/candidate"}],
  ["/api/candidate/unignore",{path:"/candidate"}],
  ["/api/ignored/clear",{confirm:"CLEAR_IGNORED_FOLDERS"}]
 ]);
});

test("blank mutation path fails before request",async()=>{
 let calls=0;
 const request=async()=>{calls++;return okJSON({})};
 await assert.rejects(
  ()=>addSource(" ",{request}),
  /requires path/
 );
 assert.equal(calls,0);
});
