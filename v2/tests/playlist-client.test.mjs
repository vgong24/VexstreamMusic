import assert from "node:assert/strict";
import test from "node:test";

import {
  addTracksToPlaylist,
  createPlaylist,
  deletePlaylist,
  loadPlaylists,
  projectPlaylists,
  removeTracksFromPlaylist,
  renamePlaylist,
  reorderPlaylist,
  resolvePlaylistAvailability,
} from "../web/features/playlists/client.js";

test("playlist projection preserves durable order including unresolved refs", () => {
  const [playlist] = projectPlaylists({
    playlists: [{
      id: "road-trip",
      name: "Road Trip",
      trackIds: ["track-a", "missing", "track-b"],
      createdAt: "c",
      updatedAt: "u",
    }],
  });

  assert.deepEqual(playlist.trackRefs, ["track-a", "missing", "track-b"]);
  assert.equal(playlist.name, "Road Trip");
});

test("availability derives from current Library without rewriting Playlist truth", () => {
  const [playlist] = projectPlaylists({
    playlists: [{
      id: "road-trip",
      name: "Road Trip",
      trackIds: ["track-a", "missing", "track-b"],
    }],
  });
  const catalog = [
    Object.freeze({ ref: "track-a", title: "A" }),
    Object.freeze({ ref: "track-b", title: "B" }),
  ];

  const resolved = resolvePlaylistAvailability(playlist, catalog);
  assert.deepEqual(resolved.playableRefs, ["track-a", "track-b"]);
  assert.deepEqual(resolved.missingRefs, ["missing"]);
  assert.deepEqual(playlist.trackRefs, ["track-a", "missing", "track-b"]);
  assert.equal(resolved.entries[1].available, false);
  assert.equal(resolved.entries[1].track, null);
});

test("playlist identity, name and refs are required", () => {
  assert.throws(() => projectPlaylists({ playlists: [{}] }), /missing id/);
  assert.throws(
    () => projectPlaylists({ playlists: [{ id: "p", name: " ", trackIds: [] }] }),
    /missing name/,
  );
  assert.throws(
    () => projectPlaylists({ playlists: [{ id: "p", name: "Mix" }] }),
    /trackIds/,
  );
  assert.throws(
    () => projectPlaylists({ playlists: [{ id: "p", name: "Mix", trackIds: [""] }] }),
    /is empty/,
  );
});

test("load uses read-only playlist route", async () => {
  let seen;
  const rows = await loadPlaylists({
    request: async (url, options) => {
      seen = { url, options };
      return {
        ok: true,
        status: 200,
        json: async () => ({
          playlists: [{ id: "p", name: "Mix", trackIds: ["a"] }],
        }),
      };
    },
  });

  assert.equal(seen.url, "/api/playlists");
  assert.equal(seen.options.method, "GET");
  assert.equal(seen.options.headers.Accept, "application/json");
  assert.equal(rows[0].id, "p");
});

test("failed playlist HTTP response remains a load failure", async () => {
  await assert.rejects(
    () => loadPlaylists({ request: async () => ({ ok: false, status: 503 }) }),
    /503/,
  );
});


test("Playlist mutations preserve durable ordered refs and explicit delete",async()=>{
 const seen=[];
 const request=async(url,options)=>{
  const body=JSON.parse(options.body);
  seen.push([url,body]);
  if(url==="/api/playlists/create"){
   return {
    ok:true,status:200,
    json:async()=>({
     ok:true,
     playlist:{
      id:"p",
      name:body.name,
      trackIds:body.trackIds,
      createdAt:"now",
      updatedAt:"now"
     }
    })
   };
  }
  if(url==="/api/playlists/add"){
   return {
    ok:true,status:200,
    json:async()=>({
     ok:true,
     added:body.trackIds.length,
     playlist:{
      id:"p",
      name:"Mix",
      trackIds:["a",...body.trackIds],
      createdAt:"now",
      updatedAt:"later"
     }
    })
   };
  }
  if(url==="/api/playlists/remove"||url==="/api/playlists/rename"){
   return {
    ok:true,status:200,
    json:async()=>({
     ok:true,
     playlist:{
      id:"p",
      name:body.name??"Mix",
      trackIds:url.endsWith("/remove")?["a"]:["a","b"],
      createdAt:"now",
      updatedAt:"later"
     }
    })
   };
  }
  return {ok:true,status:200,json:async()=>({ok:true})};
 };

 const created=await createPlaylist(" Mix ",["a","b"],{request});
 assert.equal(created.name,"Mix");
 assert.deepEqual(created.trackRefs,["a","b"]);

 const added=await addTracksToPlaylist("p",["b","c"],{request});
 assert.equal(added.added,2);
 assert.deepEqual(added.playlist.trackRefs,["a","b","c"]);

 await reorderPlaylist("p",["c","a","b"],{request});
 const removed=await removeTracksFromPlaylist("p",["b"],{request});
 assert.deepEqual(removed.trackRefs,["a"]);
 const renamed=await renamePlaylist("p","Road",{request});
 assert.equal(renamed.name,"Road");
 await deletePlaylist("p",{request});

 assert.deepEqual(seen.map(row=>row[0]),[
  "/api/playlists/create",
  "/api/playlists/add",
  "/api/playlists/order",
  "/api/playlists/remove",
  "/api/playlists/rename",
  "/api/playlists/delete"
 ]);
 assert.deepEqual(seen.at(-1)[1],{
  id:"p",
  confirm:"DELETE_PLAYLIST_ONLY"
 });
});

test("Playlist mutation inputs fail closed before request",async()=>{
 let calls=0;
 const request=async()=>{calls++;return {ok:true,json:async()=>({})}};
 await assert.rejects(
  ()=>addTracksToPlaylist("",["a"],{request}),
  /playlist id/
 );
 await assert.rejects(
  ()=>addTracksToPlaylist("p",[],{request}),
  /track refs/
 );
 assert.equal(calls,0);
});
