import assert from "node:assert/strict";
import test from "node:test";

import {
  loadTrash,
  moveTracksToTrash,
  purgeTrashItems,
  recoverTrashPurge,
  restoreTrashItems,
} from "../web/features/library/trash.js";

function okJSON(payload){
 return {ok:true,status:200,json:async()=>payload};
}

test("Trash snapshot hides raw custody paths from browser projection",async()=>{
 const snapshot=await loadTrash({
  request:async()=>okJSON({
   trash:[{
    id:"trash-a",
    track:{id:"a",title:"Song"},
    originalPath:"C:/music/a.mp3",
    trashPath:"C:/trash/a.mp3",
    trashedAt:"now"
   }],
   deletedHistory:[{
    id:"deleted-a",
    track:{id:"x",title:"Gone"},
    originalPath:"C:/music/x.mp3",
    deletedAt:"later"
   }]
  })
 });
 assert.deepEqual(snapshot.trash[0],{
  id:"trash-a",
  track:{id:"a",title:"Song"},
  trashedAt:"now"
 });
 assert.equal("trashPath" in snapshot.trash[0],false);
});

test("Trash actions carry explicit destructive confirmations",async()=>{
 const seen=[];
 const request=async(url,options)=>{
  seen.push([url,JSON.parse(options.body)]);
  return okJSON({ok:true});
 };
 await moveTracksToTrash(["a","a","b"],{request});
 await restoreTrashItems(["trash-a"],{request});
 await purgeTrashItems(["trash-a"],{
  preserveMetadata:true,
  request
 });
 await recoverTrashPurge({request});
 assert.deepEqual(seen,[
  ["/api/library/trash",{
   ids:["a","b"],
   confirm:"MOVE_TO_VEXSTREAM_TRASH"
  }],
  ["/api/library/trash/restore",{ids:["trash-a"]}],
  ["/api/library/trash/purge",{
   ids:["trash-a"],
   confirm:"PERMANENTLY_DELETE_TRASH",
   preserveMetadata:true
  }],
  ["/api/library/trash/purge/recover",{
   confirm:"RECOVER_ADMITTED_TRASH_PURGE"
  }]
 ]);
});

test("Trash refuses empty selection before request",async()=>{
 let calls=0;
 await assert.rejects(
  ()=>moveTracksToTrash([],{
   request:async()=>{calls++;return okJSON({})}
  }),
  /requires track refs/
 );
 assert.equal(calls,0);
});
