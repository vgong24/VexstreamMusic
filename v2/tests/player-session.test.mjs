import assert from "node:assert/strict";
import test from "node:test";

import { QueueState } from "../web/features/queue/behavior.js";
import {
  PlayerSessionSchema,
  loadPlayerSession,
  reconcilePlayerSession,
  savePlayerSession,
  sessionFromQueueState,
} from "../web/features/player/session.js";

function okJSON(payload) {
  return {
    ok: true,
    status: 200,
    json: async () => payload,
  };
}

test("Queue State persists current identity plus index witness", () => {
  const projected=sessionFromQueueState(
    QueueState.positioned({
      items:["a","b","a"],
      currentIndex:2,
      shuffle:true,
      repeat:true
    })
  );

  assert.equal(projected.schemaVersion,PlayerSessionSchema);
  assert.deepEqual(projected.items,["a","b","a"]);
  assert.equal(projected.currentIndex,2);
  assert.equal(projected.currentRef,"a");
  assert.equal(projected.shuffle,true);
  assert.equal(projected.repeat,true);
});

test("empty Queue persists no invented current identity", () => {
  const projected=sessionFromQueueState(
    QueueState.empty({shuffle:true})
  );
  assert.deepEqual(projected.items,[]);
  assert.equal(projected.currentIndex,-1);
  assert.equal(projected.currentRef,"");
  assert.equal(projected.shuffle,true);
});

test("restoration filters unavailable peers but preserves the exact original current entry", () => {
  const restored=reconcilePlayerSession(
    {
      found:true,
      session:{
        schemaVersion:PlayerSessionSchema,
        items:["missing","a","b","a"],
        currentIndex:3,
        currentRef:"a",
        shuffle:true,
        repeat:false
      }
    },
    ["a","b"]
  );

  assert.equal(restored.kind,"restored");
  assert.deepEqual(restored.queue.items,["a","b","a"]);
  assert.equal(restored.queue.currentIndex,2);
  assert.equal(restored.queue.shuffle,true);
  assert.deepEqual(restored.droppedRefs,["missing"]);
});

test("restoration holds when the exact current entry is unavailable", () => {
  const restored=reconcilePlayerSession(
    {
      found:true,
      session:{
        schemaVersion:PlayerSessionSchema,
        items:["a","missing","b"],
        currentIndex:1,
        currentRef:"missing",
        shuffle:false,
        repeat:true
      }
    },
    ["a","b"]
  );

  assert.equal(restored.kind,"held-current-unavailable");
  assert.equal(restored.queue,null);
  assert.equal(restored.currentRef,"missing");
  assert.deepEqual(restored.droppedRefs,["missing"]);
});

test("missing durable session stays fresh instead of seeding Queue", () => {
  const restored=reconcilePlayerSession(
    {
      found:false,
      session:{
        schemaVersion:PlayerSessionSchema,
        items:[],
        currentIndex:-1,
        currentRef:"",
        shuffle:false,
        repeat:false
      }
    },
    ["a"]
  );
  assert.equal(restored.kind,"fresh");
  assert.equal(restored.queue,null);
});

test("load and save use one V4-local server boundary", async () => {
  const seen=[];
  const request=async(url,options)=>{
    seen.push([url,options.method,options.body??null]);
    if(options.method==="GET"){
      return okJSON({
        found:true,
        session:{
          schemaVersion:PlayerSessionSchema,
          items:["a"],
          currentIndex:0,
          currentRef:"a",
          shuffle:false,
          repeat:false
        }
      });
    }
    return okJSON({
      saved:true,
      session:{
        schemaVersion:PlayerSessionSchema,
        ...JSON.parse(options.body),
        updatedAt:"now"
      }
    });
  };

  const loaded=await loadPlayerSession({request});
  assert.equal(loaded.session.currentRef,"a");

  const saved=await savePlayerSession(
    QueueState.positioned({
      items:["a","b"],
      currentIndex:1,
      repeat:true
    }),
    {request}
  );
  assert.equal(saved.currentRef,"b");
  assert.equal(saved.repeat,true);

  assert.equal(seen[0][0],"/api/player/session");
  assert.equal(seen[0][1],"GET");
  assert.equal(seen[1][0],"/api/player/session");
  assert.equal(seen[1][1],"PUT");
  assert.deepEqual(JSON.parse(seen[1][2]),{
    items:["a","b"],
    currentIndex:1,
    currentRef:"b",
    shuffle:false,
    repeat:true
  });
});

test("unsupported session response fails rather than guessing", async () => {
  await assert.rejects(
    () => loadPlayerSession({
      request:async()=>okJSON({
        found:true,
        session:{
          schemaVersion:"vexstream.desktop-player-session/v99",
          items:[]
        }
      })
    }),
    /schema is unsupported/
  );
});
